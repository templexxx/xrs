package xrs

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"
)

const (
	kb         = 1024
	mb         = 1024 * 1024
	testNumIn  = 10
	testNumOut = 4
)

const verifySize = 256 + 32 + 16 + 8

func TestEncRSBase(t *testing.T) {
	d := 5
	p := 5
	vects := [][]byte{
		{0, 1},
		{4, 5},
		{2, 3},
		{6, 7},
		{8, 9},
		{0, 0},
		{0, 0},
		{0, 0},
		{0, 0},
		{0, 0},
	}
	em, err := genEncMatrixVand(d, p)
	if err != nil {
		t.Fatal(err)
	}
	g := em[d*d:]
	e := &xrs{Data: d, Parity: p, genM: g, ext: none}
	err = e.enc(vects, true)
	if err != nil {
		t.Fatal(err)
	}
	if vects[5][0] != 12 || vects[5][1] != 13 {
		t.Fatal("vect 5 mismatch")
	}
	if vects[6][0] != 10 || vects[6][1] != 11 {
		t.Fatal("vect 6 mismatch")
	}
	if vects[7][0] != 14 || vects[7][1] != 15 {
		t.Fatal("vect 7 mismatch")
	}
	if vects[8][0] != 90 || vects[8][1] != 91 {
		t.Fatal("vect 8 mismatch")
	}
	if vects[9][0] != 94 || vects[9][1] != 95 {
		t.Fatal("shard 9 mismatch")
	}
}

//func TestGenXMap(t *testing.T) {
//	d := 5
//	p := 5
//	m := make(map[int][]int)
//	genXM(d, p, m)
//	fmt.Println(m)
//}

// xm: map[8:[2] 9:[3] 6:[0 4] 7:[1]]
func TestEncBase(t *testing.T) {
	d := 5
	p := 5
	vects := [][]byte{
		{0, 1},
		{4, 5},
		{2, 3},
		{6, 7},
		{8, 9},
		{0, 0},
		{0, 0},
		{0, 0},
		{0, 0},
		{0, 0},
	}
	em, err := genEncMatrixVand(d, p)
	if err != nil {
		t.Fatal(err)
	}
	g := em[d*d:]
	xm := make(map[int][]int)
	genXM(d, p, xm)
	x := &xrs{Data: d, Parity: p, genM: g, ext: none, xm: xm}
	err = x.Encode(vects)
	if err != nil {
		t.Fatal(err)
	}
	if vects[5][0] != 12 || vects[5][1] != 13 {
		t.Fatal("vect 5 mismatch")
	}
	if vects[6][0] != 10 || vects[6][1] != 3 {
		t.Fatal("vect 6 mismatch")
	}
	if vects[7][0] != 14 || vects[7][1] != 11 {
		t.Fatal("vect 7 mismatch")
	}
	if vects[8][0] != 90 || vects[8][1] != 89 {
		t.Fatal("vect 8 mismatch")
	}
	if vects[9][0] != 94 || vects[9][1] != 89 {
		t.Fatal("shard 9 mismatch")
	}
}

func fillRandom(v []byte) {
	for i := 0; i < len(v); i += 7 {
		val := rand.Int63()
		for j := 0; i+j < len(v) && j < 7; j++ {
			v[i+j] = byte(val)
			val >>= 8
		}
	}
}

//
func verifyEncSIMD(t *testing.T, d, p, ext int) {
	for i := 2; i <= verifySize; i += 2 {
		vects1 := make([][]byte, d+p)
		vects2 := make([][]byte, d+p)
		for j := 0; j < d+p; j++ {
			vects1[j] = make([]byte, i)
			vects2[j] = make([]byte, i)
		}
		for j := 0; j < d; j++ {
			rand.Seed(int64(j))
			fillRandom(vects1[j])
			copy(vects2[j], vects1[j])
		}
		em, err := genEncMatrixVand(d, p)
		if err != nil {
			t.Fatal(err)
		}
		g := em[d*d:]
		xm := make(map[int][]int)
		genXM(d, p, xm)
		x := &xrs{Data: d, Parity: p, genM: g, ext: ext, xm: xm}
		err = x.Encode(vects1)
		if err != nil {
			t.Fatal(err)
		}

		x2 := &xrs{Data: d, Parity: p, genM: g, ext: none, xm: xm}
		err = x2.Encode(vects2)
		for k, v1 := range vects1 {
			if !bytes.Equal(v1, vects2[k]) {
				var extS string
				if ext == avx2 {
					extS = "avx2"
				}
				if ext == ssse3 {
					extS = "ssse3"
				}
				t.Fatalf("no match enc with encBase; vect: %d; size: %d; ext: %s", k, i, extS)
			}
		}
	}
}

func TestEncSIMD(t *testing.T) {
	if getEXT() == avx2 {
		verifyEncSIMD(t, testNumIn, testNumOut, avx2)
		verifyEncSIMD(t, testNumIn, testNumOut, ssse3)
	} else if getEXT() == ssse3 {
		verifyEncSIMD(t, testNumIn, testNumOut, ssse3)
	} else {
		t.Log("can't use SIMD")
	}
}

