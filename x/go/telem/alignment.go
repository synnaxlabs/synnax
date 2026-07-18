// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem

import latest "github.com/synnaxlabs/x/telem/types/v0"

// AlignmentBounds is a set of lower and upper bounds for the alignment of a
// multi-sample data structure (such as a Series or MultiSeries). The lower bound
// represents the alignment of the first sample, while the upper bound represents the
// alignment of the last sample + 1. The lower bound is inclusive, while the upper bound
// is exclusive.
type AlignmentBounds = latest.AlignmentBounds

// MaxAlignment is the maximum possible value for an alignment.
const MaxAlignment = latest.MaxAlignment

// AlignmentBoundsZero is a set of alignment bounds whose lower and upper bound are
// both zero.
var AlignmentBoundsZero = latest.AlignmentBoundsZero

// NewAlignment takes the given array index and sample index within that array and
// returns a new Alignment (see Alignment for more information).
func NewAlignment(domainIdx, sampleIdx uint32) Alignment {
	return latest.NewAlignment(domainIdx, sampleIdx)
}
