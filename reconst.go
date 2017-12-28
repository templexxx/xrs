package xrs

import (
	"sort"
)

//import "sort"

// dp : data+parity shards, all shards size must be equal
// lost : row number in dp
// TODO 检查lost的合法性
func (pig *piggy) Reconst(dp matrix, lost []int, repairParity bool) error {
	if len(dp) != pig.shards {
		return ErrTooFewShards
	}
	size, err := checkShardSize(dp)
	if err != nil {
		return err
	}
	if len(lost) == 0 {
		return nil
	}
	if len(lost) > pig.parity {
		return ErrTooFewShards
	}
	dataLost, parityLost := splitLost(lost, pig.data)
	survived := make(map[int]int)
	var reErr error
	if len(dataLost) > 0 {
		if len(dataLost) == 1 && len(parityLost) == 0 {
			need := needParity(dataLost[0], pig.data, pig.xMap)
			reconstDataOne(pig.m, dp, dataLost[0], need, pig.data, size)
		} else {
			reErr, survived = reconstData(pig.m, dp, dataLost, parityLost, pig.data, size, pig.xMap)
			if reErr != nil {
				return reErr
			}
		}
	}
	if len(parityLost) > 0 && repairParity {
		reconstParity(pig.m, dp, parityLost, pig.data, size, pig.xMap, survived)
	}
	return nil
}

func reconstParity(encodeMatrix, dp matrix, parityLost []int, numData, size int, xmap xorMap, survived map[int]int) {
	gen := NewMatrix(len(parityLost), numData)
	outputMap := make(map[int]int)
	for i := range gen {
		l := parityLost[i]
		gen[i] = encodeMatrix[l]
		outputMap[i] = l
	}
	inMap := make(map[int]int)
	for i := 0; i < numData; i++ {
		inMap[i] = i
	}
	rsRunner(gen, dp, numData, len(parityLost), 0, size, inMap, outputMap)
	pLMap := make(map[int]int)
	for i, v := range parityLost {
		pLMap[i] = v
	}
	l := len(pLMap)
	for _, v := range survived {
		if !vInMap(v, pLMap) && v >= numData {
			pLMap[l] = v
			l++
		}
	}
	backRS(dp, xmap, numData, size, pLMap)
}

func vInMap(v int, m map[int]int) bool {
	for _, vm := range m {
		if v == vm {
			return true
		}
	}
	return false
}

func reconstData(encodeMatrix, dp matrix, dataLost, parityLost []int, numData, size int, xmap xorMap) (error, map[int]int) {
	// reconst A first
	gen, survived, output, err := genGenMatrix(encodeMatrix, numData, 0, dataLost, parityLost)
	if err != nil {
		return err, survived
	}
	rsRunner(gen, dp, numData, len(dataLost), 0, size/2, survived, output)
	// reconst B
	backRS(dp, xmap, numData, size, survived)
	rsRunner(gen, dp, numData, len(dataLost), size/2, size, survived, output)
	return nil, survived
}

// B rs+xor + xor -> B rs
func backRS(dp matrix, xmap xorMap, numData, size int, survived map[int]int) {
	inMap := make(map[int]int)
	outMap := make(map[int]int)
	for i := 0; i < numData; i++ {
		inMap[i] = i
	}
	for i := numData; i < len(dp); i++ {
		outMap[i-numData] = i
	}
	for i := 0; i < numData; i++ {
		rows := survived[i]
		if rows > numData {
			as := xmap[rows-numData]
			pigRunner(dp, as, rows-numData, size, inMap, outMap)
		}
	}
}

func reconstDataOne(encodeMatrix, dp matrix, dataLost, needParity, numData, size int) error {
	gen, survived, output, err := genGenMatrix(encodeMatrix, numData, needParity, []int{dataLost}, []int{})
	if err != nil {
		return err
	}
	rsRunner(gen, dp, numData, 1, size/2, size, survived, output)
	return nil
}

// ret genMatrix, survivedMap, outputMap
func genGenMatrix(encodeMatrix matrix, numData, need int, dataLost, parityLost []int) (matrix, map[int]int, map[int]int, error) {
	decodeMatrix := NewMatrix(numData, numData)
	survivedMap := make(map[int]int)
	numShards := len(encodeMatrix)
	// fill with survived data
	for i := 0; i < numData; i++ {
		if !inSlice(dataLost, i) {
			decodeMatrix[i] = encodeMatrix[i]
			survivedMap[i] = i
		}
	}
	// "borrow" from survived parity
	k := numData
	if need != 0 {
		decodeMatrix[dataLost[0]] = encodeMatrix[k]
		survivedMap[dataLost[0]] = k
	} else {
		for _, dl := range dataLost {
			for j := k; j < numShards; j++ {
				k++
				if !inSlice(parityLost, j) {
					decodeMatrix[dl] = encodeMatrix[j]
					survivedMap[dl] = j
					break
				}
			}
		}
	}
	var err error
	numDL := len(dataLost)
	gen := NewMatrix(numDL, numData)
	outputMap := make(map[int]int)
	decodeMatrix, err = decodeMatrix.invert()
	if err != nil {
		return gen, survivedMap, outputMap, err
	}

	// fill generator matrix with lost rows of decode matrix
	for i, l := range dataLost {
		gen[i] = decodeMatrix[l]
		outputMap[i] = l
	}
	return gen, survivedMap, outputMap, nil
}

//

//

//
func splitLost(lost []int, d int) ([]int, []int) {
	var dataLost []int
	var parityLost []int
	for _, l := range lost {
		if l < d {
			dataLost = append(dataLost, l)
		} else {
			parityLost = append(parityLost, l)
		}
	}
	sort.Ints(dataLost)
	sort.Ints(parityLost)
	return dataLost, parityLost
}

func needParity(dataLost, data int, xmap xorMap) int {
	var need int
	for k, v := range xmap {
		if inSlice(v, dataLost) {
			need = k + data
		}
	}
	return need
}

func inSlice(s []int, i int) bool {
	for _, v := range s {
		if i == v {
			return true
		}
		continue
	}
	return false
}
