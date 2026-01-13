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
var ErrNotFound = errors.Wrap(query.NotFound, "channel not found")

// NewNotFoundError returns a wrapper around ErrNotFound that includes the key of the
// the missing channel.
func NewNotFoundError(ch Key) error {
	return errors.Wrapf(ErrNotFound, "channel with key %d not found", ch)
}

// NewErrorWrapper returns a function that wraps an error with information about the
// channel that caused the error.
func NewErrorWrapper(ch Channel) func(error) error {
	msg := fmt.Sprintf("channel %v", ch)
	return func(err error) error { return errors.Wrap(err, msg) }
}
