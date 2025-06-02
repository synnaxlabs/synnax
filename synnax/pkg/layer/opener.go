// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package layer

import (
	"context"
	"io"

	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
)

// NewOpener returns a set of utility functions for opening 'layers' that contain
// collections of services.
//
// NewOpener returns two functions, cleanup and ok. cleanup must be called in a defer
// statement directly after NewOpener returns.
//
//	var (
//		err error
//		closers xio.MultiCloser
//	)
//	cleanup, ok := NewOpener(ctx, &err, &closer)
//	defer cleanup()
//
// ok should be called after every call to open a service or every call to a function
// that modifies the value of err. If ok returns false, the calling function must
// immediately return. ok also accepts an optional io.Closer that will be added to the
// provided closer. This will be used to gracefully shut down the service in case
// any subsequent service fails to open.
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
//		func OpenLayer(ctx context.Context) (*Layer, error) {
//		 	var (
//				err eror
//				myLayer = &Layer{closer: xio.MultiCloser{}}
//			)
//			cleanup, ok := layer.NewOpener(ctx, &err, &myLayer.closer)
//			defer cleanup()
//			// If creating service 2 fails, then service 1 will be shut down correctly.
//			if myLayer.Service1, err = service1.Open(...); !ok(myLayer.Service1) {
//				return nil, err
//			}
//			// service2 does not have a closer for shutdown, so we can pass nil to ok.
//			if myLayer.Service2, err = service2.New(...); !ok(nil) {
//				return nil, err
//			}
//			return myLayer, nil
//	 }
func NewOpener(ctx context.Context, err *error, closer *xio.MultiCloser) (
	cleanup func(),
	ok func(c io.Closer) bool,
) {
	cleanup = func() {
		if *err == nil {
			*err = ctx.Err()
		}
		if *err != nil {
			*err = errors.Combine(*err, closer.Close())
		}
	}
	ok = func(c io.Closer) bool {
		if *err != nil {
			return false
		}
		if c != nil {
			*closer = append(*closer, c)
		}
		*err = ctx.Err()
		return *err == nil
	}
	return cleanup, ok
}
