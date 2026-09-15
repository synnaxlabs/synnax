// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mock

import (
	"context"
	"go/types"
	"sync"

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
)

var (
	_ freighter.UnaryClient[types.Nil, any] = (*UnaryClient[types.Nil, any])(nil)
	_ freighter.UnaryServer[types.Nil, any] = (*UnaryServer[types.Nil, any])(nil)
)

// UnaryServer implements the freighter.UnaryServer interface using go channels as
// the transport.
type UnaryServer[RQ, RS freighter.Payload] struct {
	// Network is the network the server is listening on.
	Network *Network[RQ, RS]
	// Address of the server on the network.
	Address address.Address
	// Handler is the handler that is called when a request is received.
	Handler freighter.UnaryHandler[RQ, RS]
	mu      sync.RWMutex
	reporter
	freighter.MiddlewareCollector
}

// BindHandler implements the freighter.UnaryServer interface.
func (s *UnaryServer[RQ, RS]) BindHandler(handler freighter.UnaryHandler[RQ, RS]) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Handler = handler
}

func (s *UnaryServer[RQ, RS]) exec(
	ctx freighter.Context,
	req RQ,
) (res RS, oMD freighter.Context, err error) {
	s.mu.RLock()
	h := s.Handler
	s.mu.RUnlock()
	oMD, err = s.Exec(
		ctx,
		freighter.FinalizerFunc(
			func(ctx freighter.Context) (oCtx freighter.Context, err error) {
				res, err = h(ctx, req)
				return freighter.Context{
					Context:  ctx,
					Target:   s.Address,
					Protocol: protocol,
					Params:   make(freighter.Params),
				}, err
			},
		),
	)
	return res, oMD, err
}

// UnaryClient implements the freighter.UnaryClient interface using go channels as the
// transport.
type UnaryClient[RQ, RS freighter.Payload] struct {
	// Network is the network the client is connected to.
	Network *Network[RQ, RS]
	reporter
	freighter.MiddlewareCollector
}

// Send implements the freighter.UnaryClient interface.
func (c *UnaryClient[RQ, RS]) Send(
	ctx context.Context,
	target address.Address,
	req RQ,
) (res RS, err error) {
	_, err = c.Exec(
		freighter.Context{Context: ctx, Target: target, Protocol: protocol},
		freighter.FinalizerFunc(func(ctx freighter.Context) (freighter.Context, error) {
			var oMD freighter.Context
			route, ok := c.Network.resolveUnaryTarget(target)
			if !ok {
				return oMD, address.NewTargetNotFoundError(target)
			}
			route.mu.RLock()
			hasHandler := route.Handler != nil
			route.mu.RUnlock()
			if !hasHandler {
				return oMD, address.NewTargetNotFoundError(target)
			}
			res, oMD, err = route.exec(ctx, req)
			c.Network.appendEntry(target, req, res, err)
			return oMD, err
		}),
	)
	return res, err
}
