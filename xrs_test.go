// Copyright (c) 2017 Temple3x (temple3x@gmail.com)
//
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package xrs

import (
	"bytes"
	"fmt"
	"math/rand"
	"sort"
	"testing"
	"time"
)

const (
	kb            = 1 << 10
	mb            = 1 << 20
	testDataNum   = 12
	testParityNum = 4
	testSize      = 1024
)

// We need the result to be as same as old one.
func TestMakeXORSet(t *testing.T) {
	for d := 1; d <= 255; d++ {
		for p := 2; p <= 255; p++ {
			if d+p > 256 {
				continue
			}

			xs1 := make(map[int][]int)
			makeXORSet(d, p, xs1)
			xs2 := make(map[int][]int)
			makeXORSetOld(d, p, xs2)

			if len(xs1) != len(xs2) {
				t.Fatal("mismatch map len", d, p, xs1, xs2)
			}
			for k, v1 := range xs1 {
				v2 := xs2[k]
				if len(v1) != len(v2) {
					t.Fatal("mismatch len")
				}
				for j, k := range v1 {
					if k != v2[j] {
						t.Fatal("element mismatch")
					}
				}
			}

		}
	}
}

// makeXORSetOld is the old implementation.
func makeXORSetOld(d, p int, m map[int][]int) {
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

// Powered by MATLAB
func TestXRS_Encode(t *testing.T) {
	d, p := 5, 5
	x, err := New(d, p)
	if err != nil {
		t.Fatal(err)
	}
	vects := [][]byte{{0, 0}, {4, 7}, {2, 4}, {6, 9}, {8, 11},
		{0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}}
	err = x.Encode(vects)
	if err != nil {
		t.Fatal(err)
	}
	exp := [][]byte{{0, 0}, {4, 7}, {2, 4}, {6, 9}, {8, 11},
		{97, 156}, {173, 117}, {218, 110}, {107, 59}, {110, 153}}

	for i := range exp {
		if !bytes.Equal(exp[i], vects[i]) {
			t.Fatalf("encode failed: vect %d mismatch", i)
		}
	}
}

func TestXRS_GetNeedVects(t *testing.T) {
	for d := 1; d <= 255; d++ {
		for p := 2; p <= 255; p++ {
			if d+p > 256 {
				continue
			}

			xrs, err := New(d, p)
			if err != nil {
				t.Fatal(err)
			}

			for i := 0; i < d; i++ {
				a, b, err := xrs.GetNeedVects(i)
				if err != nil {
					t.Fatal(err)
				}

				a = append(a, i)
				expA := xrs.XORSet[b[1]]
				if len(a) != len(expA) {
					t.Fatal("mismatch len")
				}
				sort.Ints(a)
				for j, k := range a {
					if k != expA[j] {
						t.Fatal("element mismatch")
					}
				}
			}
		}
	}
}

func TestXRS_ReconstOne(t *testing.T) {
	testReconstOne(t, testDataNum, testParityNum, 2)
}

func testReconstOne(t *testing.T, d, p, size int) {
	rand.Seed(time.Now().UnixNano())

	for lost := 0; lost < d; lost++ {

		// init expect, result
		expect := make([][]byte, d+p)
		result := make([][]byte, d+p)
		for j := 0; j < d+p; j++ {
			expect[j] = make([]byte, size)
			result[j] = make([]byte, size)
		}
		for j := 0; j < d; j++ {
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

		// Clean all data except needed.
		// Clean needReconst.
		needReconst := lost
		result[needReconst] = make([]byte, size)
		// Clean A & B.
		aVects, bVects := make([][]byte, d+p), make([][]byte, d+p)
		half := size / 2
		for j := range result {
			aVects[j], bVects[j] = result[j][:half], result[j][half:]
		}
		aNeed, bNeed, err := x.GetNeedVects(needReconst)
		if err != nil {
			t.Fatal(err)
		}
		// Clean A.
		for j := range result {
			if !isIn(j, aNeed) {
				aVects[j] = make([]byte, half)
			}
		}
		// Clean B.
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

func fillRandom(p []byte) {
	rand.Read(p)
}

func TestXRS_retrieveRS(t *testing.T) {
	d, p := testDataNum, testParityNum
	x, err := New(d, p)
	if err != nil {
		t.Fatal(err)
	}

	rand.Seed(time.Now().UnixNano())

	vects := make([][]byte, d+p)
	results := make([][]byte, d+p)
	for i := range vects {
		vects[i] = make([]byte, testSize)
		results[i] = make([]byte, testSize)
		fillRandom(vects[i])
		copy(results[i], vects[i])
	}

	err = x.retrieveRS(results, rand.Perm(d+p))
	if err != nil {
		t.Fatal(err)
	}
	err = x.retrieveRS(results, rand.Perm(d+p))
	if err != nil {
		t.Fatal(err)
	}

	for i := range vects {
		if !bytes.Equal(vects[i], results[i]) {
			t.Fatalf("mismatch retrieveRS; vect: %d", i)
		}
	}
}

func TestXRS_Reconst(t *testing.T) {
	testReconst(t, testDataNum, testParityNum, testSize, 128)
}

func testReconst(t *testing.T, d, p, size, loop int) {

	rand.Seed(time.Now().UnixNano())

	for i := 0; i < loop; i++ {
		exp := make([][]byte, d+p)
		act := make([][]byte, d+p)
		for j := 0; j < d+p; j++ {
			exp[j], act[j] = make([]byte, size), make([]byte, size)
		}
		for j := 0; j < d; j++ {
			fillRandom(exp[j])
		}

		x, err := New(d, p)
		if err != nil {
			t.Fatal(err)
		}
		err = x.Encode(exp)
		if err != nil {
			t.Fatal(err)
		}

		lost := makeLostRandom(d+p, rand.Intn(p+1))
		needReconst := lost[:rand.Intn(len(lost)+1)]
		if len(needReconst) == 1 {
			lost = needReconst // Make sure to have correct data for reconstOne.
		}
		dpHas := makeHasFromLost(d+p, lost)
		for _, h := range dpHas {
			copy(act[h], exp[h])
		}

		// Try to reconstruct some health vectors.
		// Although we want to reconstruct these vectors,
		// but it maybe a mistake.
		for _, nr := range needReconst {
			if rand.Intn(4) == 0 { // 1/4 chance.
				copy(act[nr], exp[nr])
			}
		}

		err = x.Reconst(act, dpHas, needReconst)
		if err != nil {
			t.Fatal(err)
		}

		for _, n := range needReconst {
			if !bytes.Equal(exp[n], act[n]) {
				t.Fatalf("reconst failed: vect: %d, size: %d", n, size)
			}
		}
	}
}

func TestXRS_Update(t *testing.T) {
	testUpdate(t, testDataNum, testParityNum, testSize)
}

func testUpdate(t *testing.T, d, p, size int) {

	rand.Seed(time.Now().UnixNano())

	for i := 0; i < d; i++ {
		act := make([][]byte, d+p)
		exp := make([][]byte, d+p)
		for j := 0; j < d+p; j++ {
			act[j], exp[j] = make([]byte, size), make([]byte, size)
		}
		for j := 0; j < d; j++ {
			fillRandom(exp[j])
			copy(act[j], exp[j])
		}

		x, err := New(d, p)
		if err != nil {
			t.Fatal(err)
		}
		err = x.Encode(act)
		if err != nil {
			t.Fatal(err)
		}

		newData := make([]byte, size)
		fillRandom(newData)
		updateRow := i
		err = x.Update(act[updateRow], newData, updateRow, act[d:d+p])
		if err != nil {
			t.Fatal(err)
		}

		copy(exp[updateRow], newData)
		err = x.Encode(exp)
		if err != nil {
			t.Fatal(err)
		}
		for j := d; j < d+p; j++ {
			if !bytes.Equal(act[j], exp[j]) {
				t.Fatalf("update failed: vect: %d, size: %d", j, size)
			}
		}
	}
}

func TestXRS_Replace(t *testing.T) {
	testReplace(t, testDataNum, testParityNum, testSize, 1024, true)
	testReplace(t, testDataNum, testParityNum, testSize, 1024, false)
}

func testReplace(t *testing.T, d, p, size, loop int, toZero bool) {

	rand.Seed(time.Now().UnixNano())

	for i := 0; i < loop; i++ {
		replaceRows := makeReplaceRowRandom(d)
		act := make([][]byte, d+p)
		exp := make([][]byte, d+p)
		for j := 0; j < d+p; j++ {
			act[j], exp[j] = make([]byte, size), make([]byte, size)
		}
		for j := 0; j < d; j++ {
			fillRandom(exp[j])
			copy(act[j], exp[j])
		}

		data := make([][]byte, len(replaceRows))
		for i, rr := range replaceRows {
			data[i] = make([]byte, size)
			copy(data[i], exp[rr])
		}

		if toZero {
			for _, rr := range replaceRows {
				exp[rr] = make([]byte, size)
			}
		}

		x, err := New(d, p)
		if err != nil {
			t.Fatal(err)
		}
		err = x.Encode(exp)
		if err != nil {
			t.Fatal(err)
		}

		if !toZero {
			for _, rr := range replaceRows {
				act[rr] = make([]byte, size)
			}
		}
		err = x.Encode(act)
		if err != nil {
			t.Fatal(err)
		}

		err = x.Replace(data, replaceRows, act[d:])
		if err != nil {
			t.Fatal(err)
		}

		for j := d; j < d+p; j++ {
			if !bytes.Equal(act[j], exp[j]) {
				fmt.Println(replaceRows)
				t.Fatalf("replace failed: vect: %d, size: %d", j, size)
			}
		}

	}
}

func makeReplaceRowRandom(d int) []int {
	rand.Seed(time.Now().UnixNano())

	n := rand.Intn(d + 1)
	s := make([]int, 0)
	c := 0
	for i := 0; i < 64; i++ {
		if c == n {
			break
		}
		v := rand.Intn(d)
		if !isIn(v, s) {
			s = append(s, v)
			c++
		}
	}
	if c == 0 {
		s = []int{0}
	}
	return s
}

func makeLostRandom(n, lostN int) []int {
	l := make([]int, lostN)
	rand.Seed(time.Now().UnixNano())
	c := 0
	for {
		if c == lostN {
			break
		}
		v := rand.Intn(n)
		if !isIn(v, l) {
			l[c] = v
			c++
		}
	}
	return l
}

func makeHasFromLost(n int, lost []int) []int {
	s := make([]int, n-len(lost))
	c := 0
	for i := 0; i < n; i++ {
		if !isIn(i, lost) {
			s[c] = i
			c++
		}
	}
	return s
}

func BenchmarkXRS_Encode(b *testing.B) {
	dps := [][]int{
		[]int{12, 4},
	}

	sizes := []int{
		4 * kb,
		mb,
		8 * mb,
	}

	b.Run("", benchmarkEncode(benchEnc, dps, sizes))
}

func benchmarkEncode(f func(*testing.B, int, int, int), dps [][]int, sizes []int) func(*testing.B) {
	return func(b *testing.B) {
		for _, dp := range dps {
			d, p := dp[0], dp[1]
			for _, size := range sizes {
				b.Run(fmt.Sprintf("(%d+%d)-%s", d, p, byteToStr(size)), func(b *testing.B) {
					f(b, d, p, size)
				})
			}
		}
	}
}

func benchEnc(b *testing.B, d, p, size int) {

	vects := make([][]byte, d+p)
	for j := 0; j < d+p; j++ {
		vects[j] = make([]byte, size)
	}
	for j := 0; j < d; j++ {
		fillRandom(vects[j])
	}
	x, err := New(d, p)
	if err != nil {
		b.Fatal(err)
	}

	b.SetBytes(int64((d + p) * size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = x.Encode(vects)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkXRS_Reconst(b *testing.B) {
	d, p := 12, 4
	size := 4 * kb

	b.Run("", benchmarkReconst(benchReconst, d, p, size))
}

func benchmarkReconst(f func(*testing.B, int, int, int, []int, []int), d, p, size int) func(*testing.B) {

	datas := make([]int, d)
	for i := range datas {
		datas[i] = i
	}
	return func(b *testing.B) {
		for i := 1; i <= p; i++ {
			lost := datas[:i]
			dpHas := makeHasFromLost(d+p, lost)
			b.Run(fmt.Sprintf("(%d+%d)-%s-reconst_%d_data_vects",
				d, p, byteToStr(size), i),
				func(b *testing.B) { f(b, d, p, size, dpHas, lost) })
		}
	}
}

func benchReconst(b *testing.B, d, p, size int, dpHas, needReconst []int) {
	vects := make([][]byte, d+p)
	for j := 0; j < d+p; j++ {
		vects[j] = make([]byte, size)
	}
	for j := 0; j < d; j++ {
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

	bs := (d + len(needReconst)) * size
	if len(needReconst) == 1 {
		aNeed, _, err := x.GetNeedVects(needReconst[0])
		if err != nil {
			b.Fatal(err)
		}
		bs = (d-1+2+len(aNeed))*size/2 + size
	}

	b.SetBytes(int64(bs))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = x.Reconst(vects, dpHas, needReconst)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkXRS_Update(b *testing.B) {
	d, p := 12, 4
	size := 4 * kb

	b.Run("", benchmarkUpdate(benchUpdate, d, p, size))
}

func benchmarkUpdate(f func(*testing.B, int, int, int, int), d, p, size int) func(*testing.B) {

	return func(b *testing.B) {
		updateRow := rand.Intn(d)
		b.Run(fmt.Sprintf("(%d+%d)-%s",
			d, p, byteToStr(size)),
			func(b *testing.B) { f(b, d, p, size, updateRow) })
	}
}

func benchUpdate(b *testing.B, d, p, size, updateRow int) {
	vects := make([][]byte, d+p)
	for j := 0; j < d+p; j++ {
		vects[j] = make([]byte, size)
	}
	for j := 0; j < d; j++ {
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

	newData := make([]byte, size)
	fillRandom(newData)

	b.SetBytes(int64((p + 2 + p) * size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = x.Update(vects[updateRow], newData, updateRow, vects[d:])
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkXRS_Replace(b *testing.B) {
	d, p := 12, 4
	size := 4 * kb

	b.Run("", benchmarkReplace(benchReplace, d, p, size))
}

func benchmarkReplace(f func(*testing.B, int, int, int, int), d, p, size int) func(*testing.B) {

	return func(b *testing.B) {
		for i := 1; i <= d-p; i++ {
			b.Run(fmt.Sprintf("(%d+%d)-%s-replace_%d_data_vects",
				d, p, byteToStr(size), i),
				func(b *testing.B) { f(b, d, p, size, i) })
		}
	}
}

func benchReplace(b *testing.B, d, p, size, n int) {
	vects := make([][]byte, d+p)
	for j := 0; j < d+p; j++ {
		vects[j] = make([]byte, size)
	}
	for j := 0; j < d; j++ {
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

	updateRows := make([]int, n)
	for i := range updateRows {
		updateRows[i] = i
	}
	b.SetBytes(int64((n + p + p) * size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = x.Replace(vects[:n], updateRows, vects[d:])
		if err != nil {
			b.Fatal(err)
		}
	}
}

func byteToStr(n int) string {
	if n >= mb {
		return fmt.Sprintf("%dMB", n/mb)
	}

	return fmt.Sprintf("%dKB", n/kb)
}
