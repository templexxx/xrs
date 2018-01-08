package xrs

import (
	"errors"
	"sort"
)

func (e *encAVX2) Encode(vects [][]byte) (err error) {

}

func (e *encAVX2) reconst(vects [][]byte, has, lost []int, dataOnly bool) (err error) {
	dLost, pLost := spiltLost(e.data, lost)
	err = checkReconst(e.data, e.parity, has, dLost, pLost)
	if err != nil {
		return
	}
	sort.Ints(has)
	mid := len(vects[has[0]])
	// step1: repair a_vects
	aV := make([][]byte, e.data+e.parity)
	for i, v := range vects {
		aV[i] = v[:mid]
	}
	err = e.rsReconst(aV, has, dLost, pLost, dataOnly)
	if err != nil {
		return
	}
	// step2: Parity back to f(b)
	for _, h := range has {
		if h > e.data {
			a := e.xm[h]
			xv := make([][]byte, len(a)+1)
			xv[0] = vects[h][mid : mid*2]
			for i, a0 := range a {
				xv[i+1] = vects[a0][:mid]
			}
			xorAVX2(xv[0], xv)
		}
	}
	// step3: repair b_vects
	bV := make([][]byte, e.data+e.parity)
	for i, v := range vects {
		bV[i] = v[mid : mid*2]
	}
	err = e.rsReconst(bV, has, dLost, pLost, dataOnly)
	if err != nil {
		return
	}
	return
}

func (e *encAVX2) Reconst(vects [][]byte, has, lost []int) error {
	return e.reconst(vects, has, lost, false)
}

func (e *encAVX2) ReconstData(vects [][]byte, has, lost []int) error {
	return e.reconst(vects, has, lost, true)
}

func (e *encAVX2) VectsNeed(lost int) (pb, a []int, err error) {
	return vectsNeed(e.data, lost, e.xm)
}

func (e *encAVX2) rsReconstData(vects [][]byte, has, lost []int) (err error) {
	d := e.data
	nl := len(lost)
	v := make([][]byte, d+nl)
	for i, p := range has {
		v[i] = vects[p]
	}
	for i, p := range lost {
		v[i+d] = vects[p]
	}
	g, err := e.makeGen(has, lost)
	if err != nil {
		return
	}
	eTmp := &encAVX2{data: d, parity: nl, gen: g}
	err = eTmp.encodeGen(v)
	if err != nil {
		return
	}
	return
}

func (e *encAVX2) rsReconstParity(vects [][]byte, lost []int) (err error) {
	d := e.data
	nl := len(lost)
	v := make([][]byte, d+nl)
	g := make([]byte, nl*d)
	for i, l := range lost {
		copy(g[i*d:i*d+d], e.encode[l*d:l*d+d])
	}
	for i := 0; i < d; i++ {
		v[i] = vects[i]
	}
	for i, p := range lost {
		v[i+d] = vects[p]
	}
	eTmp := &encAVX2{data: d, parity: nl, gen: g}
	err = eTmp.encodeGen(v)
	if err != nil {
		return
	}
	return
}

func (e *encAVX2) rsReconst(vects [][]byte, has, dLost, pLost []int, dataOnly bool) (err error) {
	dl := len(dLost)
	if dl != 0 {
		err = e.rsReconstData(vects, has, dLost)
		if err != nil {
			return
		}
	}
	if dataOnly {
		return
	}
	pl := len(pLost)
	if pl != 0 {
		err = e.rsReconstParity(vects, pLost)
		if err != nil {
			return
		}
	}
	return
}

