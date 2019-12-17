// Copyright (c) 2017 Temple3x (temple3x@gmail.com)
//
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

// Package xrs implements Erasure Codes based on
// <A “Hitchhiker’s” Guide to Fast and Efﬁcient Data Reconstruction in Erasure-coded Data Centers>,
// split row vectors into two equal size parts:
// e.g. 10+4:
// +---------+
// | a1 | b1 |
// +---------+
// | a2 | b2 |
// +---------+
// | a3 | b3 |
// +---------+
//     ...
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

// XRS X-Reed-Solomon Codes receiver.
type XRS struct {
	// RS is the backend of XRS>
	RS *rs.RS
	// XORSet shows how XRS combines sub-vectors by xor.
	//
	// Key: Parity index(except first parity).
	// Value: Data indexes.
	XORSet map[int][]int
}

// New create an XRS with specific data & parity numbers.
//
// Warn:
// parityNum can't be 1.
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

// e.g. 10+4:
//
// We will have this xor_set: 11:[0 3 6 9] 12:[1 4 7] 13:[2 5 8],
// which means:
// b11 ⊕ a0 ⊕ a3 ⊕ a6 ⊕ a9 = new_b11
// b12 ⊕ a1 ⊕ a4 ⊕ a7 = new_b12
// b13 ⊕ a2 ⊕ a5 ⊕ a8 = new_b13
func makeXORSet(d, p int, m map[int][]int) {

	// Init map.
	for i := d + 1; i < d+p; i++ {
		m[i] = make([]int, 0)
	}

	// Fill map.
	j := d + 1
	for i := 0; i < d; i++ {
		if j > d+p-1 {
			j = d + 1
		}
		m[j] = append(m[j], i)
		j++
	}

	// Clean map.
	for k, v := range m {
		if len(v) == 0 {
			delete(m, k)
		}
	}
}

// Encode encodes data for generating parity.
// Write parity vectors into vects[r.DataNum:].
func (x *XRS) Encode(vects [][]byte) (err error) {

	err = checkSize(vects[0])
	if err != nil {
		return
	}
	size := len(vects[0])

	// Step1: Reed-Solomon encode.
	err = x.RS.Encode(vects)
	if err != nil {
		return
	}

	// Step2: XOR by xor_set.
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

// GetNeedVects receives needReconst index(it must be data vector)
// returns a_vectors' indexes and b_parity_vectors' indexes for reconstructing needReconst.
// It's used for ReconstOne to read correct vectors for saving I/O.
//
// bNeed always has two elements, the first one is DataNum.
func (x *XRS) GetNeedVects(needReconst int) (aNeed, bNeed []int, err error) {
	d := x.RS.DataNum
	if needReconst < 0 || needReconst >= d {
		err = fmt.Errorf("illegal data index: %d", needReconst)
		return
	}

	// Find b.
	bNeed = make([]int, 2)
	bNeed[0] = d // Must has b_vects[d].
	xs := x.XORSet
	for i, s := range xs {
		if isIn(needReconst, s) {
			bNeed[1] = i
			break
		}
	}

	// Get a (except needReconst).
	for _, i := range xs[bNeed[1]] {
		if i != needReconst {
			aNeed = append(aNeed, i)
		}
	}
	return
}

// ReconstOne reconstruct one data vector, it saves I/O.
// Make sure you have some specific vectors data. ( you can get the vectors' indexes from GetNeedVects)
func (x *XRS) ReconstOne(vects [][]byte, needReconst int) (err error) {

	err = checkSize(vects[0])
	if err != nil {
		return
	}

	aNeed, bNeed, err := x.GetNeedVects(needReconst)
	if err != nil {
		return
	}

	// Step1: Reconstruct b_needReconst & rs(bNeed[1]), using original Reed-Solomon Codes.
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

	// Step2: Reconstruct a_needReconst
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

// Reconst reconstructs missing vectors,
// vects: All vectors, len(vects) = dataNum + parityNum.
// dpHas: Survived data & parity index, need dataNum indexes at least.
// needReconst: Vectors indexes which need to be reconstructed.
//
// Warn:
// If there is only one needReconst, it will call ReconstOne,
// so make sure you have correct data, if there is only one vectors need to repair.
//
// e.g:
// in 3+2, the whole index: [0,1,2,3,4],
// if vects[0,4] are lost & they need to be reconstructed
// (Maybe you only need vects[0], so the needReconst should be [0], but not [0,4]).
// the "dpHas" will be [1,2,3] ,and you must be sure that vects[1] vects[2] vects[3] have correct data,
// results will be written into vects[0]&vects[4] directly.
func (x *XRS) Reconst(vects [][]byte, dpHas, needReconst []int) (err error) {

	if len(needReconst) == 1 && needReconst[0] < x.RS.DataNum {
		return x.ReconstOne(vects, needReconst[0])
	}

	err = checkSize(vects[0])
	if err != nil {
		return
	}

	// Step1: Reconstruct all a_vectors.
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

	// Step2: Retrieve b_vectors to RS codes(if has).
	err = x.retrieveRS(vects, dpHas)
	if err != nil {
		return
	}

	// Step3: Reconstruct b_vectors using RS codes.
	bVects := make([][]byte, len(vects))
	for i := range vects {
		bVects[i] = vects[i][half:]
	}
	err = x.RS.Reconst(bVects, dpHas, needReconst)
	if err != nil {
		return
	}

	// Step4: XOR b_parity_vectors according to XORSet(if need).
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

// retrieveRS retrieves b_parity_vects(if has) to RS codes
// by XOR itself and a_vects in XORSet.
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

// Update updates parity_data when one data_vect changes.
// row: It's the new data's index in the whole vectors.
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

// Replace replaces oldData vectors with 0 or replaces 0 with newData vectors.
//
// In practice,
// If len(replaceRows) > dataNum-parityNum, it's better to use Encode,
// because Replace need to read len(replaceRows) + parityNum vectors,
// if replaceRows are too many, the cost maybe larger than Encode
// (Encode only need read dataNum).
// Think about an EC compute node, and dataNum+parityNum data nodes model.
//
// It's used in two situations:
// 1. We didn't have enough data for filling in a stripe, but still did ec encode,
// we need replace several zero vectors with new vectors which have data after we get enough data finally.
// 2. After compact, we may have several useless vectors in a stripe,
// we need replaces these useless vectors with zero vectors for free space.
//
// Warn:
// data's index & replaceRows must has the same sort.
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
