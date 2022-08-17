package iter

// Endlessly returns a function that iterates over a collection of values
// indefinitely.
func Endlessly[T any](values []T) func() T {
	i := 0
	return func() T {
		val := values[i]
		if i < (len(values) - 1) {
			i++
		} else {
			i = 0
		}
		return val
	}
}
