package xrs

import "errors"

func checkEnc(d, p int, vs [][]byte) (size int, err error) {
	total := len(vs)
	if d+p != total {
		err = errors.New("xrs.checkEnc: vects not match rs args")
		return
	}
	size = len(vs[0])
	if size == 0 {
		err = errors.New("xrs.checkEnc: vects size = 0")
		return
	}
	if !((size & 1) == 0) { // it isn't even
		err = errors.New("xrs.checkEnc: vects size is odd")
		return
	}
	for i := 1; i < total; i++ {
		if len(vs[i]) != size {
			err = errors.New("xrs.checkEnc: vects size mismatch")
			return
		}
	}
	return
}

func (x *xrs) Encode(vects [][]byte) (err error) {
	return x.enc(vects, false)
}

func (x *xrs) enc(vects [][]byte, rsOnly bool) (err error) {
	d, p := x.Data, x.Parity
	size, err := checkEnc(d, p, vects)
	if err != nil {
		return
	}
	if (x.ext != none) && (size >= 16) { // 16bytes is the smallest SIMD register size
		return x.encSIMD(vects, rsOnly)
	} else {
		return x.encBase(vects, rsOnly)
	}
}

// Size of sub-vector, fit for L1 Data Cache (32KB)
const halfL1 = 16 * 1024

// Split vector (divisible by 16)
// n >= 16
func getDo(n int) int {
	if n < halfL1 {
		return (n >> 4) << 4
	}
	return halfL1
}

func xorSIMD(dst []byte, src [][]byte, ext int) {
	if ext == avx2 {
		xorAVX2(dst, src)
	} else {
		xorSSE2(dst, src)
	}
}

func encXOR(dst []byte, src [][]byte, ext int) {
	if ext != none {
		xorSIMD(dst, src, ext)
	} else {
		xorBase(dst, src)
	}
}

// Encode SIMD combines ssse3 avx avx2, if will add some branches, but codes get clean
// we have to make a balance between performance & clean codes
func (x *xrs) encSIMD(vects [][]byte, rsOnly bool) (err error) {
	// step1: reedsolomon encM
	dv, pv := vects[:x.Data], vects[x.Data:]
	size := len(vects[0])
	start, end := 0, 0
	do := getDo(size)
	for start < size {
		end = start + do
		if end <= size {
			x.matrixMul(start, end, dv, pv)
			start = end
		} else {
			x.matrixMulRemain(start, size, dv, pv) // calculate left data (< do)
			start = size
		}
	}
	if rsOnly == true {
		return
	}
	// step2: xor a & f(b)
	for i, a := range x.xm {
		v := make([][]byte, len(a)+1)
		v[0] = vects[i][size : size*2]
		for i, k := range a {
			v[i+1] = vects[k][0:size]
		}
		xorSIMD(vects[i][size:size*2], v, x.ext)
	}
	return
}

//go:noescape
func mulVectAVX2(tbl, d, p []byte)

//go:noescape
func mulVectSSSE3(tbl, d, p []byte)

func mulVectSIMD(tbl, d, p []byte, ext int) {
	if ext == avx2 {
		mulVectAVX2(tbl, d, p)
	} else {
		mulVectSSSE3(tbl, d, p)
	}
}

//go:noescape
func mulVectAddAVX2(tbl, d, p []byte)

//go:noescape
func mulVectAddSSSE3(tbl, d, p []byte)

func mulVectAddSIMD(tbl, d, p []byte, ext int) {
	if ext == avx2 {
		mulVectAddAVX2(tbl, d, p)
	} else {
		mulVectAddSSSE3(tbl, d, p)
	}
}

func (x *xrs) matrixMul(start, end int, dv, pv [][]byte) {
	d, p := x.Data, x.Parity
	g := x.genM
	off := 0
	for i := 0; i < d; i++ {
		for j := 0; j < p; j++ {
			t := lowHighTbl[g[j*d+i]][:]
			if i != 0 {
				mulVectAddSIMD(t, dv[i][start:end], pv[j][start:end], x.ext)
			} else {
				mulVectSIMD(t, dv[0][start:end], pv[j][start:end], x.ext)
			}
			off += 32
		}
	}
}

func (x *xrs) matrixMulRemain(start, end int, dv, pv [][]byte) { // end >= 16
	undone := end - start
	do := (undone >> 4) << 4 // it could be 0(when undone < 16)
	d, p := x.Data, x.Parity
	g := x.genM
	if do >= 16 {
		end2 := start + do
		off := 0
		for i := 0; i < d; i++ {
			for j := 0; j < p; j++ {
				t := lowHighTbl[g[j*d+i]][:]
				if i != 0 {
					mulVectAddSIMD(t, dv[i][start:end2], pv[j][start:end2], x.ext)
				} else {
					mulVectSIMD(t, dv[0][start:end2], pv[j][start:end2], x.ext)
				}
				off += 32
			}
		}
	}
	if undone > do { // 0 < undone - do < 16
		// may recalculate some Data(<16B), but still improve a lot (SIMD&reduce Cache pollution)
		start2 := end - 16
		off := 0
		for i := 0; i < d; i++ {
			for j := 0; j < p; j++ {
				t := lowHighTbl[g[j*d+i]][:]
				if i != 0 {
					mulVectAddSIMD(t, dv[i][start2:end], pv[j][start2:end], x.ext)
				} else {
					mulVectSIMD(t, dv[0][start2:end], pv[j][start2:end], x.ext)
				}
				off += 32
			}
		}
	}
}

func mulVectBase(c byte, a, b []byte) {
	t := mulTbl[c]
	for i := 0; i < len(a); i++ {
		b[i] = t[a[i]]
	}
}

func mulVectAddBase(c byte, a, b []byte) {
	t := mulTbl[c]
	for i := 0; i < len(a); i++ {
		b[i] ^= t[a[i]]
	}
}

// base method can't use compressed tables for encoding
func (x *xrs) encBase(vects [][]byte, rsOnly bool) (err error) {
	d := x.Data
	p := x.Parity
	size := len(vects[0])
	// step1: reedsolomon encM
	dv := vects[:d]
	pv := vects[d:]
	g := x.genM
	for i := 0; i < d; i++ {
		for j := 0; j < p; j++ {
			if i != 0 {
				mulVectAddBase(g[j*d+i], dv[i], pv[j])
			} else {
				mulVectBase(g[j*d], dv[0], pv[j])
			}
		}
	}
	if rsOnly == true {
		return
	}
	// step2: xor a & f(b)
	for i, a := range x.xm {
		v := make([][]byte, len(a)+1)
		v[0] = vects[i][size : size*2]
		for i, k := range a {
			v[i+1] = vects[k][0:size]
		}
		xorBase(vects[i][size:size*2], v)
	}
	return
}
