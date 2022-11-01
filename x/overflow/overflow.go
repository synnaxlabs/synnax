package overflow

import "math"

// CapInt64 caps the addition of two int64 values
// to the max/min of int64.
func CapInt64(a, b int64) int64 {
	if b > 0 && a > math.MaxInt64-b {
		return math.MaxInt64
	}
	if b < 0 && a < math.MinInt64-b {
		return math.MinInt64
	}
	return a + b
}