func (e *encAVX2) ReconstOne(vects [][]byte, lost int) error {
	// step1: recover b
	d := e.data
	bVects := make([][]byte, d+e.parity)
	mid := len(vects) / 2
	for i, v := range vects {
		bVects[i] = v[mid : mid*2]
	}
	bHas := make([]int, d)
	for i := 0; i < d; i++ {
		bHas[i] = i
	}
	bHas[lost] = d
	sort.Ints(bHas)
	err := e.rsReconstData(bVects, bHas, []int{lost})
	if err != nil {
		return err
	}
	// step2: encM -> f(b)
	pb, a, _ := e.VectsNeed(lost)
	bVTmp := make([][]byte, d+1)
	for i := 0; i < d; i++ {
		bVTmp[i] = bVects[i]
	}
	bVTmp[d] = bVects[pb[1]]
	g := make([]byte, d)
	copy(g, e.encode[pb[1]*d:pb[1]*d+d])
	etmp := &encAVX2{data: d, parity: 1, gen: g}
	etmp.Encode(bVTmp)
	// step3: pb xor f(b)&vects[a].. = lost_a
	xorV := make([][]byte, len(a)+1)
	xorV[0] = bVTmp[d]
	for i, a0 := range a {
		xorV[i+1] = vects[a0][:mid]
	}
	xorAVX2(vects[lost][:mid], xorV)
	return nil
}

func (e *encSSSE3) Encode(vects [][]byte) (err error) {
	d := e.data
	p := e.parity
	size, err := checkEnc(d, p, vects)
	if err != nil {
		return
	}
	dv := vects[:d]
	pv := vects[d:]
	start, end := 0, 0
	do := getDo(size)
	for start < size {
		end = start + do
		if end <= size {
			e.matrixMul(start, end, dv, pv)
			start = end
		} else {
			e.matrixMulRemain(start, size, dv, pv)
			start = size
		}
	}
	return
}

func (e *encSSSE3) matrixMul(start, end int, dv, pv [][]byte) {
	d := e.data
	p := e.parity
	tbl := e.tbl
	off := 0
	for i := 0; i < d; i++ {
		for j := 0; j < p; j++ {
			t := tbl[off : off+32]
			if i != 0 {
				mulVectAddSSSE3(t, dv[i][start:end], pv[j][start:end])
			} else {
				mulVectSSSE3(t, dv[0][start:end], pv[j][start:end])
			}
			off += 32
		}
	}
}

func (e *encSSSE3) matrixMulRemain(start, end int, dv, pv [][]byte) {
	undone := end - start
	do := (undone >> 4) << 4
	d := e.data
	p := e.parity
	tbl := e.tbl
	if do >= 16 {
		end2 := start + do
		off := 0
		for i := 0; i < d; i++ {
			for j := 0; j < p; j++ {
				t := tbl[off : off+32]
				if i != 0 {
					mulVectAddSSSE3(t, dv[i][start:end2], pv[j][start:end2])
				} else {
					mulVectSSSE3(t, dv[0][start:end2], pv[j][start:end2])
				}
				off += 32
			}
		}
		start = end
	}
	if undone > do {
		start2 := end - 16
		if start2 >= 0 {
			off := 0
			for i := 0; i < d; i++ {
				for j := 0; j < p; j++ {
					t := tbl[off : off+32]
					if i != 0 {
						mulVectAddSSSE3(t, dv[i][start2:end], pv[j][start2:end])
					} else {
						mulVectSSSE3(t, dv[0][start2:end], pv[j][start2:end])
					}
					off += 32
				}
			}
		} else {
			g := e.gen
			for i := 0; i < d; i++ {
				for j := 0; j < p; j++ {
					if i != 0 {
						mulVectAddBase(g[j*d+i], dv[i][start:], pv[j][start:])
					} else {
						mulVectBase(g[j*d], dv[0][start:], pv[j][start:])
					}
				}
			}
		}
	}
}

func (e *encSSSE3) encodeGen(vects [][]byte) (err error) {
	d := e.data
	p := e.parity
	size, err := checkEnc(d, p, vects)
	if err != nil {
		return
	}
	dv := vects[:d]
	pv := vects[d:]
	start, end := 0, 0
	do := getDo(size)
	for start < size {
		end = start + do
		if end <= size {
			e.matrixMulGen(start, end, dv, pv)
			start = end
		} else {
			e.matrixMulRemainGen(start, size, dv, pv)
			start = size
		}
	}
	return
}

