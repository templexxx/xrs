package xrs

import (
	"bytes"
	crand "crypto/rand"
	"fmt"
	"io"
	"math/rand"
	"testing"
	"time"
)

const (
	kb            = 1 << 10
	mb            = 1 << 20
	testDataCnt   = 10
	testParityCnt = 4
	// 128: avx_loop/sse_loop(RS&xor), 16: xmm_register(RS&xor)/general_register(xor), 8:general_register(xor), 1: byte by byte(RS&xor)
	// more details: RS/encode.go, RS/rs_amd64.s xor/xor_amd64.s
	verifySize = 256 + 32 + 16 + 8 + 2
)

var testUpdateRow = 0

// Powered by MATLAB
func TestVerifyEncodeBase(t *testing.T) {
	d, p := 5, 5
	x, err := New(d, p)
	if err != nil {
		t.Fatal(err)
	}
	vects := [][]byte{{0, 0}, {4, 7}, {2, 4}, {6, 9}, {8, 11}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}}
	err = x.Encode(vects)
	if err != nil {
		t.Fatal(err)
	}

	if vects[5][0] != 97 || vects[5][1] != 156 {
		t.Fatal("vect 5 mismatch")
	}
	if vects[6][0] != 173 || vects[6][1] != 117 {
		t.Fatal("vect 6 mismatch")
	}
	if vects[7][0] != 218 || vects[7][1] != 110 {
		t.Fatal("vect 7 mismatch")
	}
	if vects[8][0] != 107 || vects[8][1] != 59 {
		t.Fatal("vect 8 mismatch")
	}
	if vects[9][0] != 110 || vects[9][1] != 153 {
		t.Fatal("vect 9 mismatch")
	}
}

func TestVerifyReconst(t *testing.T) {
	verifyReconst(t, testDataCnt, testParityCnt)
}

func verifyReconst(t *testing.T, d, p int) {
	for size := 2; size <= verifySize; size += 2 {
		expect := make([][]byte, d+p)
		result := make([][]byte, d+p)
		for j := 0; j < d+p; j++ {
			expect[j] = make([]byte, size)
			result[j] = make([]byte, size)
		}
		for j := 0; j < d; j++ {
			err := fillRandom(expect[j])
			if err != nil {
				t.Fatal(err)
			}
		}
		x, err := New(d, p)
		if err != nil {
			t.Fatal(err)
		}
		err = x.Encode(expect)
		if err != nil {
			t.Fatal(err)
		}
		needReconst := makeLost(d, p)
		dpHas := makeDPHas(d, p, needReconst)
		for _, h := range dpHas {
			copy(result[h], expect[h])
		}
		err = x.Reconst(result, dpHas, needReconst)
		if err != nil {
			t.Fatal(err)
		}
		for _, n := range needReconst {
			if !bytes.Equal(expect[n], result[n]) {
				t.Fatalf("no match reconst; vect: %d; size: %d", n, size)
			}
		}
	}
}

// reconst part of lost
func TestVerifyReconstPart(t *testing.T) {
	d, p := 5, 3
	for i := 0; i < 1024; i++ {
		lost := makeLost(d, p)
		for j := 0; j <= p; j++ {
			for _, l := range lost[:j] {
				testVerifyReconstPart(t, d, p, lost[:j], l)
			}
		}
	}
}

func testVerifyReconstPart(t *testing.T, d, p int, lost []int, needReconst int) {
	size := 2
	expect := make([][]byte, d+p)
	result := make([][]byte, d+p)
	for j := 0; j < d+p; j++ {
		expect[j] = make([]byte, size)
		result[j] = make([]byte, size)
	}
	for j := 0; j < d; j++ {
		err := fillRandom(expect[j])
		if err != nil {
			t.Fatal(err)
		}
	}
	x, err := New(d, p)
	if err != nil {
		t.Fatal(err)
	}
	err = x.Encode(expect)
	if err != nil {
		t.Fatal(err)
	}
	dpHas := makeDPHas(d, p, lost)
	for _, h := range dpHas {
		copy(result[h], expect[h])
	}
	err = x.Reconst(result, dpHas, []int{needReconst})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(expect[needReconst], result[needReconst]) {
		t.Fatal("lost:", lost, "needReconst:", needReconst)
	}

}

