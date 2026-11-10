// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package dispatch builds the handlers of the endpoints that apply actions to a record.
package dispatch

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
)

// Writer is a writer that stamps a dispatch key onto the actions it attaches.
type Writer[W any] interface {
	// WithDispatchKey returns a copy of the writer that stamps dispatchKey.
	WithDispatchKey(dispatchKey string) W
}

// Action is an action that a client sends.
type Action interface {
	// CheckCoreOnly returns an error wrapping actions.ErrCoreOnly when only the Core
	// can dispatch the action.
	CheckCoreOnly() error
}

// Handler applies the actions of a request to one record in tx. The response holds the
// actions that the Core applied, which are the actions that the commit broadcasts with
// the dispatch key of the request.
type Handler[K comparable, A any] = func(
	context.Context,
	gorp.Tx,
	actions.DispatchRequest[K, A],
) (actions.DispatchResponse[A], error)

// New returns a Handler that applies actions with apply, on the writer that open
// returns for the transaction, with the dispatch key of the request. The response holds
// the actions that apply returns. The Handler applies nothing and returns an error
// wrapping actions.ErrCoreOnly when the request holds an action that only the Core can
// dispatch. An error returns a zero response.
func New[W Writer[W], K comparable, A Action](
	open func(gorp.Tx) W,
	apply func(W, context.Context, K, ...A) ([]A, error),
) Handler[K, A] {
	return func(
		ctx context.Context,
		tx gorp.Tx,
		req actions.DispatchRequest[K, A],
	) (actions.DispatchResponse[A], error) {
		for _, a := range req.Actions {
			if err := a.CheckCoreOnly(); err != nil {
				return actions.DispatchResponse[A]{}, err
			}
		}
		w := open(tx).WithDispatchKey(req.DispatchKey)
		applied, err := apply(w, ctx, req.Key, req.Actions...)
		if err != nil {
			return actions.DispatchResponse[A]{}, err
		}
		return actions.DispatchResponse[A]{Actions: applied}, nil
	}
}