func (e *encSSSE3) matrixMulGen(start, end int, dv, pv [][]byte) {
	d := e.data
	p := e.parity
	g := e.gen
	for i := 0; i < d; i++ {
		for j := 0; j < p; j++ {
			t := lowHighTbl[g[j*d+i]][:]
			if i != 0 {
				mulVectAddSSSE3(t, dv[i][start:end], pv[j][start:end])
			} else {
				mulVectSSSE3(t, dv[0][start:end], pv[j][start:end])
			}
		}
	}
}

func (e *encSSSE3) matrixMulRemainGen(start, end int, dv, pv [][]byte) {
	undone := end - start
	do := (undone >> 4) << 4
	d := e.data
	p := e.parity
	g := e.gen
	if do >= 16 {
		end2 := start + do
		for i := 0; i < d; i++ {
			for j := 0; j < p; j++ {
				t := lowHighTbl[g[j*d+i]][:]
				if i != 0 {
					mulVectAddSSSE3(t, dv[i][start:end2], pv[j][start:end2])
				} else {
					mulVectSSSE3(t, dv[0][start:end2], pv[j][start:end2])
				}
			}
		}
		start = end
	}
	if undone > do {
		start2 := end - 16
		if start2 >= 0 {
			for i := 0; i < d; i++ {
				for j := 0; j < p; j++ {
					t := lowHighTbl[g[j*d+i]][:]
					if i != 0 {
						mulVectAddSSSE3(t, dv[i][start2:end], pv[j][start2:end])
					} else {
						mulVectSSSE3(t, dv[0][start2:end], pv[j][start2:end])
					}
				}
			}
		} else {
			for i := 0; i < d; i++ {
				for j := 0; j < p; j++ {
					if i != 0 {
						mulVectAddBase(g[j*d+i], dv[i][start:], pv[j][start:])
					} else {
						mulVectBase(g[j*d], dv[0][start:], pv[j][start:])
					}
				}
			}
		}
	}
}

func (e *encSSSE3) Reconstruct(vects [][]byte) (err error) {
	return e.reconstruct(vects, false)
}

func (e *encSSSE3) ReconstructData(vects [][]byte) (err error) {
	return e.reconstruct(vects, true)
}

func (e *encSSSE3) ReconstWithPos(vects [][]byte, has, dLost, pLost []int) error {
	return e.reconstWithPos(vects, has, dLost, pLost, false)
}

func (e *encSSSE3) ReconstDataWithPos(vects [][]byte, has, dLost []int) error {
	return e.reconstWithPos(vects, has, dLost, nil, true)
}

func (e *encSSSE3) makeGen(has, dLost []int) (gen []byte, err error) {
	d := e.data
	em := e.encode
	cnt := len(dLost)
	if !e.enableCache {
		matrixbuf := make([]byte, 4*d*d+cnt*d)
		m := matrixbuf[:d*d]
		for i, l := range has {
			copy(m[i*d:i*d+d], em[l*d:l*d+d])
		}
		raw := matrixbuf[d*d : 3*d*d]
		im := matrixbuf[3*d*d : 4*d*d]
		err2 := matrix(m).invert(raw, d, im)
		if err2 != nil {
			return nil, err2
		}
		g := matrixbuf[4*d*d:]
		for i, l := range dLost {
			copy(g[i*d:i*d+d], im[l*d:l*d+d])
		}
		return g, nil
	}
	var ikey uint32
	for _, p := range has {
		ikey += 1 << uint8(p)
	}
	e.inverseCache.RLock()
	v, ok := e.inverseCache.data[ikey]
	if ok {
		im := v
		g := make([]byte, cnt*d)
		for i, l := range dLost {
			copy(g[i*d:i*d+d], im[l*d:l*d+d])
		}
		e.inverseCache.RUnlock()
		return g, nil
	}
	e.inverseCache.RUnlock()
	matrixbuf := make([]byte, 4*d*d+cnt*d)
	m := matrixbuf[:d*d]
	for i, l := range has {
		copy(m[i*d:i*d+d], em[l*d:l*d+d])
	}
	raw := matrixbuf[d*d : 3*d*d]
	im := matrixbuf[3*d*d : 4*d*d]
	err2 := matrix(m).invert(raw, d, im)
	if err2 != nil {
		return nil, err2
	}
	e.inverseCache.Lock()
	e.inverseCache.data[ikey] = im
	e.inverseCache.Unlock()
	g := matrixbuf[4*d*d:]
	for i, l := range dLost {
		copy(g[i*d:i*d+d], im[l*d:l*d+d])
	}
	return g, nil
}

