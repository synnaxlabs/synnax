package compare

import "github.com/synnaxlabs/x/types"

type Result uint8

const (
	Less Result = iota
	Equal
	Greater
)

// TupleFunc is a function that compares two values and returns a Result.
type TupleFunc[T any] func(T, T) Result

// UnaryFunc returns a TupleFunc that compares a value and returns a Result.
type UnaryFunc[T any] func(T) Result

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

func NumericUnary[T types.Numeric](target T) UnaryFunc[T] {
	t := NumericTuple[T]()
	return func(a T) Result { return t(a, target) }
}
