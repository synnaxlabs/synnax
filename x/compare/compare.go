package compare

import "github.com/synnaxlabs/x/types"

// Result is the outcome of a comparison between two values.
type Result uint8

const (
	// Less is a value that is less than another value.
	Less Result = iota
	// Equal is a value that is equal to another value.
	Equal
	// Greater is a value that is greater than another value.
	Greater
)

// TupleFunc is a function that compares two values and returns a Result.
type TupleFunc[T any] func(T, T) Result

// UnaryFunc returns a TupleFunc that compares a value and returns a Result.
type UnaryFunc[T any] func(T) Result

// NumericTuple returns a TupleFunc that compares two numeric values and returns
// a Result.
func NumericTuple[T types.Numeric]() TupleFunc[T] {
	return func(a, b T) Result {
		switch {
		case a < b:
			return Less
		case a > b:
			return Greater
		default:
			return Equal
		}
	}
}

// NumericUnary returns a UnaryFunc that compares a numeric value against a given target
// and returns a Result.
func NumericUnary[T types.Numeric](target T) UnaryFunc[T] {
	t := NumericTuple[T]()
	return func(a T) Result { return t(a, target) }
}
