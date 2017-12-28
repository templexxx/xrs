package xrs

//go:noescape
func copyAVX2(in, out []byte)

//go:noescape
func xorAVX2(in, out []byte)

func xorRunner(dp matrix, output []byte, size, numIn int, inMap map[int]int) {
	start := 0
	unitSize := 16 * 1024 // concurrency unit size（Haswell， Skylake， Kabylake's L1 data cache size is 32KB)
	do := unitSize
	for start < size {
		if start+do <= size {
			xorWorker(start, do, numIn, dp, output, inMap)
			start = start + do
		} else {
			xorRemain(start, size, numIn, dp, output, inMap)
			start = size
		}
	}
}

func xorWorker(start, do, numIn int, dp matrix, output []byte, inMap map[int]int) {
	end := start + do
	for i := 0; i < numIn; i++ {
		j := inMap[i]
		in := dp[j]
		if i == 0 {
			// actually it maybe slower than "copy"
			// but it's more cache-friendly than golang's copy
			copyAVX2(in[start:end], output[start:end])
		} else {
			xorAVX2(in[start:end], output[start:end])
		}
	}
}

func xorRemain(start, size, numIn int, dp matrix, output []byte, inMap map[int]int) {
	do := size - start
	for i := 0; i < numIn; i++ {
		j := inMap[i]
		in := dp[j]
		if i == 0 {
			copyRemain(in[start:size], output[start:size], do)
		} else {
			xorRemainAVX2(in[start:size], output[start:size], do)
		}
	}
}

func copyRemain(input, output []byte, size int) {
	var done int
	if size < 32 {
		for i, _ := range input {
			output[i] = input[i]
		}
	} else {
		copyAVX2(input, output)
		done = (size >> 5) << 5
		remain := size - done
		if remain > 0 {
			for i := done; i < size; i++ {
				output[i] = input[i]
			}
		}
	}
}

func xorRemainAVX2(input, output []byte, size int) {
	var done int
	if size < 32 {
		for i, v := range output {
			v = v ^ input[i]
			output[i] = v
		}
	} else {
		xorAVX2(input, output)
		done = (size >> 5) << 5
		remain := size - done
		if remain > 0 {
			for i := done; i < size; i++ {
				v0 := output[i]
				v1 := input[i]
				v := v0 ^ v1
				output[i] = v
			}
		}
	}
}