func fillRandom(v []byte) (err error) {
	_, err = io.ReadFull(crand.Reader, v)
	return
}

func makeDPHas(dataCnt, parityCnt int, needReconst []int) []int {
	dpHas := make([]int, dataCnt)
	for i := range dpHas {
		for j := 0; j < dataCnt+parityCnt; j++ {
			if !isIn(j, needReconst) {
				if !isIn(j, dpHas) {
					dpHas[i] = j
				}
			}
		}
	}
	return dpHas
}

func makeLost(dataCnt, parityCnt int) []int {
	needReconst := make([]int, parityCnt)
	off := 0
	rand.Seed(time.Now().UnixNano())
	for {
		if off == parityCnt-1 {
			break
		}
		n := rand.Intn(dataCnt + parityCnt)
		if !isIn(n, needReconst) {
			needReconst[off] = n
			off++
		}
	}
	return needReconst
}

func TestVerifyReconstOne(t *testing.T) {
	verifyReconstOne(t, testDataCnt, testParityCnt)
}

func verifyReconstOne(t *testing.T, d, p int) {
	for size := 2; size <= verifySize; size += 2 {
		// init expect, result
		expect := make([][]byte, d+p)
		result := make([][]byte, d+p)
		for j := 0; j < d+p; j++ {
			expect[j] = make([]byte, size)
			result[j] = make([]byte, size)
		}
		for j := 0; j < d; j++ {
			rand.Seed(int64(j))
			fillRandom(expect[j])
		}
		x, err := New(d, p)
		if err != nil {
			t.Fatal(err)
		}
		err = x.Encode(expect)
		if err != nil {
			t.Fatal(err)
		}
		for j := 0; j < d+p; j++ {
			copy(result[j], expect[j])
		}
		// clean needReconst
		needReconst := rand.Intn(d)
		result[needReconst] = make([]byte, size)
		// init a&b Vects
		aVects, bVects := make([][]byte, d+p), make([][]byte, d+p)
		half := size / 2
		for j := range result {
			aVects[j], bVects[j] = result[j][:half], result[j][half:]
		}
		aNeed, bNeed, err := x.GetNeedVects(needReconst)
		if err != nil {
			t.Fatal(err)
		}
		// clean A
		for j := range result {
			if !isIn(j, aNeed) {
				aVects[j] = make([]byte, half)
			}
		}
		// clean B
		bVects[needReconst] = make([]byte, size)
		for j := d; j < d+p; j++ {
			if !isIn(j, bNeed) {
				bVects[j] = make([]byte, half)
			}
		}
		for j := range result {
			copy(result[j][:half], aVects[j])
			copy(result[j][half:], bVects[j])

		}
		err = x.ReconstOne(result, needReconst)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(result[needReconst], expect[needReconst]) {
			t.Fatalf("mismatch reconstOne; vect: %d; size: %d", needReconst, size)
		}
	}
}

func TestVerifyUpdate(t *testing.T) {
	verifyUpdate(t, testDataCnt, testParityCnt, testUpdateRow)
}

// compare encode&update results
func verifyUpdate(t *testing.T, d, p, updateRow int) {
	for size := 2; size <= verifySize; size += 2 {
		updateRet := make([][]byte, d+p)
		encodeRet := make([][]byte, d+p)
		for j := 0; j < d+p; j++ {
			updateRet[j], encodeRet[j] = make([]byte, size), make([]byte, size)
		}
		for j := 0; j < d; j++ {
			err := fillRandom(encodeRet[j])
			if err != nil {
				t.Fatal(err)
			}
			copy(updateRet[j], encodeRet[j])
		}
		x, err := New(d, p)
		if err != nil {
			t.Fatal(err)
		}
		err = x.Encode(updateRet)
		if err != nil {
			t.Fatal(err)
		}
		oldData, newData := make([]byte, size), make([]byte, size)
		oldData = updateRet[updateRow]
		err = fillRandom(newData)
		if err != nil {
			t.Fatal(err)
		}
		err = x.Update(oldData, newData, updateRow, updateRet[d:])
		if err != nil {
			t.Fatal(err)
		}
		encodeRet[updateRow] = newData
		err = x.Encode(encodeRet)
		if err != nil {
			t.Fatal(err)
		}
		for j := d; j < d+p; j++ {
			if !bytes.Equal(updateRet[j], encodeRet[j]) {
				t.Fatalf("update mismatch; vect: %d; size: %d", j, size)
			}
		}
	}
}