func verifyReconst(t *testing.T, d, p, ext int, has, lost []int) {
	for i := 2; i <= verifySize; i = i + 2 {
		vects1 := make([][]byte, d+p)
		vects2 := make([][]byte, d+p)
		for j := 0; j < d+p; j++ {
			vects1[j] = make([]byte, i)
			vects2[j] = make([]byte, i)
		}
		for j := 0; j < d; j++ {
			rand.Seed(int64(j))
			fillRandom(vects1[j])
		}
		em, err := genEncMatrixVand(d, p)
		if err != nil {
			t.Fatal(err)
		}
		g := em[d*d:]
		xm := make(map[int][]int)
		genXM(d, p, xm)
		x := &xrs{Data: d, Parity: p, genM: g, ext: ext, xm: xm, encM: em}
		err = x.Encode(vects1)
		if err != nil {
			t.Fatal(err)
		}

		for j := 0; j < d+p; j++ {
			copy(vects2[j], vects1[j])
		}
		for _, l := range lost {
			vects2[l] = make([]byte, i)
		}
		if len(lost) == 1 && lost[0] < d {
			err = x.ReconstOne(vects2, lost[0])
			if err != nil {
				t.Fatal(err)
			}
		} else {
			err = x.Reconst(vects2, has, lost)
			if err != nil {
				t.Fatal(err)
			}
		}
		for k, v1 := range vects1 {
			if !bytes.Equal(v1, vects2[k]) {
				t.Fatalf("no match reconst; vect: %d; size: %d", k, i)
			}
		}
	}
}

func TestReconstOne(t *testing.T) {
	for i := 0; i < testNumIn; i++ {
		has := make([]int, testNumIn+testNumOut)
		for j := 0; j < testNumIn+testNumOut; j++ {
			has[j] = j
		}
		if getEXT() == avx2 {
			verifyReconst(t, testNumIn, testNumOut, avx2, has, []int{i})
			verifyReconst(t, testNumIn, testNumOut, ssse3, has, []int{i})
			verifyReconst(t, testNumIn, testNumOut, none, has, []int{i})
		} else if getEXT() == ssse3 {
			verifyReconst(t, testNumIn, testNumOut, ssse3, has, []int{i})
			verifyReconst(t, testNumIn, testNumOut, none, has, []int{i})
		} else {
			verifyReconst(t, testNumIn, testNumOut, none, has, []int{i})
		}
	}
}

func TestReconst(t *testing.T) {
	lost := []int{3, 9, 0, 11}
	has := []int{10, 8, 5, 6, 7, 4, 2, 12, 13, 1}
	if getEXT() == avx2 {
		verifyReconst(t, testNumIn, testNumOut, avx2, has, lost)
		verifyReconst(t, testNumIn, testNumOut, ssse3, has, lost)
		verifyReconst(t, testNumIn, testNumOut, none, has, lost)
	} else if getEXT() == ssse3 {
		verifyReconst(t, testNumIn, testNumOut, ssse3, has, lost)
		verifyReconst(t, testNumIn, testNumOut, none, has, lost)
	} else {
		verifyReconst(t, testNumIn, testNumOut, none, has, lost)
	}
}

func benchEnc(b *testing.B, d, p, size int) {
	vects := make([][]byte, d+p)
	for j := 0; j < d+p; j++ {
		vects[j] = make([]byte, size)
	}
	for j := 0; j < d; j++ {
		rand.Seed(int64(j))
		fillRandom(vects[j])
	}
	x, err := New(d, p)
	if err != nil {
		b.Fatal(err)
	}
	err = x.Encode(vects)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(d * size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = x.Encode(vects)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func benchEncRun(f func(*testing.B, int, int, int), d, p int, size []int) func(*testing.B) {
	return func(b *testing.B) {
		for _, s := range size {
			b.Run(fmt.Sprintf("%d+%d_%d", d, p, s), func(b *testing.B) {
				f(b, d, p, s)
			})
		}
	}
}

func BenchmarkEnc(b *testing.B) {
	s2 := []int{1400, 4 * kb, 64 * kb, mb, 16 * mb}
	b.Run("", benchEncRun(benchEnc, testNumIn, testNumOut, s2))
}

func benchReconstPos(b *testing.B, d, p, size int, has, lost []int) {
	vects := make([][]byte, d+p)
	for j := 0; j < d+p; j++ {
		vects[j] = make([]byte, size)
	}
	for j := 0; j < d; j++ {
		rand.Seed(int64(j))
		fillRandom(vects[j])
	}
	e, err := New(d, p)
	if err != nil {
		b.Fatal(err)
	}
	err = e.Encode(vects)
	if err != nil {
		b.Fatal(err)
	}
	err = e.Reconst(vects, has, lost)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(d * size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = e.Reconst(vects, has, lost)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func benchReconstPosRun(f func(*testing.B, int, int, int, []int, []int), d, p int, size,
	has, lost []int) func(*testing.B) {
	return func(b *testing.B) {
		for _, s := range size {
			b.Run(fmt.Sprintf("%dx%d_%d", d, p, s), func(b *testing.B) {
				f(b, d, p, s, has, lost)
			})
		}
	}
}

func BenchmarkReconstWithPos(b *testing.B) {
	has := []int{0, 1, 3, 5, 6, 8, 10, 11, 12, 13}
	lost := []int{2, 4, 7, 9}
	size := []int{1400, 4 * kb, 64 * kb, mb, 16 * mb}
	b.Run("", benchReconstPosRun(benchReconstPos, testNumIn, testNumOut, size, has, lost))
}

// TODO benchmarkReconstOne
