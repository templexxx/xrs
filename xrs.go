// Copyright (c) 2017 Temple3x (temple3x@gmail.com)
//
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

// Package xrs implements erasure codes based on
// "A Hitchhiker's Guide to Fast and Efficient Data Reconstruction in
// Erasure-coded Data Centers".
//
// XRS splits each row vector into two equal-sized parts.
// Example: 10+4:
// +---------+
// | a1 | b1 |
// +---------+
// | a2 | b2 |
// +---------+
// | a3 | b3 |
// +---------+
//
//	...
//
// +---------+
// | a10| b10|
// +---------+
// | a11| b11|
// +---------+
// | a12| b12|
// +---------+
// | a13| b13|
// +---------+
package xrs

import (
	"errors"
	"fmt"

	rs "github.com/templexxx/reedsolomon"
	xor "github.com/templexxx/xorsimd"
)

// XRS is the X-Reed-Solomon codec.
type XRS struct {
	// RS is the backend Reed-Solomon codec.
	RS *rs.RS
	// XORSet describes how XRS combines subvectors with XOR.
	//
	// Key: parity index (excluding the first parity shard).
	// Value: data indexes.
	XORSet map[int][]int
}

// New creates an XRS codec with the given data and parity shard counts.
//
// parityNum cannot be 1.
func New(dataNum, parityNum int) (x *XRS, err error) {
	if parityNum == 1 {
		err = errors.New("illegal parity")
		return
	}
	r, err := rs.New(dataNum, parityNum)
	if err != nil {
		return
	}
	xs := make(map[int][]int)
	makeXORSet(dataNum, parityNum, xs)
	x = &XRS{RS: r, XORSet: xs}
	return
}

// e.g., 10+4:
//
// The resulting XOR set is: 11:[0 3 6 9] 12:[1 4 7] 13:[2 5 8],
// which means:
// b11 ⊕ a0 ⊕ a3 ⊕ a6 ⊕ a9 = new_b11
// b12 ⊕ a1 ⊕ a4 ⊕ a7 = new_b12
// b13 ⊕ a2 ⊕ a5 ⊕ a8 = new_b13
func makeXORSet(d, p int, m map[int][]int) {

	// Initialize map.
	for i := d + 1; i < d+p; i++ {
		m[i] = make([]int, 0)
	}

	// Populate map.
	j := d + 1
	for i := 0; i < d; i++ {
		if j > d+p-1 {
			j = d + 1
		}
		m[j] = append(m[j], i)
		j++
	}

	// Remove empty entries.
	for k, v := range m {
		if len(v) == 0 {
			delete(m, k)
		}
	}
}

// Encode encodes data and writes parity vectors into vects[r.DataNum:].
func (x *XRS) Encode(vects [][]byte) (err error) {

	err = checkSize(vects[0])
	if err != nil {
		return
	}
	size := len(vects[0])

	// Step 1: Reed-Solomon encode.
	err = x.RS.Encode(vects)
	if err != nil {
		return
	}

	// Step 2: XOR based on XORSet.
	half := size / 2
	for bi, xs := range x.XORSet {
		xv := make([][]byte, len(xs)+1)
		xv[0] = vects[bi][half:]
		for j, ai := range xs {
			xv[j+1] = vects[ai][:half]
		}
		xor.Encode(vects[bi][half:], xv)
	}
	return
}

func checkSize(vect []byte) error {
	size := len(vect)
	if size&1 != 0 {
		return fmt.Errorf("vect size not even: %d", size)
	}
	return nil
}

// GetNeedVects takes needReconst (which must be a data index) and returns:
// 1) a-vector indexes
// 2) b-parity-vector indexes
// required to reconstruct needReconst.
//
// It is used by ReconstOne to reduce reconstruction I/O.
//
// bNeed always has two elements, and the first is DataNum.
func (x *XRS) GetNeedVects(needReconst int) (aNeed, bNeed []int, err error) {
	d := x.RS.DataNum
	if needReconst < 0 || needReconst >= d {
		err = fmt.Errorf("illegal data index: %d", needReconst)
		return
	}

	// Find b.
	bNeed = make([]int, 2)
	bNeed[0] = d // Must have b_vects[d].
	xs := x.XORSet
	for i, s := range xs {
		if isIn(needReconst, s) {
			bNeed[1] = i
			break
		}
	}

	// Get a (excluding needReconst).
	for _, i := range xs[bNeed[1]] {
		if i != needReconst {
			aNeed = append(aNeed, i)
		}
	}
	return
}

// ReconstOne reconstructs a single data vector with reduced I/O.
// Ensure required vectors are available (see GetNeedVects).
func (x *XRS) ReconstOne(vects [][]byte, needReconst int) (err error) {

	err = checkSize(vects[0])
	if err != nil {
		return
	}

	aNeed, bNeed, err := x.GetNeedVects(needReconst)
	if err != nil {
		return
	}

	// Step 1: Reconstruct b_needReconst and rs(bNeed[1]) using Reed-Solomon.
	bVects := make([][]byte, len(vects))
	half := len(vects[0]) / 2
	for i, v := range vects {
		bVects[i] = v[half:]
	}

	d := x.RS.DataNum
	bDPHas := make([]int, d)
	for i := 0; i < d; i++ {
		bDPHas[i] = i
	}
	bDPHas[needReconst] = d // Replace needReconst with DataNum.

	bi := bNeed[1] // B index in XORSet.

	bRS := make([]byte, half)
	bVects[bi] = bRS
	err = x.RS.Reconst(bVects, bDPHas, []int{needReconst, bi})
	if err != nil {
		return
	}

	// Step 2: Reconstruct a_needReconst.
	// ∵ a_needReconst ⊕ a_need ⊕ bRS = vects[bi]
	// ∴ a_needReconst = vects[bi] ⊕ bRS ⊕ a_need
	xorV := make([][]byte, len(aNeed)+2)
	xorV[0] = vects[bi][half:]
	xorV[1] = bRS
	for i, ai := range aNeed {
		xorV[i+2] = vects[ai][:half]
	}
	xor.Encode(vects[needReconst][:half], xorV)
	return
}

