/*
	X-Reed-Solomon Codes over GF(2^8)
	Primitive Polynomial:  x^8+x^4+x^3+x^2+1
	Galois Filed arithmetic using Intel SIMD instructions (AVX2 or SSSE3)

	xrs core ideas:
	1. reed-solomon encode
	2. split each vect into two equal pieces: a&b
	3. xor some vects from a with f(b) (f(b) is the result of rs encode bVects)

	More details:
	docs/..
*/

package xrs

import (
	"errors"
	"fmt"
	"sort"
)

// Encoder implements for X-Reed-Solomon Encoding/Reconstructing
// size of vects(vectors) must be equal
type Encoder interface {
	// Encode outputs parity into vects
	Encode(vects [][]byte) error

	// Reconst repair missing vects (not just missing one data vects)
	// e.g:
	// in 3+2, the whole index: [0,1,2,3,4]
	// if vects[0,4] are lost
	// the "has" will be [1,2,3] ,and you must be sure that vects[1] vects[2] vects[3] have correct data
	// len(has) must be equal with num_data
	// results will be put into vects[0]&vects[4]
	Reconst(vects [][]byte, has, lost []int) error

	// ReconstData only repair data
	ReconstData(vects [][]byte, has, lost []int) error

	// VectsNeed receive lost index return a&parity_b vects' index that are needed for reconstructing
	VectsNeed(lost int) (pb, a []int, err error)

	// ReconstOne reconst one data vect, it saves I/O
	// Make sure you have some specific vects ( you can get the vects index from VectsNeed)
	ReconstOne(vects [][]byte, lost int) error
}

func checkCfg(d, p int) error {
	if (d <= 0) || (p <= 0) {
		return errors.New("xrs.checkCfg: data or parity <= 0")
	}
	if d+p >= 256 {
		return errors.New("xrs.checkCfg: data+parity >= 256")
	}
	return nil
}

// generator xor_map <vects_index>:<data_a_index_list>
// e.g. 10+4
// 11:[0 3 6 9] 12:[1 4 7] 13:[2 5 8]
func genXM(d, p int, m map[int][]int) {
	a := 0
	for {
		if a == d {
			break
		}
		for i := d + 1; i < d+p; i++ {
			if a == d {
				break
			}
			l := m[i]
			l = append(l, a)
			m[i] = l
			a++
		}
	}
	return
}

// New create an Encoder (vandermonde matrix)
func New(data, parity int) (enc Encoder, err error) {
	err = checkCfg(data, parity)
	if err != nil {
		return
	}
	e, err := genEncMatrixVand(data, parity)
	if err != nil {
		return
	}
	m := make(map[int][]int)
	genXM(data, parity, m)
	return newRS(data, parity, e, m), nil
}

// NewCauchy create an Encoder (cauchy matrix)
func NewCauchy(data, parity int) (enc Encoder, err error) {
	err = checkCfg(data, parity)
	if err != nil {
		return
	}
	e := genEncMatrixCauchy(data, parity)
	m := make(map[int][]int)
	genXM(data, parity, m)
	return newRS(data, parity, e, m), nil
}

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

type encBase struct {
	data   int
	parity int
	encode []byte        // encode matrix
	gen    []byte        // generator matrix
	xm     map[int][]int // <vects_index>:<data_a_index_list>
}

func mulVect(c byte, a, b []byte) {
	t := mulTbl[c]
	for i := 0; i < len(a); i++ {
		b[i] = t[a[i]]
	}
}

func mulVectAdd(c byte, a, b []byte) {
	t := mulTbl[c]
	for i := 0; i < len(a); i++ {
		b[i] ^= t[a[i]]
	}
}

