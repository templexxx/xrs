package xrs

import (
	"errors"
	"fmt"
	rs "github.com/templexxx/reedsolomon"
	xor "github.com/templexxx/xorsimd"
)

type XRS struct {
	RS     *rs.RS
	XORSet map[int][]int // xor_set map
}

// New create an XRS
//
// parityCnt can't be 1
func New(dataCnt, parityCnt int) (x *XRS, err error) {
	if parityCnt == 1 {
		err = errors.New("illegal parity")
		return
	}
	r, err := rs.New(dataCnt, parityCnt)
	if err != nil {
		return
	}
	xs := make(map[int][]int)
	makeXORSet(dataCnt, parityCnt, xs)
	x = &XRS{RS: r, XORSet: xs}
	return
}

/*
	<vects_index>:<data_a_index_list>
	e.g. 10+4
	11:[0 3 6 9] 12:[1 4 7] 13:[2 5 8]
 	b-11 ⊕ a-0 ⊕ a-3 ⊕ a-6 ⊕ a-9 = new_b-11
 	b-12 ⊕ a-1 ⊕ a-4 ⊕ a-7 = new_b-12
	b-13 ⊕ a-2 ⊕ a-5 ⊕ a-8 = new_b-13
*/
func makeXORSet(d, p int, m map[int][]int) {
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

// Encode outputs parity into vects
func (x *XRS) Encode(vects [][]byte) (err error) {
	size := len(vects[0])
	if !((size & 1) == 0) {
		err = errors.New(fmt.Sprintf("vect size not even: %d", size))
		return
	}
	err = x.RS.Encode(vects)
	if err != nil {
		return
	}

	half := size / 2
	for i, a := range x.XORSet {
		v := make([][]byte, len(a)+1)
		v[0] = vects[i][half:]
		for i, k := range a {
			v[i+1] = vects[k][:half]
		}
		xor.Encode(vects[i][half:], v)
	}
	return
}

// Update parity when one data_vect changes
func (x *XRS) Update(oldData, newData []byte, updateRow int, parity [][]byte) (err error) {
	err = x.RS.Update(oldData, newData, updateRow, parity)
	if err != nil {
		return
	}

	_, bNeed, err := x.GetNeedVects(updateRow)
	if err != nil {
		return
	}
	half := len(oldData) / 2
	xor.Update(oldData[:half], newData[:half], parity[bNeed[1]-x.RS.DataCnt][half:])
	return
}

// Reconst repair missing vects, len(dpHas) must == dataCnt
// It's not the most efficient way to reconst only one lost, pls use reconstOne if only lost one data_vect
// e.g:
// in 3+2, the whole index: [0,1,2,3,4]
// if vects[0,4] are lost
// the "dpHas" will be [1,2,3] ,and you must be sure that vects[1] vects[2] vects[3] have correct data
// results will be put into vects[0]&vects[4]
//
// if you don't want reconst parity, needReconst will be [0], and it will repair vects[0] only
func (x *XRS) Reconst(vects [][]byte, dpHas, needReconst []int) (err error) {
	// vects: [A|B]
	// step1: reconst all lost A
	half := len(vects[dpHas[0]]) / 2
	vectsA := make([][]byte, len(vects))
	for i := range vects {
		vectsA[i] = vects[i][:half]
	}
	aLost := make([]int, 0)
	for i := 0; i < x.RS.DataCnt+x.RS.ParityCnt; i++ {
		if !isIn(i, dpHas) {
			aLost = append(aLost, i)
		}
	}
	err = x.RS.Reconst(vectsA, dpHas, aLost)
	if err != nil {
		return
	}
	// step2: B -> rs_codes
	err = x.retrieveRS(half, dpHas, vects)
	if err != nil {
		return
	}
	// step3: reconst B
	vectsB := make([][]byte, len(vects))
	for i := range vects {
		vectsB[i] = vects[i][half:]
	}
	err = x.RS.Reconst(vectsB, dpHas, needReconst)
	if err != nil {
		return
	}

	// step4: xor B (if need reconst)
	d := x.RS.DataCnt
	_, pn := rs.SplitNeedReconst(d, needReconst)
	if len(pn) != 0 {
		if len(pn) == 1 && pn[0] == d {
			return nil
		}
		for _, p := range pn {
			if p != d {
				xs := x.XORSet[p]
				xv := make([][]byte, len(xs)+1)
				xv[0] = vects[p][half:]
				for k, e := range xs {
					xv[k+1] = vects[e][:half]
				}
				xor.Encode(vects[p][half:], xv)
			}
		}
	}
	return nil
}

// ReconstOne reconst one data vect, it saves I/O
// Make sure you have some specific vects ( you can get the vects index from GetNeedVects)
func (x *XRS) ReconstOne(vects [][]byte, needReconst int) (err error) {
	aNeed, bNeed, err := x.GetNeedVects(needReconst) // help with checking needReconst index
	if err != nil {
		return
	}
	// step1: reconst b_data & bNeed[1] -> original rs_codes
	d := x.RS.DataCnt
	bVects := make([][]byte, len(vects))
	half := len(vects[0]) / 2
	for i, v := range vects {
		bVects[i] = v[half:]
	}
	bHas := make([]int, d)
	for i := 0; i < d; i++ {
		bHas[i] = i
	}
	bHas[needReconst] = d
	bRSV := make([]byte, half)
	bVects[bNeed[1]] = bRSV
	err = x.RS.Reconst(bVects, bHas, []int{needReconst, bNeed[1]})
	if err != nil {
		return
	}
	// step2: reconst a_lost
	aXorV := make([][]byte, len(aNeed)+2)
	aXorV[0] = vects[bNeed[1]][half:]
	aXorV[1] = bRSV
	for i, a0 := range aNeed {
		aXorV[i+2] = vects[a0][:half]
	}
	xor.Encode(vects[needReconst][:half], aXorV)
	return
}

// GetNeedVects receive needReconst index return a&b vects' index for reconstructing
func (x *XRS) GetNeedVects(needReconst int) (aNeed, bNeed []int, err error) {
	d, xs := x.RS.DataCnt, x.XORSet
	if needReconst < 0 || needReconst >= d {
		err = errors.New(fmt.Sprintf("illegal index: %d", needReconst))
		return
	}
	bNeed = make([]int, 2)
	bNeed[0] = d // must has b_vects[d]
	for k, s := range xs {
		if isIn(needReconst, s) {
			bNeed[1] = k
			break
		}
	}
	for _, v := range xs[bNeed[1]] {
		if v != needReconst {
			aNeed = append(aNeed, v)
		}
	}
	return
}

// xor a_vects & b_vect -> rs_codes
func (x *XRS) retrieveRS(half int, dpHas []int, vects [][]byte) (err error) {
	for _, h := range dpHas {
		if h > x.RS.DataCnt { // vects[data] is rs_codes
			a := x.XORSet[h]
			xv := make([][]byte, len(a)+1)
			xv[0] = vects[h][half:] // put B first
			for i, a0 := range a {
				xv[i+1] = vects[a0][:half]
			}
			xor.Encode(vects[h][half:], xv)
		}
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
