// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package service

import (
	"context"
	"io"

	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
)

// NewOpener returns a set of utility functions for opening collections of services
// that have a specific startup and shutdown order.
//
// NewOpener returns two functions, cleanup and ok. cleanup must be called in a defer
// statement directly after NewOpener returns, passing the final error.
//
//	var (
//		err error
//		closers xio.MultiCloser
//	)
//	cleanup, ok := NewOpener(ctx, &closers)
//	defer func() {
//		err = cleanup(err)
//	}()
//
// ok should be called after every call to open a service or every call to a function
// that modifies the value of err. If ok returns false, the calling function must
// immediately return. ok also accepts an optional io.Closer that will be added to the
// provided closer. This will be used to gracefully shut down the service in case
// any later service fails to open.
//
// if ok returns false, this means either err != nil or ctx.Err() != nil, in which case
// all previously opened services that have closers will be closed in the cleanup()
// statement.
//
// If err never occurs or ctx.Err() == nil throughout the opening process, the closers
// in closer will not be called.
//
// Here's an example:
//
//	func OpenLayer(ctx context.Context) (*Layer, error) {
//		var (
//			err error
//			myLayer = &Layer{closer: xio.MultiCloser{}}
//		)
//		cleanup, ok := NewOpener(ctx, &myLayer.closer)
//		defer func() {
//			err = cleanup(err)
//		}()
//		// If creating service 2 fails, then service 1 will be shut down correctly.
//		if myLayer.Service1, err = service1.Open(...); !ok(err, myLayer.Service1) {
//			return nil, err
//		}
//		// service2 does not have a closer for shutdown, so we can pass nil to ok.
//		if myLayer.Service2, err = service2.Provision(...); !ok(err, nil) {
//			return nil, err
//		}
//		return myLayer, nil
//	 }
func NewOpener(ctx context.Context, closer *xio.MultiCloser) (
	cleanup func(error) error,
	ok func(err error, c io.Closer) bool,
) {
	cleanup = func(err error) error {
		if err = errors.Combine(err, ctx.Err()); err != nil {
			err = errors.Combine(err, closer.Close())
		}
		return err
	}
	ok = func(err error, c io.Closer) bool {
		if err != nil {
			return false
		}
		if c != nil {
			*closer = append(*closer, c)
		}
		return ctx.Err() == nil
	}
	return cleanup, ok
}
