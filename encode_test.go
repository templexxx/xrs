package xrs

import (
	"math/rand"
	"testing"
)

func TestEncode(t *testing.T) {
	size := 3334
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
	badDP := NewMatrix(14, 100)
	badDP[0] = make([]byte, 1)
	err = pig.Encode(badDP)
	if err != ErrDataShardsOdd {
		t.Errorf("expected %v, got %v", ErrDataShardsOdd, err)
	}
}

func BenchmarkEncode10x4x16M(b *testing.B) {
	benchmarkEncode(b, 10, 4, 16*1024*1024)
}

func benchmarkEncode(b *testing.B, data, parity, size int) {
	pig, err := New(data, parity)
	if err != nil {
		b.Fatal(err)
	}
	dp := NewMatrix(data+parity, size)
	rand.Seed(0)
	for i := 0; i < data; i++ {
		fillRandom(dp[i])
	}
	b.SetBytes(int64(size * data))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = pig.Encode(dp)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// test low, high table work
func TestVerifyEncode(t *testing.T) {
	pig, err := New(10, 4)
	if err != nil {
		t.Fatal(err)
	}
	shards := [][]byte{
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
	err = pig.Encode(shards)
	if err != nil {
		t.Fatal(err)
	}
	if shards[10][0] != 45 || shards[10][1] != 169 || shards[10][2] != 230 || shards[10][3] != 199 {
		t.Fatal("shard 10 mismatch")
	}
	if shards[11][0] != 210 || shards[11][1] != 140 || shards[11][2] != 33 || shards[11][3] != 163 {
		t.Fatal("shard 11 mismatch")
	}
	if shards[12][0] != 39 || shards[12][1] != 232 || shards[12][2] != 31 || shards[12][3] != 59 {
		t.Fatal("shard 12 mismatch")
	}
	if shards[13][0] != 87 || shards[13][1] != 43 || shards[13][2] != 234 || shards[13][3] != 184 {
		t.Fatal("shard 13 mismatch")
	}
}

//
//func TestSSSE3(t *testing.T) {
//	d := 10
//	pig := 4
//	size := 65 * 1024
//	pig, err := New(d, pig)
//	if err != nil {
//		t.Fatal(err)
//	}
//	pig.ins = ssse3
//	// asm
//	dp := NewMatrix(d+pig, size)
//	rand.Seed(0)
//	for i := 0; i < d; i++ {
//		fillRandom(dp[i])
//	}
//	err = pig.Encode(dp)
//	if err != nil {
//		t.Fatal(err)
//	}
//	// mulTable
//	mDP := NewMatrix(d+pig, size)
//	for i := 0; i < d; i++ {
//		mDP[i] = dp[i]
//	}
//	err = pig.Encode(mDP)
//	if err != nil {
//		t.Fatal(err)
//	}
//	for i, asm := range dp {
//		if !bytes.Equal(asm, mDP[i]) {
//			t.Fatal("verify asm failed, no match noasm version; shards: ", i)
//		}
//	}
//}
//
//func BenchmarkEncode10x4x8K(b *testing.B) {
//	benchmarkEncode(b, 10, 4, 8*1024)
//}
//
//func BenchmarkEncode10x4x16K(b *testing.B) {
//	benchmarkEncode(b, 10, 4, 16*1024)
//}
//
//func BenchmarkEncode10x4x32K(b *testing.B) {
//	benchmarkEncode(b, 10, 4, 32*1024)
//}
//
//func BenchmarkEncode10x4x64K(b *testing.B) {
//	benchmarkEncode(b, 10, 4, 64*1024)
//}
//
//func BenchmarkEncode10x4x128K(b *testing.B) {
//	benchmarkEncode(b, 10, 4, 128*1024)
//}
//
//func BenchmarkEncode10x4x256K(b *testing.B) {
//	benchmarkEncode(b, 10, 4, 256*1024)
//}
//
//func BenchmarkEncode10x4x512K(b *testing.B) {
//	benchmarkEncode(b, 10, 4, 512*1024)
//}
//
//func BenchmarkEncode10x4x1M(b *testing.B) {
//	benchmarkEncode(b, 10, 4, 1024*1024)
//}
//
//func BenchmarkEncode10x4x4M(b *testing.B) {
//	benchmarkEncode(b, 10, 4, 4*1024*1024)
//}
//

//
//func BenchmarkEncode17x3x1M(b *testing.B) {
//	benchmarkEncode(b, 17, 3, 1024*1024)
//}
//
//func BenchmarkEncode17x3x4M(b *testing.B) {
//	benchmarkEncode(b, 17, 3, 4*1024*1024)
//}
//
//func BenchmarkEncode17x3x16M(b *testing.B) {
//	benchmarkEncode(b, 17, 3, 16*1024*1024)
//}
//
//func BenchmarkEncode28x4x1M(b *testing.B) {
//	benchmarkEncode(b, 28, 4, 1024*1024)
//}
//
//func BenchmarkEncode28x4x4M(b *testing.B) {
//	benchmarkEncode(b, 28, 4, 4*1024*1024)
//}
//
//func BenchmarkEncode28x4x16M(b *testing.B) {
//	benchmarkEncode(b, 28, 4, 16*1024*1024)
//}
//
//func BenchmarkEncode14x10x1M(b *testing.B) {
//	benchmarkEncode(b, 14, 10, 1024*1024)
//}
//
//func BenchmarkEncode14x10x4M(b *testing.B) {
//	benchmarkEncode(b, 14, 10, 4*1024*1024)
//}
//
//func BenchmarkEncode14x10x16M(b *testing.B) {
//	benchmarkEncode(b, 14, 10, 16*1024*1024)
//}
//

//
//func BenchmarkSSSE3Encode28x4x16M(b *testing.B) {
//	benchmarkSSSE3Encode(b, 28, 4, 16*1024*1024)
//}
//
//func benchmarkSSSE3Encode(b *testing.B, data, parity, size int) {
//	pig, err := New(data, parity)
//	pig.ins = ssse3
//	if err != nil {
//		b.Fatal(err)
//	}
//	dp := NewMatrix(data+parity, size)
//	rand.Seed(0)
//	for i := 0; i < data; i++ {
//		fillRandom(dp[i])
//	}
//	b.SetBytes(int64(size * data))
//	b.ResetTimer()
//	for i := 0; i < b.N; i++ {
//		pig.Encode(dp)
//	}
//}
//
//func BenchmarkNOASMEncode28x4x16M(b *testing.B) {
//	benchmarkNOASMEncode(b, 28, 4, 16*1024*1024)
//}
//

//
func fillRandom(p []byte) {
	for i := 0; i < len(p); i += 7 {
		val := rand.Int63()
		for j := 0; i+j < len(p) && j < 7; j++ {
			p[i+j] = byte(val)
			val >>= 8
		}
	}
}

//

//
//// test no simd asm
//func TestNoSIMD(t *testing.T) {
//	d := 10
//	pig := 1
//	size := 10
//	pig, err := New(d, pig)
//	if err != nil {
//		t.Fatal(err)
//	}
//	// asm
//	dp := NewMatrix(d+pig, size)
//	rand.Seed(0)
//	for i := 0; i < d; i++ {
//		fillRandom(dp[i])
//	}
//	err = pig.nosimdEncode(dp)
//	if err != nil {
//		t.Fatal(err)
//	}
//	// mulTable
//	mDP := NewMatrix(d+pig, size)
//	for i := 0; i < d; i++ {
//		mDP[i] = dp[i]
//	}
//	err = pig.noasmEncode(mDP)
//	if err != nil {
//		t.Fatal(err)
//	}
//	for i, asm := range dp {
//		if !bytes.Equal(asm, mDP[i]) {
//			t.Fatal("verify simd failed, no match noasm version; shards: ", i)
//		}
//	}
//}
//
//func BenchmarkNOSIMDncode28x4x16M(b *testing.B) {
//	benchmarkNOSIMDEncode(b, 28, 4, 16*1024*1024)
//}
//
//func benchmarkNOSIMDEncode(b *testing.B, data, parity, size int) {
//	pig, err := New(data, parity)
//	if err != nil {
//		b.Fatal(err)
//	}
//	dp := NewMatrix(data+parity, size)
//	rand.Seed(0)
//	for i := 0; i < data; i++ {
//		fillRandom(dp[i])
//	}
//	b.SetBytes(int64(size * data))
//	b.ResetTimer()
//	for i := 0; i < b.N; i++ {
//		pig.nosimdEncode(dp)
//	}
//}
