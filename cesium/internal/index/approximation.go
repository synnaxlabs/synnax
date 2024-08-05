// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package index

import (
	"fmt"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/types"
)

type TimeStampApproximation = Approximation[telem.TimeStamp]

type DistanceApproximation struct {
	Approximation[int64]
	// StartExact denotes whether the start timestamp of the requested time range is in
	// the index.
	StartExact bool
	// EndExact denotes whether the end timestamp of the requested time range is in the
	// index.
	EndExact bool
}

type Approximation[T types.Numeric] struct {
	Lower T
	Upper T
}

func (a Approximation[T]) Exact() bool {
	return a.Lower-a.Upper == 0
}

func (a Approximation[T]) String() string {
	return fmt.Sprintf("[%v, %v]", a.Lower, a.Upper)
}

func Exactly[T types.Numeric](v T) Approximation[T] {
	return Between(v, v)
}

func Between[T types.Numeric](lower, upper T) Approximation[T] {
	return Approximation[T]{Lower: lower, Upper: upper}
}
