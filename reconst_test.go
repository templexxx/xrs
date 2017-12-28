package xrs

import (
	"bytes"
	"math/rand"
	"testing"
)

func TestVerifyReconst(t *testing.T) {
	pig, err := New(10, 4)
	if err != nil {
		t.Fatal(err)
	}
	dp := [][]byte{
		{0, 13, 12, 1},
		{4, 14, 14, 5},
		{2, 17, 19, 7},
		{6, 23, 32, 7},
		{21, 24, 25, 23},
		{33, 36, 26, 35},
		{44, 27, 37, 47},
		{11, 42, 43, 16},
		{13, 101, 103, 46},
		{98, 177, 186, 65},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	}
	err = pig.Encode(dp)
	if err != nil {
		t.Fatal(err)
	}
	dp[0] = make([]byte, 4)
	dp[4] = make([]byte, 4)
	dp[11] = make([]byte, 4)
	dp[12] = make([]byte, 4)
	// Reconstruct with all dp present
	// 4 dp "missing"
	lost := []int{0, 4, 11, 12}
	err = pig.Reconst(dp, lost, true)
	if err != nil {
		t.Fatal(err)
	}
	if dp[10][0] != 45 || dp[10][1] != 169 || dp[10][2] != 230 || dp[10][3] != 199 {
		t.Fatal("shard 10 mismatch")
	}
	if dp[11][0] != 210 || dp[11][1] != 140 || dp[11][2] != 33 || dp[11][3] != 163 {
		t.Fatal("shard 11 mismatch")
	}
	if dp[12][0] != 39 || dp[12][1] != 232 || dp[12][2] != 31 || dp[12][3] != 59 {
		t.Fatal("shard 12 mismatch")
	}
	if dp[13][0] != 87 || dp[13][1] != 43 || dp[13][2] != 234 || dp[13][3] != 184 {
		t.Fatal("shard 13 mismatch")
	}
}

//TODO add repair 1 test

func TestReconst(t *testing.T) {
	size := 64 * 1024
	pig, err := New(10, 4)
	if err != nil {
		t.Fatal(err)
	}
	dp := NewMatrix(14, size)
	rand.Seed(0)
	for s := 0; s < 10; s++ {
		fillRandom(dp[s])
	}
	err = pig.Encode(dp)
	if err != nil {
		t.Fatal(err)
	}
	// restore encode result
	store := NewMatrix(4, size)
	copy(store[0], dp[0])
	copy(store[1], dp[4])
	copy(store[2], dp[11])
	copy(store[3], dp[12])
	dp[0] = make([]byte, size)
	dp[4] = make([]byte, size)
	dp[11] = make([]byte, size)
	dp[12] = make([]byte, size)
	// Reconstruct with all dp present
	var lost []int
	err = pig.Reconst(dp, lost, true)
	if err != nil {
		t.Fatal(err)
	}
	// 4 dp "missing"
	lost = []int{0, 4, 11, 12}
	err = pig.Reconst(dp, lost, true)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(store[0], dp[0]) {
		t.Fatal("reconst data mismatch: dp[0]")
	}
	if !bytes.Equal(store[1], dp[4]) {
		t.Fatal("reconst data mismatch: dp[4]")
	}
	if !bytes.Equal(store[2], dp[11]) {
		t.Fatal("reconst data mismatch: dp[11]")
	}
	if !bytes.Equal(store[3], dp[12]) {
		t.Fatal("reconst data mismatch: dp[12]")
	}
	// Reconstruct with 9 dp present (should fail)
	lost = append(lost, 11)
	err = pig.Reconst(dp, lost, true)
	if err != ErrTooFewShards {
		t.Errorf("expected %v, got %v", ErrTooFewShards, err)
	}
}

func BenchmarkReconst10x4x16MRepair1(b *testing.B) {
	benchmarkReconst(b, 10, 4, 16*1024*1024, 1)
}

