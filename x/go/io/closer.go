// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package io

import (
	"io"

	"github.com/synnaxlabs/x/errors"
)

// CloserFunc allows a function to implement the io.Closer interface.
type CloserFunc func() error

var _ io.Closer = CloserFunc(nil)

// Close implements io.Closer.
func (c CloserFunc) Close() error { return c() }

type nopCloser struct{}

var _ io.Closer = nopCloser{}

func (nopCloser) Close() error { return nil }

// NopCloser is an io.Closer that does nothing.
var NopCloser io.Closer = nopCloser{}

// NoFailCloserFunc is an io.Closer that executes a function and returns nil.
type NoFailCloserFunc func()

var _ io.Closer = NoFailCloserFunc(nil)

// Close implements io.Closer.
func (c NoFailCloserFunc) Close() error { c(); return nil }

// MultiCloser is a collection of io.Closer objects that can be closed together. The
// Close method will close each closer in the collection in the reverse order they were
// added.
type MultiCloser []io.Closer

var _ io.Closer = MultiCloser(nil)

func (c MultiCloser) Close() error {
	ca := errors.NewCatcher()
	for i := len(c) - 1; i >= 0; i-- {
		closer := c[i]
		ca.Exec(closer.Close)
	}
	return ca.Error()
}
