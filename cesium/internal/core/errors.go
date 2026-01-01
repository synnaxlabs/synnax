// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package core

import (
	"fmt"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

var (
	// ErrChannelNotFound is returned when a particular channel cannot be found in the DB.
	ErrChannelNotFound = errors.Wrap(query.NotFound, "channel not found")
	// ErrOpenResource is returned when a resource cannot be closed because there are still
	// open resources on it (readers, writers, etc.).
	ErrOpenResource = errors.New("cannot close database because there are open resources on it")
	// ErrClosedResource is returns when an operation cannot be completed because the resource
	// being operator on is already closed.
	ErrClosedResource = errors.New("resource closed")
)

// NewErrChannelNotFound returns a wrapper around ErrChannelNotFound that includes the
// key of the missing channel.
func NewErrChannelNotFound(ch ChannelKey) error {
	return errors.Wrapf(ErrChannelNotFound, "channel with key %d not found", ch)
}

// NewErrResourceClosed returns a new error that wraps ErrClosedResource and includes the
// name of the resource that is closed. This is used to indicate that an operation cannot
// be completed because the resource is closed.
func NewErrResourceClosed(resourceName string) error {
	return errors.Wrapf(ErrClosedResource, "cannot complete operation on closed %s", resourceName)
}

// NewChannelErrWrapper returns a function that wraps an error with information about
// the channel that caused the error.
func NewChannelErrWrapper(ch Channel) func(error) error {
	msg := fmt.Sprintf("channel %v", ch)
	return func(err error) error { return errors.Wrap(err, msg) }
}