func BenchmarkReconst10x4x16MRepair1Data(b *testing.B) {
	benchmarkReconstOneData(b, 10, 4, 16*1024*1024)
}

func BenchmarkReconst10x4x16MRepair2(b *testing.B) {
	benchmarkReconst(b, 10, 4, 16*1024*1024, 2)
}

func BenchmarkReconst10x4x16MRepair3(b *testing.B) {
	benchmarkReconst(b, 10, 4, 16*1024*1024, 3)
}

func BenchmarkReconst10x4x16MRepair4(b *testing.B) {
	benchmarkReconst(b, 10, 4, 16*1024*1024, 4)
}

func BenchmarkReconst28x4x16MRepair1(b *testing.B) {
	benchmarkReconst(b, 28, 4, 16*1024*1024, 1)
}

func BenchmarkReconst28x4x16MRepair1Data(b *testing.B) {
	benchmarkReconstOneData(b, 28, 4, 16*1024*1024)
}

func BenchmarkReconst28x4x16MRepair2(b *testing.B) {
	benchmarkReconst(b, 28, 4, 16*1024*1024, 2)
}

func BenchmarkReconst28x4x16MRepair3(b *testing.B) {
	benchmarkReconst(b, 28, 4, 16*1024*1024, 3)
}

func BenchmarkReconst28x4x16MRepair4(b *testing.B) {
	benchmarkReconst(b, 28, 4, 16*1024*1024, 4)
}

func BenchmarkReconst14x10x16MRepair1(b *testing.B) {
	benchmarkReconst(b, 14, 10, 16*1024*1024, 1)
}

func BenchmarkReconst14x10x16MRepair1Data(b *testing.B) {
	benchmarkReconstOneData(b, 14, 10, 16*1024*1024)
}

func BenchmarkReconst14x10x16MRepair2(b *testing.B) {
	benchmarkReconst(b, 14, 10, 16*1024*1024, 2)
}

func BenchmarkReconst14x10x16MRepair3(b *testing.B) {
	benchmarkReconst(b, 14, 10, 16*1024*1024, 3)
}

func BenchmarkReconst14x10x16MRepair4(b *testing.B) {
	benchmarkReconst(b, 14, 10, 16*1024*1024, 4)
}

func benchmarkReconst(b *testing.B, d, p, size, repair int) {
	pig, err := New(d, p)
	if err != nil {
		b.Fatal(err)
	}
	dp := NewMatrix(d+p, size)
	rand.Seed(0)
	for s := 0; s < d; s++ {
		fillRandom(dp[s])
	}
	err = pig.Encode(dp)
	if err != nil {
		b.Fatal(err)
	}
	var lost []int
	if repair == 1 {
		r := rand.Intn(p)+d
		lost = append(lost, r)
	} else {
		for i := 1; i <= repair; i++ {
			r := rand.Intn(d + p)
			if !inSlice(lost, r) {
				lost = append(lost, r)
			}
		}
	}
	for _, l := range lost {
		dp[l] = make([]byte, size)
	}
	b.SetBytes(int64(size * d))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pig.Reconst(dp, lost, true)
	}
}

func benchmarkReconstOneData(b *testing.B, d, p, size int) {
	pig, err := New(d, p)
	if err != nil {
		b.Fatal(err)
	}
	dp := NewMatrix(d+p, size)
	rand.Seed(0)
	for s := 0; s < d; s++ {
		fillRandom(dp[s])
	}
	err = pig.Encode(dp)
	if err != nil {
		b.Fatal(err)
	}
	var lost []int
	r := rand.Intn(d)
	lost = append(lost, r)
	dp[r] = make([]byte, size)
	lenXor := d
	for _, v := range pig.xMap {
		if inSlice(v, r) {
			lenXor = lenXor + len(v)
		}
	}
	b.SetBytes(int64((size/2) * lenXor))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pig.Reconst(dp, lost, true)
	}
}
