package xrs

import (
	"errors"
	"fmt"
	"sort"
)

// if enableCache, search cache first
// if not found generate one
func (x *xrs) makeGen(has, lost []int) (gen []byte, err error) {
	d := x.Data
	em := x.encM
	dl := len(lost)
	if !x.enableCache {
		mBuf := make([]byte, 4*d*d+dl*d)
		m := mBuf[:d*d]
		for i, l := range has {
			copy(m[i*d:i*d+d], em[l*d:l*d+d])
		}
		raw := mBuf[d*d : 3*d*d]
		im := mBuf[3*d*d : 4*d*d]
		err2 := matrix(m).invert(raw, d, im)
		if err2 != nil {
			return nil, err2
		}
		g := mBuf[4*d*d:]
		for i, l := range lost {
			// TODO do I need copy here?
			copy(g[i*d:i*d+d], im[l*d:l*d+d])
		}
		return g, nil
	}
	var ikey uint32
	for _, p := range has {
		ikey += 1 << uint8(p)
	}
	v, ok := x.inverseCache.Load(ikey)
	if ok {
		im := v.([]byte)
		g := make([]byte, dl*d)
		for i, l := range lost {
			copy(g[i*d:i*d+d], im[l*d:l*d+d])
		}
		return g, nil
	}
	mBuf := make([]byte, 4*d*d+dl*d)
	m := mBuf[:d*d]
	for i, l := range has {
		copy(m[i*d:i*d+d], em[l*d:l*d+d])
	}
	raw := mBuf[d*d : 3*d*d]
	im := mBuf[3*d*d : 4*d*d]
	err2 := matrix(m).invert(raw, d, im)
	if err2 != nil {
		return nil, err2
	}
	x.inverseCache.Store(ikey, im)
	g := mBuf[4*d*d:]
	for i, l := range lost {
		copy(g[i*d:i*d+d], im[l*d:l*d+d])
	}
	return g, nil
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
	g, err := x.makeGen(has, lost)
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

func (e *xrs) rsReconstParity(vects [][]byte, lost []int) (err error) {
	d := e.data
	nl := len(lost)
	v := make([][]byte, d+nl)
	g := make([]byte, nl*d)
	for i, l := range lost {
		copy(g[i*d:i*d+d], e.encM[l*d:l*d+d])
	}
	for i := 0; i < d; i++ {
		v[i] = vects[i]
	}
	for i, p := range lost {
		v[i+d] = vects[p]
	}
	etmp := &encBase{data: d, parity: nl, gen: g}
	err = etmp.Encode(v)
	if err != nil {
		return
	}
	return
}

func (e *xrs) rsReconst(vects [][]byte, has, dLost, pLost []int, dataOnly bool) (err error) {
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

func (e *xrs) reconst(vects [][]byte, has, lost []int, dataOnly bool) (err error) {
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
			xorBase(xv[0], xv)
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

func (e *xrs) Reconst(vects [][]byte, has, lost []int) error {
	return e.reconst(vects, has, lost, false)
}

func (e *xrs) ReconstData(vects [][]byte, has, lost []int) error {
	return e.reconst(vects, has, lost, true)
}

func vectsNeed(d, lost int, xm map[int][]int) (pb, a []int, err error) {
	if lost < 0 || lost >= d {
		err = errors.New(fmt.Sprintf("xrs.VectsNeed: can't rsReconst vects[%d] by xor; numData is %d", lost, d))
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
	err = x.rsReconstData(bVects, bHas, []int{lost})
	if err != nil {
		return
	}
	// step2: encM -> f(b)
	bVTmp := make([][]byte, d+1)
	for i := 0; i < d; i++ {
		bVTmp[i] = bVects[i]
	}
	bVTmp[d] = bVects[pb[1]]
	g := make([]byte, d)
	copy(g, x.encM[pb[1]*d:pb[1]*d+d])
	xTmp := &xrs{Data: d, Parity: 1, genM: g, ext: x.ext}
	xTmp.enc(bVTmp, true)
	// step3: pb xor f(b)&vects[a].. = lost_a
	xorV := make([][]byte, len(a)+1)
	xorV[0] = bVTmp[d]
	for i, a0 := range a {
		xorV[i+1] = vects[a0][:mid]
	}
	encXOR(vects[lost][:mid], xorV, x.ext)
	return
}
