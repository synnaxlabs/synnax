// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resource

import "github.com/synnaxlabs/x/errors"

var (
	// ErrOpen is returned when a resource cannot be closed because there are still open
	// resources on it (readers, writers, etc.).
	ErrOpen = errors.New("cannot close database because there are open resources on it")
	// ErrClosed is returns when an operation cannot be completed because the resource
	// being operated on is already closed.
	ErrClosed = errors.New("resource closed")
)

// NewErrClosed returns a new error that wraps ErrClosed and includes the name of the
// resource that is closed. This is used to indicate that an operation cannot be
// completed because the resource is closed.
func NewErrClosed(resourceName string) error {
	return errors.Wrapf(
		ErrClosed,
		"cannot complete operation on closed %s",
		resourceName,
	)
}
