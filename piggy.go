/**
 * piggybacking over in GF(2^8).
 * Primitive Polynomial: x^8 + x^4 + x^3 + x^2 + 1 (0x1d)
 */
package xrs

import "errors"

type piggy struct {
	data   int    // Number of data shards
	parity int    // Number of parity shards(reedsolomon codes parity)
	shards int    // Total number of shards
	xMap   xorMap // key:row of parity; value: rows of datas
	m      matrix // encoding matrix, identity matrix(upper) + generator matrix(lower)
	gen    matrix // generator matrix(cauchy matrix)
	ins    int    // Extensions Instruction(avx2)
}

type xorMap map[int][]int

const (
	avx2 = 0
)

var ErrNoSupportINS = errors.New("piggy: no avx2")

// New : create a encoding matrix for encoding, reconstruction
func New(d, p int) (*piggy, error) {
	err := checkShards(d, p)
	if err != nil {
		return nil, err
	}
	m := make(map[int][]int)
	m = genXorMap(d, p)
	pig := piggy{
		data:   d,
		parity: p,
		shards: d + p,
		xMap:   m,
	}
	if hasAVX2() {
		pig.ins = avx2
	} else {
		return &pig, ErrNoSupportINS
	}
	e := genEncodeMatrix(pig.shards, d) // create encoding matrix
	pig.m = e
	pig.gen = NewMatrix(p, d)
	for i := range pig.gen {
		pig.gen[i] = pig.m[d+i]
	}
	return &pig, err
}

func genXorMap(d, p int) map[int][]int {
	unit := d / (p-1)
	left := d - unit*(p-1)
	xmap := make(map[int][]int)
	if left == 0 {
		offset := 0
		for j := 1; j < p; j++ {
			var xors []int
			for i := offset; i < unit+offset; i++ {
				xors = append(xors, i)
			}
			offset = offset + unit
			xmap[j] = xors
		}
		return xmap
	}
	unit2 := unit +1
	// au+(p-1-a)(u+1) = d
	a := unit2 * (p-1) - d
	offset := 0
	for j := 1; j <= a; j++ {
		var xors []int
		for i := offset; i < unit+offset; i++ {
			xors = append(xors, i)
		}
		offset = offset + unit
		xmap[j] = xors
	}
	for j := a+1; j < p;j++ {
		var xors []int
		for i := offset; i < unit2+offset; i++ {
			xors = append(xors, i)
		}
		offset = offset + unit2
		xmap[j] = xors
	}
	return xmap
}

var ErrTooFewShards = errors.New("piggy: too few shards given for encoding")
var ErrTooManyShards = errors.New("piggy: too many shards given for encoding")

func checkShards(d, p int) error {
	if (d <= 0) || (p <= 0) {
		return ErrTooFewShards
	}
	if d+p >= 255 {
		return ErrTooManyShards
	}
	return nil
}

//go:noescape
func hasAVX2() bool
