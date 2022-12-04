package index

import (
	"fmt"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/types"
)

type (
	DistanceApproximation  = Approximation[int64]
	TimeStampApproximation = Approximation[telem.TimeStamp]
)

type Approximation[T types.Numeric] struct {
	Lower T
	Upper T
}

func (a Approximation[T]) Exact() bool { return a.Lower-a.Upper == 0 }

func (a Approximation[T]) String() string { return fmt.Sprintf("[%v, %v]", a.Lower, a.Upper) }

func Exactly[T types.Numeric](v T) Approximation[T] { return Between(v, v) }

func Between[T types.Numeric](lower, upper T) Approximation[T] {
	return Approximation[T]{Lower: lower, Upper: upper}
}
