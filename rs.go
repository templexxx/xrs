/*
	X-Reed-Solomon Codes over GF(2^8)
	Primitive Polynomial:  x^8+x^4+x^3+x^2+1
	Galois Filed arithmetic using Intel SIMD instructions (AVX2 or SSSE3)
*/

package reedsolomon

import (
	"errors"
	"unicode"
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
}

func checkCfg(d, p int) error {
	if (d <= 0) || (p <= 0) {
		return errors.New("rs.New: data or parity <= 0")
	}
	if d+p >= 256 {
		return errors.New("rs.New: data+parity >= 256")
	}
	return nil
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
	return newRS(data, parity, e), nil
}

// NewCauchy create an Encoder (cauchy matrix)
func NewCauchy(data, parity int) (enc Encoder, err error) {
	err = checkCfg(data, parity)
	if err != nil {
		return
	}
	e := genEncMatrixCauchy(data, parity)
	return newRS(data, parity, e), nil
}

type encBase struct {
	data   int
	parity int
	encode []byte
	gen    []byte
}

func checkEnc(d, p int, vs [][]byte) (size int, err error) {
	total := len(vs)
	if d+p != total {
		err = errors.New("rs.checkER: vects not match rs args")
		return
	}
	size = len(vs[0])
	if size == 0 {
		err = errors.New("rs.checkER: vects size = 0")
		return
	}
	for i := 1; i < total; i++ {
		if len(vs[i]) != size {
			err = errors.New("rs.checkER: vects size mismatch")
			return
		}
	}
	return
}

func (e *encBase) Encode(vects [][]byte) (err error) {
	d := e.data
	p := e.parity
	_, err = checkEnc(d, p, vects)
	if err != nil {
		return
	}
	dv := vects[:d]
	pv := vects[d:]
	g := e.gen
	for i := 0; i < d; i++ {
		for j := 0; j < p; j++ {
			if i != 0 {
				mulVectAdd(g[j*d+i], dv[i], pv[j])
			} else {
				mulVect(g[j*d], dv[0], pv[j])
			}
		}
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
	return e.reconstWithPos(vects, has, lost, false)
}

func (e *encBase) ReconstData(vects [][]byte, has, lost []int) error {
	return e.reconstWithPos(vects, has, lost, true)
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

func (e *encBase) reconst(vects [][]byte, has, dLost, pLost []int, dataOnly bool) (err error) {
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

func (e *encBase) reconstWithPos(vects [][]byte, has, lost []int, dataOnly bool) (err error) {
	d := e.data
	p := e.parity
	// TODO check more, maybe element in has show in lost & deal with len(has) > d
	if len(has) != d {
		return errors.New("rs.Reconst: not enough vects")
	}
	dLost, pLost := spiltLost(e.data, lost)
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
