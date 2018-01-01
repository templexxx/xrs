package xrs

// non-temporal hint store
const nontmp = 8 * 1024

func xorAVX2(dst []byte, src [][]byte) {
	size := len(dst)
	if size > nontmp {
		xorAVX2big(dst, src)
	} else {
		xorAVX2small(dst, src)
	}
}

func xorSSE2(dst []byte, src [][]byte) {
	size := len(dst)
	if size > nontmp {
		xorSSE2big(dst, src)
	} else {
		xorSSE2small(dst, src)
	}
}

//go:noescape
func xorAVX2small(dst []byte, src [][]byte)

//go:noescape
func xorAVX2big(dst []byte, src [][]byte)

//go:noescape
func xorSSE2small(dst []byte, src [][]byte)

//go:noescape
func xorSSE2big(dst []byte, src [][]byte)
