// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package index

import (
	"github.com/synnaxlabs/x/bounds"
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

// Approximation represents the approximate location of a continuous value within
// a discrete set of values, i.e., the 'index' of the value 1.5 within the indexes
// [0,1,2,3] would lie between 1 and 2. The approximation is represented by an
// upper and lower bound of candidate indices.
type Approximation[T types.Numeric] struct{ bounds.Bounds[T] }

// Exact returns true if the approximation represents an exact location i.e., a.Lower
// and a.Upper are equal.
func (a Approximation[T]) Exact() bool { return a.Lower == a.Upper }

// Exactly returns an approximation for an exact position. i.e., the approximation
// for value 2 within [1,2,3,4,5] would be Exactly(1).
func Exactly[T types.Numeric](v T) Approximation[T] { return Between(v, v) }

// Between returns an approximation for a position that is between two locations
// in the discrete set i.e., the approximation for value 2 within [1,3,5,7] would
// be Between(0, 1).
func Between[T types.Numeric](lower, upper T) Approximation[T] {
	return Approximation[T]{bounds.Bounds[T]{Lower: lower, Upper: upper}}
}
