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
	"context"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// ErrDiscontinuous is returned if the index has discontinuities across an evaluated
// time range.
var ErrDiscontinuous = errors.New("discontinuous")

func NewErrDiscontinuousTR(tr telem.TimeRange) error {
	return errors.Wrapf(ErrDiscontinuous, "the time range %s does not exist in the index", tr)
}

func NewErrDiscontinuousStamp(offset int64, domainLen int64) error {
	return errors.Wrapf(ErrDiscontinuous, "failed to resolve position %d in continuous index of length %d", offset, domainLen)
}

// Index implements an index over a time series.
type Index interface {
	// Distance calculates an approximate distance (arithmetic difference in offset)
	// between the start and end timestamps of the given time range. If continuous is
	// true, the index will return an error if the underlying telemetry has
	// discontinuities across the time range.
	//
	// The distance is approximated using a lower and upper bound. The underlying time
	// series can be viewed as a contiguous slice of timestamps, where each timestamp
	// exists at a specific index (i.e. slice[x]). The lower bound of the distance is
	// the index of the timestamp less than or equal to the end timestamp and
	// the index of the timestamp greater than or equal to the start timestamp. The upper
	// bound is calculated using the opposite approach (i.e. finding the index of the
	// timestamp greater than or equal to the end timestamp and the index of the
	// timestamp less than or equal to the start timestamp). Naturally, a time range
	// whose start timestamp and end timestamps are both known will have an equal lower
	// and upper bound.
	Distance(ctx context.Context, tr telem.TimeRange, continuous bool) (DistanceApproximation, error)
	// Stamp calculates an approximate ending timestamp for a range given a known distance
	// in the number of samples. This operation may be understood as the
	// opposite of Distance.
	// Stamp assumes the caller is aware of discontinuities in the underlying time
	// series, and will calculate the ending timestamp even across discontinuous ranges.
	Stamp(ctx context.Context, ref telem.TimeStamp, distance int64, continuous bool) (TimeStampApproximation, error)
	// Info returns the key and name of the channel of the index. If the database is
	// domain-indexed, the information of the domain channel is returned. If the database
	// is rate-based (i.e. self-indexing), the channel itself is returned.
	Info() string
}
