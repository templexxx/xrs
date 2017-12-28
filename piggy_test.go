package xrs

import (
	"testing"
)

func TestGetXorMap(t *testing.T) {
	xmap := genXorMap(14, 10)
	m := make(map[int][]int)
	m[1] = []int{0}
	m[2] = []int{1}
	m[3] = []int{2}
	m[4] = []int{3}
	m[5] = []int{4,5}
	m[6] = []int{6,7}
	m[7] = []int{8,9}
	m[8] = []int{10,11}
	m[9] = []int{12,13}
	for k, v := range m {
		if !eqSlice(xmap[k], v) {
			t.Fatalf("make xor map error, shards %d is not equal", k)
		}
	}
}

func eqSlice(a, b []int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