// Reconst reconstructs missing vectors.
// vects: All vectors, len(vects) = dataNum + parityNum.
// dpHas: Survived data and parity index, need dataNum indexes at least.
// needReconst: Vectors indexes which need to be reconstructed.
//
// If there is exactly one missing data vector, Reconst calls ReconstOne.
// In that case, ensure the required vectors in vects are valid.
//
// Example:
// in 3+2, the whole index: [0,1,2,3,4],
// if vects[0,4] are lost and both need reconstruction,
// dpHas should be [1,2,3], and vects[1], vects[2], vects[3] must be valid.
// Reconstructed results are written back to vects[0] and vects[4] directly.
func (x *XRS) Reconst(vects [][]byte, dpHas, needReconst []int) (err error) {

	if len(needReconst) == 1 && needReconst[0] < x.RS.DataNum {
		return x.ReconstOne(vects, needReconst[0])
	}

	err = checkSize(vects[0])
	if err != nil {
		return
	}

	// Step 1: Reconstruct all a-vectors.
	half := len(vects[0]) / 2
	aVects := make([][]byte, len(vects))
	for i := range vects {
		aVects[i] = vects[i][:half]
	}
	aLost := make([]int, 0)
	for i := 0; i < x.RS.DataNum+x.RS.ParityNum; i++ {
		if !isIn(i, dpHas) {
			aLost = append(aLost, i)
		}
	}
	err = x.RS.Reconst(aVects, dpHas, aLost)
	if err != nil {
		return
	}

	// Step 2: Convert available b-vectors back to RS form when needed.
	err = x.retrieveRS(vects, dpHas)
	if err != nil {
		return
	}

	// Step 3: Reconstruct b-vectors using RS codes.
	bVects := make([][]byte, len(vects))
	for i := range vects {
		bVects[i] = vects[i][half:]
	}
	err = x.RS.Reconst(bVects, dpHas, needReconst)
	if err != nil {
		return
	}

	// Step 4: Apply XOR to b-parity-vectors according to XORSet when needed.
	d := x.RS.DataNum
	_, pn := rs.SplitNeedReconst(d, needReconst)
	if len(pn) != 0 {
		if len(pn) == 1 && pn[0] == d {
			return nil
		}
		for _, i := range pn {
			if i != d {
				xs := x.XORSet[i]
				xv := make([][]byte, len(xs)+1)
				xv[0] = vects[i][half:]
				for j, ai := range xs {
					xv[j+1] = vects[ai][:half]
				}
				xor.Encode(vects[i][half:], xv)
			}
		}
	}

	return nil
}

// retrieveRS converts available b-parity-vectors back to RS form
// by XOR-ing with the corresponding a-vectors defined in XORSet.
func (x *XRS) retrieveRS(vects [][]byte, dpHas []int) (err error) {

	half := len(vects[0]) / 2
	for _, h := range dpHas {
		if h > x.RS.DataNum { // vects[data] is rs_codes
			xs := x.XORSet[h]
			xv := make([][]byte, len(xs)+1)
			xv[0] = vects[h][half:] // put B first
			for i, ai := range xs {
				xv[i+1] = vects[ai][:half]
			}
			xor.Encode(vects[h][half:], xv)
		}
	}
	return
}

// Update updates parity data when one data vector changes.
// row is the index of the updated data vector in the full set.
func (x *XRS) Update(oldData, newData []byte, row int, parity [][]byte) (err error) {

	err = checkSize(oldData)
	if err != nil {
		return
	}

	err = x.RS.Update(oldData, newData, row, parity)
	if err != nil {
		return
	}

	_, bNeed, err := x.GetNeedVects(row)
	if err != nil {
		return
	}
	half := len(oldData) / 2
	src := make([][]byte, 3)
	bv := parity[bNeed[1]-x.RS.DataNum][half:]
	src[0], src[1], src[2] = oldData[:half], newData[:half], bv
	xor.Encode(bv, src)
	return
}

// Replace replaces oldData vectors with zero vectors, or replaces zero vectors
// with newData vectors.
//
// In practice,
// if len(replaceRows) > dataNum-parityNum, Encode is usually better.
// Replace reads len(replaceRows)+parityNum vectors; with many replacements,
// it may cost more than Encode (which only needs dataNum vectors).
//
// It's used in two situations:
//  1. The stripe was encoded before all data arrived; later, zero vectors are
//     replaced by real data vectors.
//  2. After compaction, obsolete vectors in a stripe are replaced by zero
//     vectors to free space.
//
// data indexes and replaceRows must use the same order.
func (x *XRS) Replace(data [][]byte, replaceRows []int, parity [][]byte) (err error) {

	err = checkSize(data[0])
	if err != nil {
		return
	}

	err = x.RS.Replace(data, replaceRows, parity)
	if err != nil {
		return
	}

	for i := range replaceRows {
		_, bNeed, err2 := x.GetNeedVects(replaceRows[i])
		if err2 != nil {
			return err2
		}

		half := len(data[0]) / 2
		bv := parity[bNeed[1]-x.RS.DataNum][half:]
		xor.Encode(bv, [][]byte{bv, data[i][:half]})
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
