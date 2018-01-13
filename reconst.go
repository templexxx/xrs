package xrs

import (
	"errors"
	"fmt"
	"sort"
)

func (x *xrs) makeGen(has, lost []int) (im, gen []byte, err error) {
	d := x.Data
	em := x.encM
	nl := len(lost)
	mBuf := make([]byte, 4*d*d+nl*d)
	m := mBuf[:d*d]
	for i, l := range has {
		copy(m[i*d:i*d+d], em[l*d:l*d+d])
	}
	raw := mBuf[d*d : 3*d*d]
	im = mBuf[3*d*d : 4*d*d] // inverse Matrix
	err = matrix(m).invert(raw, d, im)
	if err != nil {
		return
	}
	gen = mBuf[4*d*d:]
	for i, l := range lost {
		copy(gen[i*d:i*d+d], im[l*d:l*d+d])
	}
	return
}

// TODO use this one
// TODO drop all copy
//func (x *xrs) getDecMatrix(has, lost []int) (im []byte, err error) {
//	d := x.Data
//	em := x.encM
//	nl := len(lost)
//	mBuf := make([]byte, 4*d*d+nl*d)
//	m := mBuf[:d*d]
//	for i, l := range has {
//		copy(m[i*d:i*d+d], em[l*d:l*d+d])
//	}
//	raw := mBuf[d*d : 3*d*d]
//	im = mBuf[3*d*d : 4*d*d] // inverse Matrix
//	err = matrix(m).invert(raw, d, im)
//	if err != nil {
//		return
//	}
//	return
//}

// if enableCache, search cache first
// if not found make one
func (x *xrs) getGen(has, lost []int) (gen []byte, err error) {
	d := x.Data
	dl := len(lost)
	if !x.enableCache {
		_, gen, err = x.makeGen(has, lost)
		return
	}
	var ikey uint32
	for _, p := range has {
		ikey += 1 << uint8(p)
	}
	v, ok := x.inverseCache.Load(ikey)
	if ok {
		im := v.([]byte)
		gen = make([]byte, dl*d)
		for i, l := range lost {
			copy(gen[i*d:i*d+d], im[l*d:l*d+d])
		}
		return
	}
	im, gen, err := x.makeGen(has, lost)
	if err != nil {
		return
	}
	x.inverseCache.Store(ikey, im)
	return
}

func (x *xrs) rsReconstData(vects [][]byte, has, lost []int) (err error) {
	d := x.Data
	nl := len(lost)
	v := make([][]byte, d+nl)
	for i, p := range has {
		v[i] = vects[p]
	}
	for i, p := range lost {
		v[i+d] = vects[p]
	}
	g, err := x.getGen(has, lost)
	if err != nil {
		return
	}
	xTmp := &xrs{Data: d, Parity: nl, genM: g, ext: x.ext}
	err = xTmp.enc(v, true)
	if err != nil {
		return
	}
	return
}

func (x *xrs) rsReconstParity(vects [][]byte, lost []int) (err error) {
	d := x.Data
	nl := len(lost)
	v := make([][]byte, d+nl)
	g := make([]byte, nl*d)
	for i, l := range lost {
		copy(g[i*d:i*d+d], x.encM[l*d:l*d+d])
	}
	for i := 0; i < d; i++ {
		v[i] = vects[i]
	}
	for i, p := range lost {
		v[i+d] = vects[p]
	}
	xTmp := &xrs{Data: d, Parity: nl, genM: g, ext: x.ext}
	err = xTmp.enc(v, true)
	if err != nil {
		return
	}
	return
}

func (x *xrs) rsReconst(vects [][]byte, has, dLost, pLost []int, dataOnly bool) (err error) {
	dl := len(dLost)
	if dl != 0 {
		err = x.rsReconstData(vects, has, dLost)
		if err != nil {
			return
		}
	}
	if dataOnly {
		return
	}
	pl := len(pLost)
	if pl != 0 {
		err = x.rsReconstParity(vects, pLost)
		if err != nil {
			return
		}
	}
	return
}

func isIn(e int, s []int) bool {
	for _, v := range s {
		if e == v {
			return true
		}
	}
	return false
}

func checkReconst(d, p int, has, dLost, pLost []int) (err error) {
	if len(has) < d {
		err = errors.New("xrs.Reconst: not enough vects")
		return
	}
	if len(has) > d { // = d is the best practice
		err = errors.New("xrs.Reconst: too many vects")
		return
	}
	for _, h := range has {
		if isIn(h, dLost) {
			err = errors.New("xrs.Reconst: has&lost are conflicting")
			return
		}
		if isIn(h, pLost) {
			err = errors.New("xrs.Reconst: has&lost are conflicting")
			return
		}
	}
	if len(dLost)+len(pLost) > p {
		err = errors.New("xrs.Reconst: not enough vects")
		return
	}
	return
}