func (e *encSSSE3) reconst(vects [][]byte, has, dLost, pLost []int, dataOnly bool) (err error) {
	d := e.data
	em := e.encode
	dCnt := len(dLost)
	size := len(vects[has[0]])
	if dCnt != 0 {
		vtmp := make([][]byte, d+dCnt)
		for i, p := range has {
			vtmp[i] = vects[p]
		}
		for i, p := range dLost {
			if len(vects[p]) == 0 {
				vects[p] = make([]byte, size)
			}
			vtmp[i+d] = vects[p]
		}
		g, err2 := e.makeGen(has, dLost)
		if err2 != nil {
			return
		}
		etmp := &encSSSE3{data: d, parity: dCnt, gen: g}
		err2 = etmp.encodeGen(vtmp)
		if err2 != nil {
			return err2
		}
	}
	if dataOnly {
		return
	}
	pCnt := len(pLost)
	if pCnt != 0 {
		g := make([]byte, pCnt*d)
		for i, l := range pLost {
			copy(g[i*d:i*d+d], em[l*d:l*d+d])
		}
		vtmp := make([][]byte, d+pCnt)
		for i := 0; i < d; i++ {
			vtmp[i] = vects[i]
		}
		for i, p := range pLost {
			if len(vects[p]) == 0 {
				vects[p] = make([]byte, size)
			}
			vtmp[i+d] = vects[p]
		}
		etmp := &encSSSE3{data: d, parity: pCnt, gen: g}
		err2 := etmp.encodeGen(vtmp)
		if err2 != nil {
			return err2
		}
	}
	return
}

func (e *encSSSE3) reconstWithPos(vects [][]byte, has, dLost, pLost []int, dataOnly bool) (err error) {
	d := e.data
	p := e.parity
	if len(has) != d {
		return errors.New("rs.Reconst: not enough vects")
	}
	dCnt := len(dLost)
	if dCnt > p {
		return errors.New("rs.Reconst: not enough vects")
	}
	pCnt := len(pLost)
	if pCnt > p {
		return errors.New("rs.Reconst: not enough vects")
	}
	return e.reconst(vects, has, dLost, pLost, dataOnly)
}

func (e *encSSSE3) reconstruct(vects [][]byte, dataOnly bool) (err error) {
	d := e.data
	p := e.parity
	t := d + p
	listBuf := make([]int, t+p)
	has := listBuf[:d]
	dLost := listBuf[d:t]
	pLost := listBuf[t : t+p]
	hasCnt, dCnt, pCnt := 0, 0, 0
	for i := 0; i < t; i++ {
		if vects[i] != nil {
			if hasCnt < d {
				has[hasCnt] = i
				hasCnt++
			}
		} else {
			if i < d {
				if dCnt < p {
					dLost[dCnt] = i
					dCnt++
				} else {
					return errors.New("rs.Reconst: not enough vects")
				}
			} else {
				if pCnt < p {
					pLost[pCnt] = i
					pCnt++
				} else {
					return errors.New("rs.Reconst: not enough vects")
				}
			}
		}
	}
	if hasCnt != d {
		return errors.New("rs.Reconst: not enough vects")
	}
	dLost = dLost[:dCnt]
	pLost = pLost[:pCnt]
	return e.reconst(vects, has, dLost, pLost, dataOnly)
}
