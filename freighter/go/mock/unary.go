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
	"github.com/synnaxlabs/x/types"
)

var (
	_ freighter.UnaryClient[any, any] = (*UnaryClient[any, any])(nil)
	_ freighter.UnaryServer[any, any] = (*UnaryServer[any, any])(nil)
)

// UnaryServer implements the freighter.UnaryServer interface using go channels as the
// transport. Use Network.UnaryServer to construct one.
type UnaryServer[RQ, RS freighter.Payload] struct {
	mu      sync.RWMutex
	handler freighter.UnaryHandler[RQ, RS]
	network *Network[RQ, RS]
	address address.Address
	reporter
	freighter.MiddlewareCollector
}

// BindHandler implements the freighter.UnaryServer interface.
func (us *UnaryServer[RQ, RS]) BindHandler(handler freighter.UnaryHandler[RQ, RS]) {
	us.mu.Lock()
	defer us.mu.Unlock()
	us.handler = handler
}

// Address returns where the server is reachable on its network. Callers need it when
// they let Network.UnaryServer assign an address instead of naming one.
func (us *UnaryServer[RQ, RS]) Address() address.Address { return us.address }

func (us *UnaryServer[RQ, RS]) boundHandler() freighter.UnaryHandler[RQ, RS] {
	us.mu.RLock()
	defer us.mu.RUnlock()
	return us.handler
}

func (us *UnaryServer[RQ, RS]) exec(
	ctx freighter.Context,
	req RQ,
) (RS, freighter.Context, error) {
	var (
		res RS
		oMD freighter.Context
		err error
	)
	h := us.boundHandler()
	oMD, err = us.Exec(
		ctx,
		freighter.FinalizerFunc(
			func(ctx freighter.Context) (freighter.Context, error) {
				res, err = h(ctx, req)
				if err != nil {
					return freighter.Context{}, err
				}
				return freighter.Context{
					Context:  ctx,
					Target:   us.address,
					Protocol: protocol,
					Params:   make(freighter.Params),
				}, nil
			},
		),
	)
	if err != nil {
		return types.Zero[RS](), oMD, err
	}
	return res, oMD, nil
}

// UnaryClient implements the freighter.UnaryClient interface using go channels as the
// transport. Use Network.UnaryClient to construct one.
type UnaryClient[RQ, RS freighter.Payload] struct {
	network *Network[RQ, RS]
	reporter
	freighter.MiddlewareCollector
}

// Send implements the freighter.UnaryClient interface. It returns an
// address.ErrNotFound error when no server on the network hosts the target or the
// server there has no handler bound.
func (uc *UnaryClient[RQ, RS]) Send(
	ctx context.Context,
	target address.Address,
	req RQ,
) (res RS, err error) {
	_, err = uc.Exec(
		freighter.Context{Context: ctx, Target: target, Protocol: protocol},
		freighter.FinalizerFunc(func(ctx freighter.Context) (freighter.Context, error) {
			var oMD freighter.Context
			route, ok := uc.network.resolveUnaryTarget(target)
			if !ok || route.boundHandler() == nil {
				return oMD, address.NewTargetNotFoundError(target)
			}
			res, oMD, err = route.exec(ctx, req)
			uc.network.appendEntry(target, req, res, err)
			return oMD, err
		}),
	)
	return res, err
}
