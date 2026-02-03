// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package domain

import (
	"github.com/synnaxlabs/cesium/internal/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

var (
	// ErrWriteConflict is returned when a domain overlaps with an existing domain in
	// the DB.
	ErrWriteConflict = errors.Wrap(validate.ErrValidation, "write overlaps with existing data in database")
	// ErrRangeNotFound is returned when a requested domain is not found in the DB.
	ErrRangeNotFound = errors.Wrap(query.ErrNotFound, "time range not found")
	// ErrDBClosed is returned when an operation is attempted on a closed DB.
	ErrDBClosed = resource.NewClosedError("domain.db")
)

// NewRangeWriteConflictError creates a new error returned when existing data in the
// database overlaps with a callers attempt to write new data.
func NewRangeWriteConflictError(newTR, existingTR telem.TimeRange) error {
	if newTR.Span().IsZero() {
		return NewPointWriteConflictError(newTR.Start, existingTR)
	}
	intersection := newTR.Intersection(existingTR)
	return errors.Wrapf(
		ErrWriteConflict,
		"write for range %s overlaps with existing data occupying time range "+
			"%s for a time span of %s",
		newTR,
		existingTR,
		intersection.Span(),
	)
}

// NewPointWriteConflictError creates a new error that details a callers attempt to
// open a new writer on a region that already has existing data.
func NewPointWriteConflictError(ts telem.TimeStamp, existingTr telem.TimeRange) error {
	before, after := existingTr.Split(ts)
	return errors.Wrapf(
		ErrWriteConflict,
		"%s overlaps with existing data occupying time range %s. Timestamp occurs "+
			"%s after the start and %s before the end of the range",
		ts,
		existingTr,
		before.Span(),
		after.Span(),
	)
}

// NewRangeNotFoundError is returned when a resource for a specified time range is not
// found in the DB.
func NewRangeNotFoundError(tr telem.TimeRange) error {
	return errors.Wrapf(ErrRangeNotFound, "time range %s cannot be found", tr)
}

func newResourceInUseError(resource string, fileKey uint16) error {
	return errors.Newf("%s for file %d is in use and cannot be closed", resource, fileKey)
}
