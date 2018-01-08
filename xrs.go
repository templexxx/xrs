/*
	X-Reed-Solomon Codes over GF(2^8)
	Primitive Polynomial:  x^8+x^4+x^3+x^2+1
	Galois Filed arithmetic using Intel SIMD instructions (AVX2 or SSSE3)
	Platform: X86-64

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
	"sync"

	"github.com/templexxx/cpufeat"
)

// Encoder implements for X-Reed-Solomon Encoding/Reconstructing
// size of vects(vectors) must be equal
// this interface can fit common Reed-Solomon Codes too
type EncReconster interface {
	// Encode outputs Parity into vects
	// each vect must has the same size && !=0 && !=odd
	Encode(vects [][]byte) error
	// Reconst repair missing vects (not just missing one Data vects)
	// e.g:
	// in 3+2, the whole index: [0,1,2,3,4]
	// if vects[0,4] are lost
	// the "has" will be [1,2,3] ,and you must be sure that vects[1] vects[2] vects[3] have correct Data
	// len(has) must be equal with num_data
	// results will be put into vects[0]&vects[4]
	Reconst(vects [][]byte, has, lost []int) error
	// ReconstData only repair Data
	ReconstData(vects [][]byte, has, lost []int) error
}

func checkCfg(d, p int) error {
	if (d <= 0) || (p <= 0) {
		return errors.New("xrs.checkCfg: Data or Parity <= 0")
	}
	if d+p >= 256 {
		return errors.New("xrs.checkCfg: Data+Parity >= 256")
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

// SIMD Instruction Extensions
const (
	none = iota
	ssse3
	avx2
)

func getEXT() int {
	if cpufeat.X86.HasAVX2 {
		return avx2
	} else if cpufeat.X86.HasSSSE3 {
		return ssse3
	} else {
		return none
	}
}

// TODO Data 可以被其它包直接引用吗？
type (
	xrs struct {
		Data   int
		Parity int
		ext    int    // cpu extension
		encM   matrix // encoding matrix
		genM   matrix // generator matrix
		xm     map[int][]int
		// TODO need _padding here?
		enableCache  bool // save time for calculating inverse matrix
		inverseCache sync.Map
	}
	// TODO maybe put it into xrs
	//encFunc struct {
	//	mulVectAddBase func(tbl, d, p []byte)
	//	mulVectBase    func(tbl, d, p []byte)
	//	xor        func(dst []byte, src [][]byte)
	//}
)

// TODO change the limits
// At most 3060 inverse matrix (when Data=14, Parity=4, calc by github.com/templexxx/reedsolomon/mathtool/cntinverse)
// In practice,  Data usually below 12, Parity below 5
func okCache(data, parity int) bool {
	if data < 15 && parity < 5 { // you can change it, but the Data+Parity can't be bigger than 32 (tips: see the codes about make inverse matrix)
		return true
	}
	return false
}

// TODO pointer or ？
func newXRS(d, p int, em matrix) (enc EncReconster) {
	g := em[d*d:]
	m := make(map[int][]int)
	genXM(d, p, m)
	return &xrs{Data: d, Parity: p, ext: getEXT(), encM: em, genM: g, xm: m, enableCache: okCache(d, p)}
}

// New create an Encoder (vandermonde matrix)
func New(data, parity int) (enc EncReconster, err error) {
	err = checkCfg(data, parity)
	if err != nil {
		return
	}
	e, err := genEncMatrixVand(data, parity)
	if err != nil {
		return
	}
	return newXRS(data, parity, e), nil
}

// NewCauchy create an Encoder (cauchy matrix)
func NewCauchy(data, parity int) (enc EncReconster, err error) {
	err = checkCfg(data, parity)
	if err != nil {
		return
	}
	e := genEncMatrixCauchy(data, parity)
	return newXRS(data, parity, e), nil
}