func (e *encBase) Encode(vects [][]byte) (err error) {
	d := e.data
	p := e.parity
	size, err := checkEnc(d, p, vects)
	if err != nil {
		return
	}
	dv := vects[:d]
	pv := vects[d:]
	g := e.gen
	// step1: reedsolomon encode
	for i := 0; i < d; i++ {
		for j := 0; j < p; j++ {
			if i != 0 {
				mulVectAdd(g[j*d+i], dv[i], pv[j])
			} else {
				mulVect(g[j*d], dv[0], pv[j])
			}
		}
	}
	// step2: xor a & f(b)
	for p, a := range e.xm {
		v := make([][]byte, len(a)+1)
		v[0] = vects[p][size : size*2]
		for i, k := range a {
			v[i+1] = vects[k][0:size]
		}
		xorBase(vects[p][size:size*2], v)
	}
	return
}

func (e *encBase) rsReconstData(vects [][]byte, has, lost []int) (err error) {
	d := e.data
	nl := len(lost)
	v := make([][]byte, d+nl)
	for i, p := range has {
		v[i] = vects[p]
	}
	for i, p := range lost {
		v[i+d] = vects[p]
	}
	mBuf := make([]byte, 4*d*d+nl*d) // help to reduce GC
	m := mBuf[:d*d]
	for i, l := range has {
		copy(m[i*d:i*d+d], e.encode[l*d:l*d+d])
	}
	raw := mBuf[d*d : 3*d*d]
	im := mBuf[3*d*d : 4*d*d] // inverse matrix
	err = matrix(m).invert(raw, d, im)
	if err != nil {
		return
	}
	g := mBuf[4*d*d:]
	for i, l := range lost {
		copy(g[i*d:i*d+d], im[l*d:l*d+d])
	}
	eTmp := &encBase{data: d, parity: nl, gen: g}
	err = eTmp.Encode(v[:d+nl])
	if err != nil {
		return
	}
	return
}

func (e *encBase) rsReconstParity(vects [][]byte, pLost []int) (err error) {
	d := e.data
	pl := len(pLost)
	v := make([][]byte, d+pl)
	g := make([]byte, pl*d)
	for i, l := range pLost {
		copy(g[i*d:i*d+d], e.encode[l*d:l*d+d])
	}
	for i := 0; i < d; i++ {
		v[i] = vects[i]
	}
	for i, p := range pLost {
		v[i+d] = vects[p]
	}
	etmp := &encBase{data: d, parity: pl, gen: g}
	err = etmp.Encode(v[:d+pl])
	if err != nil {
		return
	}
	return
}

func (e *encBase) rsReconst(vects [][]byte, has, dLost, pLost []int, dataOnly bool) (err error) {
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

func (e *encBase) reconst(vects [][]byte, has, lost []int, dataOnly bool) (err error) {
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
	// step2: parity back to f(b)
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

func (e *encBase) Reconst(vects [][]byte, has, lost []int) error {
	return e.reconst(vects, has, lost, false)
}

func (e *encBase) ReconstData(vects [][]byte, has, lost []int) error {
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

func (e *encBase) VectsNeed(lost int) (pb, a []int, err error) {
	return vectsNeed(e.data, lost, e.xm)
}

func (e *encBase) ReconstOne(vects [][]byte, lost int) error {
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
	// step2: encode -> f(b)
	pb, a, _ := e.VectsNeed(lost)
	bVTmp := make([][]byte, d+1)
	for i := 0; i < d; i++ {
		bVTmp[i] = bVects[i]
	}
	bVTmp[d] = bVects[pb[1]]
	g := make([]byte, d)
	copy(g, e.encode[pb[1]*d:pb[1]*d+d])
	etmp := &encBase{data: d, parity: 1, gen: g}
	etmp.Encode(bVTmp)
	// step3: pb xor f(b)&vects[a].. = lost_a
	xorV := make([][]byte, len(a)+1)
	xorV[0] = bVTmp[d]
	for i, a0 := range a {
		xorV[i+1] = vects[a0][:mid]
	}
	xorBase(vects[lost][:mid], xorV)
	return nil
}
