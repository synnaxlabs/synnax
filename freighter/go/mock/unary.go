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
	"sync"

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
)

var (
	_ freighter.UnaryClient[any, any] = (*UnaryClient[any, any])(nil)
	_ freighter.UnaryServer[any, any] = (*UnaryServer[any, any])(nil)
)

// UnaryServer implements the freighter.UnaryServer interface using go channels as the
// transport. Use Network.UnaryServer to construct one.
type UnaryServer[RQ, RS freighter.Payload] struct {
	// mu guards handler.
	mu sync.RWMutex
	// handler answers every request sent to this server. Nil until BindHandler.
	handler freighter.UnaryHandler[RQ, RS]
	// network is the network the server is listening on.
	network *Network[RQ, RS]
	// address is where the server is reachable on its network.
	address address.Address
	reporter
	freighter.MiddlewareCollector
}

// BindHandler implements the freighter.UnaryServer interface.
func (s *UnaryServer[RQ, RS]) BindHandler(handler freighter.UnaryHandler[RQ, RS]) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handler = handler
}

// Address returns where the server is reachable on its network. Callers need it when
// they let Network.UnaryServer assign an address instead of naming one.
func (s *UnaryServer[RQ, RS]) Address() address.Address { return s.address }

func (s *UnaryServer[RQ, RS]) boundHandler() freighter.UnaryHandler[RQ, RS] {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.handler
}

func (s *UnaryServer[RQ, RS]) exec(
	ctx freighter.Context,
	req RQ,
) (res RS, oMD freighter.Context, err error) {
	h := s.boundHandler()
	oMD, err = s.Exec(
		ctx,
		freighter.FinalizerFunc(
			func(ctx freighter.Context) (oCtx freighter.Context, err error) {
				res, err = h(ctx, req)
				return freighter.Context{
					Context:  ctx,
					Target:   s.address,
					Protocol: protocol,
					Params:   make(freighter.Params),
				}, err
			},
		),
	)
	return res, oMD, err
}

// UnaryClient implements the freighter.UnaryClient interface using go channels as the
// transport. Use Network.UnaryClient to construct one.
type UnaryClient[RQ, RS freighter.Payload] struct {
	// network resolves a dialed target to a server.
	network *Network[RQ, RS]
	reporter
	freighter.MiddlewareCollector
}

// Send implements the freighter.UnaryClient interface. It returns an
// address.ErrNotFound error when no server on the network hosts the target or the
// server there has no handler bound.
func (c *UnaryClient[RQ, RS]) Send(
	ctx context.Context,
	target address.Address,
	req RQ,
) (res RS, err error) {
	_, err = c.Exec(
		freighter.Context{Context: ctx, Target: target, Protocol: protocol},
		freighter.FinalizerFunc(func(ctx freighter.Context) (freighter.Context, error) {
			var oMD freighter.Context
			route, ok := c.network.resolveUnaryTarget(target)
			if !ok || route.boundHandler() == nil {
				return oMD, address.NewTargetNotFoundError(target)
			}
			res, oMD, err = route.exec(ctx, req)
			c.network.appendEntry(target, req, res, err)
			return oMD, err
		}),
	)
	return res, err
}