// TODO drop sort, sort 代价多大？
func spiltLost(d int, lost []int) (dLost, pLost []int) {
	for _, l := range lost {
		if l >= d {
			pLost = append(pLost, l)
		} else {
			dLost = append(dLost, l)
		}
	}
	sort.Ints(dLost)
	sort.Ints(pLost)
	return
}

func (x *xrs) reconst(vects [][]byte, has, lost []int, dataOnly bool) (err error) {
	dLost, pLost := spiltLost(x.Data, lost)
	err = checkReconst(x.Data, x.Parity, has, dLost, pLost)
	if err != nil {
		return
	}
	sort.Ints(has)
	mid := len(vects[has[0]]) / 2
	aV := make([][]byte, x.Data+x.Parity)
	bV := make([][]byte, x.Data+x.Parity)
	for i, v := range vects {
		aV[i] = v[:mid]
		bV[i] = v[mid:]
	}

	// step1: repair a_vects
	err = x.rsReconst(aV, has, dLost, pLost, dataOnly)
	if err != nil {
		return
	}
	// step2: b_vects back to rs
	for _, h := range has {
		if h > x.Data {
			a := x.xm[h]
			xv := make([][]byte, len(a)+1)
			xv[0] = vects[h][mid:]
			for i, a0 := range a {
				xv[i+1] = vects[a0][:mid]
			}
			bRS := make([]byte, mid)
			bV[h] = bRS
			encXOR(bRS, xv, x.ext)
		}
	}
	// step3: repair b_vects (RS)
	err = x.rsReconst(bV, has, dLost, pLost, dataOnly)
	if err != nil {
		return
	}
	// step4: xor a & f(b)
	for _, l := range pLost {
		if l > x.Data {
			a := x.xm[l]
			xv := make([][]byte, len(a)+1)
			xv[0] = bV[l]
			for i, a0 := range a {
				xv[i+1] = vects[a0][:mid]
			}
			encXOR(vects[l][mid:], xv, x.ext)
		}
	}
	return
}

func (x *xrs) Reconst(vects [][]byte, has, lost []int) error {
	return x.reconst(vects, has, lost, false)
}

func (x *xrs) ReconstData(vects [][]byte, has, lost []int) error {
	return x.reconst(vects, has, lost, true)
}

func vectsNeed(d, lost int, xm map[int][]int) (pb, a []int, err error) {
	if lost < 0 || lost >= d {
		err = fmt.Errorf("xrs.VectsNeed: can't rsReconst vects[%d] by xor; numData is %d", lost, d)
		return
	}
	pb = make([]int, 2)
	pb[0] = d // must has b_vects[d]
	for k, s := range xm {
		if isIn(lost, s) {
			pb[1] = k
			break
		}
	}
	for _, v := range xm[pb[1]] {
		if v != lost {
			a = append(a, v)
		}
	}
	return
}

// VectsNeed receive lost index return a&parity_b vects' index that are needed for reconstructing
func (x *xrs) VectsNeed(lost int) (pb, a []int, err error) {
	return vectsNeed(x.Data, lost, x.xm)
}

// ReconstOne reconst one Data vect, it saves I/O
// Make sure you have some specific vects ( you can get the vects index from VectsNeed)
func (x *xrs) ReconstOne(vects [][]byte, lost int) (err error) {
	pb, a, err := x.VectsNeed(lost) // help with checking lost index
	if err != nil {
		return
	}
	// step1: recover b
	d := x.Data
	bVects := make([][]byte, d+x.Parity)
	mid := len(vects[0]) / 2
	for i, v := range vects {
		bVects[i] = v[mid : mid*2]
	}
	bHas := make([]int, d)
	for i := 0; i < d; i++ {
		bHas[i] = i
	}
	bHas[lost] = d
	sort.Ints(bHas)
	err = x.rsReconstData(bVects, bHas, []int{lost})
	if err != nil {
		return
	}
	// step2: encM -> f(b)
	bVTmp := make([][]byte, d+1)
	for i := 0; i < d; i++ {
		bVTmp[i] = bVects[i]
	}
	bVTmp[d] = make([]byte, mid)
	//bVTmp[d] = bVects[pb[1]]
	g := make([]byte, d)
	copy(g, x.encM[pb[1]*d:pb[1]*d+d])
	xTmp := &xrs{Data: d, Parity: 1, genM: g, ext: x.ext}
	xTmp.enc(bVTmp, true)
	// step3: pb xor f(b)&vects[a].. = lost_a
	xorV := make([][]byte, len(a)+2)
	xorV[0] = bVTmp[d]
	xorV[1] = vects[pb[1]][mid:]
	for i, a0 := range a {
		xorV[i+2] = vects[a0][:mid]
	}
	encXOR(vects[lost][:mid], xorV, x.ext)
	return
}
