// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package io

import (
	"github.com/synnaxlabs/x/errutil"
	"io"
)

// CloserFunc allows a function to implement the io.Closer interface.
type CloserFunc func() error

var _ io.Closer = CloserFunc(nil)

// Close implements io.Closer.
func (c CloserFunc) Close() error { return c() }

type NopCloserFunc func()

func (c NopCloserFunc) Close() error {
	c()
	return nil
}

type MultiCloser []io.Closer

var _ io.Closer = MultiCloser(nil)

func (c MultiCloser) Close() error {
	ca := errutil.NewCatch(errutil.WithAggregation())
	for _, closer := range c {
		ca.Exec(closer.Close)
	}
	return ca.Error()
}
