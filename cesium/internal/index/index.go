// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package index provides functionality for efficient queries of time series on top of
// a data source.
//
// The Index interface provides utilities for finding the number of samples between two
// timestamps, and for finding the timestamp between a starting timestamp and a number of
// samples.
//
// The Domain index is an implementation designed for lookups on domain databases.
package index

import (
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// ErrDiscontinuous is returned if the index has discontinuities across an evaluated
// time range.
var ErrDiscontinuous = errors.New("discontinuous")

func NewErrDiscontinuousTR(tr telem.TimeRange) error {
	return errors.Wrapf(ErrDiscontinuous, "the time range %s is not continuous in the index", tr)
}

func NewErrDiscontinuousOffset(offset int64, domainLen int64) error {
	return errors.Wrapf(ErrDiscontinuous, "failed to resolve position %d in continuous index of length %d", offset, domainLen)
}

func NewErrDiscontinuousStamp(stamp telem.TimeStamp) error {
	return errors.Wrapf(ErrDiscontinuous, "the timestamp %s does not exist in the index", stamp)
}

// ContinuousPolicy is a type alias for a boolean that indicates whether a domain
// Distance or Stamp lookup must require a continuous set of samples. Used to improve
// readability.
type ContinuousPolicy = bool

const (
	// AllowDiscontinuous allows a lookup to span across multiple domains that
	// do not contain continuous samples i.e. at some point in the future a caller
	// can insert samples between the two domains without needing to delete first.
	AllowDiscontinuous ContinuousPolicy = false
	// MustBeContinuous requires that a lookup span only continuous domains i.e. there
	// is a guarantee that new samples cannot be inserted between the domains.
	MustBeContinuous ContinuousPolicy = true
)
