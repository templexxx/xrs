package xrs

import "errors"

func (pig *piggy) Encode(dp matrix) error {
	if len(dp) != pig.shards {
		return ErrTooFewShards
	}
	size, err := checkShardSize(dp)
	if err != nil {
		return err
	}
	inMap := make(map[int]int)
	outMap := make(map[int]int)
	for i := 0; i < pig.data; i++ {
		inMap[i] = i
	}
	for i := pig.data; i < pig.shards; i++ {
		outMap[i-pig.data] = i
	}
	rsRunner(pig.gen, dp, pig.data, pig.parity, 0, size, inMap, outMap)
	pigEncode(dp, pig.xMap, size, inMap, outMap)
	return nil
}

func pigEncode(dp matrix, xmap xorMap, size int, inMap, outMap map[int]int) {
	for p, ds := range xmap {
		pigRunner(dp, ds, p, size, inMap, outMap)
	}
}

func pigRunner(dp matrix, ds []int, p, size int, inMap, outMap map[int]int) {
	startIn := 0
	unit := 16 * 1024
	do := unit
	half := size / 2
	startOut := half
	for startIn < half {
		if startIn+do <= size {
			pigWorker(dp, ds, p, startIn, startOut,  do, inMap, outMap)
			startIn = startIn + do
			startOut = startOut + do
		} else {
			pigRemain(dp, ds, p, startIn, startOut, size, inMap, outMap)
			startIn = size
		}
	}
}

func pigWorker(dp matrix, ds []int, p, startIn, startOut, do int, inMap, outMap map[int]int) {
	endIn := startIn + do
	endOut := startOut + do
	k := outMap[p]
	for _, i := range ds {
		j := inMap[i]
		in := dp[j]
		xorAVX2(in[startIn:endIn], dp[k][startOut:endOut])
	}
}

func pigRemain(dp matrix, ds []int, p, startIn, startOut, size int, inMap, outMap map[int]int) {
	do := size - startOut
	half := size / 2
	k := outMap[p]
	for _, i := range ds {
		j := inMap[i]
		in := dp[j]
		xorRemainAVX2(in[startIn:half], dp[k][startOut:size], do)
	}
}

func rsRunner(gen, dp matrix, numIn, numOut, start, size int, inMap, outMap map[int]int) {
	unitSize := 16 * 1024 // concurrency unit size（Haswell， Skylake， Kabylake's L1 data cache size is 32KB)
	do := unitSize
	for start < size {
		if start+do <= size {
			rsWorker(gen, dp, start, do, numIn, numOut, inMap, outMap)
			start = start + do
		} else {
			rsRemain(start, size, gen, dp, numIn, numOut, inMap, outMap)
			start = size
		}
	}
}

func rsWorker(gen, dp matrix, start, do, numIn, numOut int, inMap, outMap map[int]int) {
	end := start + do
	for i := 0; i < numIn; i++ {
		j := inMap[i]
		in := dp[j]
		for oi := 0; oi < numOut; oi++ {
			k := outMap[oi]
			c := gen[oi][i]
			if i == 0 { // it means don't need to copy parity data for xor
				gfMulAVX2(mulTableLow[c][:], mulTableHigh[c][:], in[start:end], dp[k][start:end])
			} else {
				gfMulXorAVX2(mulTableLow[c][:], mulTableHigh[c][:], in[start:end], dp[k][start:end])
			}
		}
	}
}

func rsRemain(start, size int, gen, dp matrix, numIn, numOut int, inMap, outMap map[int]int) {
	do := size - start
	for i := 0; i < numIn; i++ {
		j := inMap[i]
		in := dp[j]
		for oi := 0; oi < numOut; oi++ {
			k := outMap[oi]
			c := gen[oi][i]
			if i == 0 {
				gfMulRemain(c, in[start:size], dp[k][start:size], do)
			} else {
				gfMulRemainXor(c, in[start:size], dp[k][start:size], do)
			}
		}
	}
}

var ErrShardSize = errors.New("piggy: shards size equal 0 or not match")
var ErrDataShardsOdd = errors.New("piggy: shards size num must be even")

func checkShardSize(m matrix) (int, error) {
	size := len(m[0])
	if size == 0 {
		return size, ErrShardSize
	}
	if !even(size) {
		return size, ErrDataShardsOdd
	}
	for _, v := range m {
		if len(v) != size {
			return 0, ErrShardSize
		}
	}
	return size, nil
}

func even(num int) bool {
	return (num & 1) == 0
}