func BenchmarkEncode(b *testing.B) {
	sizes := []int{4 * kb, 64 * kb, mb}
	b.Run("", benchEncRun(benchEnc, testDataCnt, testParityCnt, sizes))
}

func benchEncRun(f func(*testing.B, int, int, int), d, p int, sizes []int) func(*testing.B) {
	return func(b *testing.B) {
		for _, s := range sizes {
			b.Run(fmt.Sprintf("(%d+%d)*%dKB", d, p, s/kb), func(b *testing.B) {
				f(b, d, p, s)
			})
		}
	}
}

func benchEnc(b *testing.B, d, p, size int) {
	vects := make([][]byte, d+p)
	for j := 0; j < d+p; j++ {
		vects[j] = make([]byte, size)
	}
	for j := 0; j < d; j++ {
		err := fillRandom(vects[j])
		if err != nil {
			b.Fatal(err)
		}
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

func BenchmarkReconst(b *testing.B) {
	sizes := []int{4 * mb}
	needReconst := makeLost(testDataCnt, testParityCnt)
	dpHas := makeDPHas(testDataCnt, testParityCnt, needReconst)
	b.Run("", benchmarkReconst(benchReconst, testDataCnt, testParityCnt, sizes, dpHas, needReconst))
}

func benchmarkReconst(f func(*testing.B, int, int, int, []int, []int),
	d, p int, sizes, dpHas, needReconst []int) func(*testing.B) {
	return func(b *testing.B) {
		for _, s := range sizes {
			b.Run(fmt.Sprintf("(%d+%d)*%dKB", d, p, s/kb), func(b *testing.B) {
				f(b, d, p, s, dpHas, needReconst)
			})
		}
	}
}

func benchReconst(b *testing.B, d, p, size int, dpHas, needReconst []int) {
	vects := make([][]byte, d+p)
	for j := 0; j < d+p; j++ {
		vects[j] = make([]byte, size)
	}
	for j := 0; j < d; j++ {
		err := fillRandom(vects[j])
		if err != nil {
			b.Fatal(err)
		}
	}
	x, err := New(d, p)
	if err != nil {
		b.Fatal(err)
	}
	err = x.Encode(vects)
	if err != nil {
		b.Fatal(err)
	}
	err = x.Reconst(vects, dpHas, needReconst)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(d * size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = x.Reconst(vects, dpHas, needReconst)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReconstOne(b *testing.B) {
	sizes := []int{4 * mb}
	needReconst := rand.Intn(testDataCnt)
	b.Run("", benchmarkReconstOne(benchReconstOne, testDataCnt, testParityCnt, sizes, needReconst))
}

func benchReconstOne(b *testing.B, d, p, size, needReconst int) {
	vects := make([][]byte, d+p)
	for j := 0; j < d+p; j++ {
		vects[j] = make([]byte, size)
	}
	for j := 0; j < d; j++ {
		err := fillRandom(vects[j])
		if err != nil {
			b.Fatal(err)
		}
	}
	x, err := New(d, p)
	if err != nil {
		b.Fatal(err)
	}
	err = x.Encode(vects)
	if err != nil {
		b.Fatal(err)
	}
	err = x.ReconstOne(vects, needReconst)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(d * size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = x.ReconstOne(vects, needReconst)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkReconstOne(f func(*testing.B, int, int, int, int), d, p int, sizes []int, lost int) func(*testing.B) {
	return func(b *testing.B) {
		for _, s := range sizes {
			b.Run(fmt.Sprintf("(%d+%d)*%dKB", d, p, s/kb), func(b *testing.B) {
				f(b, d, p, s, lost)
			})
		}
	}
}
