/*
	X-Reed-Solomon Codes over GF(2^8)
	Primitive Polynomial:  x^8+x^4+x^3+x^2+1
	Galois Filed arithmetic using Intel SIMD instructions (AVX2 or SSSE3)

	xrs core ideas:
	1. reed-solomon encode
	2. Split each vect into two equal pieces: a&b
	3. xor some vects from a with f(b) (f(b) is the result of rs encode)
*/

package xrs

import (
	"errors"
	"fmt"
)

// Encoder implements for X-Reed-Solomon Encoding/Reconstructing
type Encoder interface {
	// Encode outputs parity into vects
	// warning: len(vects) must be equal with num of data+parity
	Encode(vects [][]byte) error
	// Reconst repair lost data&parity with has&lost vects position
	// e.g:
	// in 3+2, the whole position: [0,1,2,3,4]
	// if lost vects[0,4]
	// the "has" will be [1,2,3]
	// then you must be sure that vects[1] vects[2] vects[3] have correct data
	// results will be put into vects[0]&vects[4]
	// warning:
	// 1. each vect has same len, don't set it nil
	// 2. len(has) must equal num of data vects
	Reconst(vects [][]byte, has, lost []int) error
	// ReconstData only repair lost data with survived&lost vects position
	ReconstData(vects [][]byte, has, lost []int) error
	// VectsNeed receive lost position return vects position that are needed for reconstructing
	VectsNeed(lost int) (pb int, a []int, err error) // pb: index of b parity
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

type encBase struct {
	data   int
	parity int
	encode []byte
	gen    []byte
	xm     map[int][]int // <vects_index>:<data_a_index_list>
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
		i := 1
		for _, k := range a {
			v[i] = vects[k][0:size]
			i++
		}
		v[0] = vects[p][size : size*2]
		xorBase(vects[p][size:size*2], v)
	}
	return
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

func (e *encBase) Reconst(vects [][]byte, has, lost []int) error {
	return e.reconst(vects, has, lost, false)
}

func (e *encBase) ReconstData(vects [][]byte, has, lost []int) error {
	return e.reconst(vects, has, lost, true)
}

func (e *encBase) rsReconst(vects [][]byte, has, dLost, pLost []int, dataOnly bool) (err error) {
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
		matrixbuf := make([]byte, 4*d*d+dCnt*d)
		m := matrixbuf[:d*d]
		for i, l := range has {
			copy(m[i*d:i*d+d], em[l*d:l*d+d])
		}
		raw := matrixbuf[d*d : 3*d*d]
		im := matrixbuf[3*d*d : 4*d*d]
		err2 := matrix(m).invert(raw, d, im)
		if err2 != nil {
			return err2
		}
		g := matrixbuf[4*d*d:]
		for i, l := range dLost {
			copy(g[i*d:i*d+d], im[l*d:l*d+d])
		}
		etmp := &encBase{data: d, parity: dCnt, gen: g}
		err2 = etmp.Encode(vtmp[:d+dCnt])
		if err2 != nil {
			return err2
		}
	}
	if dataOnly {
		return
	}
	pCnt := len(pLost)
	if pCnt != 0 {
		vtmp := make([][]byte, d+pCnt)
		g := make([]byte, pCnt*d)
		for i, l := range pLost {
			copy(g[i*d:i*d+d], em[l*d:l*d+d])
		}
		for i := 0; i < d; i++ {
			vtmp[i] = vects[i]
		}
		for i, p := range pLost {
			if len(vects[p]) == 0 {
				vects[p] = make([]byte, size)
			}
			vtmp[i+d] = vects[p]
		}
		etmp := &encBase{data: d, parity: pCnt, gen: g}
		err2 := etmp.Encode(vtmp[:d+pCnt])
		if err2 != nil {
			return err2
		}
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
	return
}

func (e *encBase) reconst(vects [][]byte, has, lost []int, dataOnly bool) (err error) {
	d := e.data
	p := e.parity
	// TODO check more, maybe element in has show in lost & deal with len(has) > d
	if len(has) < d {
		return errors.New("xrs.Reconst: not enough vects")
	}
	has = has[:d]
	if len(has) != d {
		return errors.New("xrs.Reconst: not enough vects")
	}
	dLost, pLost := spiltLost(e.data, lost)
	dCnt := len(dLost)
	if dCnt > p {
		return errors.New("xrs.Reconst: not enough vects")
	}
	pCnt := len(pLost)
	if pCnt > p {
		return errors.New("xrs.Reconst: not enough vects")
	}
	return e.rsReconst(vects, has, dLost, pLost, dataOnly)
}

func isIn(e int, s []int) bool {
	for _, v := range s {
		if e == v {
			return true
		}
	}
	return false
}

func (e *encBase) VectsNeed(lost int) (pb int, a []int, err error) {
	if lost < 0 || lost >= e.data {
		err = errors.New(fmt.Sprintf("xrs.VectsNeed: can't rsReconst vects[%d] by xor; numData is %d", lost, e.data))
		return
	}
	for k, s := range e.xm {
		if isIn(lost, s) {
			pb = k
			break
		}
	}
	for _, v := range e.xm[pb] {
		if v != lost {
			a = append(a, v)
		}
	}
	return
}
