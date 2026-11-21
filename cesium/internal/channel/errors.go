// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"fmt"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// ErrNotFound is returned when a particular channel cannot be found in the DB.
var ErrNotFound = errors.Wrap(query.ErrNotFound, "channel not found")

// NewNotFoundError returns a wrapper around ErrNotFound that includes the key of the
// the missing channel.
func NewNotFoundError(ch Key) error {
	return errors.Wrapf(ErrNotFound, "channel with key %d not found", ch)
}

// Error is an error of one channel. It carries the key of the channel, never its name.
type Error struct {
	Err error
	Key Key
}

// Error implements error.
func (e Error) Error() string { return fmt.Sprintf("channel <%d>: %v", e.Key, e.Err) }

// Unwrap returns the error of the channel.
func (e Error) Unwrap() error { return e.Err }

// Wrap returns err as an Error of the channel with key, or nil when err is nil.
func Wrap(key Key, err error) error {
	if err == nil {
		return nil
	}
	return Error{Key: key, Err: err}
}

// NewErrorWrapper returns a function that wraps an error as an Error of ch.
func NewErrorWrapper(ch Channel) func(error) error {
	return func(err error) error { return Wrap(ch.Key, err) }
}
