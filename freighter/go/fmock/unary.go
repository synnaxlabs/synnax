// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fmock

import (
	"context"

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
)

// UnaryServer implements the freighter.UnaryServer interface using go channels as the
// transport.
type UnaryServer[RQ, RS freighter.Payload] struct {
	// Network is the network the server is listening on. In the case where a server is
	// directly connected to a client, this is nil.
	Network *Network[RQ, RS]
	// Address of the server on the network. This field is only defined if network is
	// not nil.
	Address address.Address
	// Handler is the handler that is called when a request is received.
	Handler func(context.Context, RQ) (RS, error)
	freighter.Reporter
	freighter.MiddlewareCollector
}

var _ freighter.UnaryServer[any, any] = (*UnaryServer[any, any])(nil)

// BindHandler implements the freighter.UnaryServer interface.
func (us *UnaryServer[RQ, RS]) BindHandler(handler func(context.Context, RQ) (RS, error)) {
	us.Handler = handler
}

func (us *UnaryServer[RQ, RS]) exec(ctx freighter.Context, req RQ) (RS, freighter.Context, error) {
	var res RS
	oMD, err := us.MiddlewareCollector.Exec(
		ctx,
		freighter.MiddlewareHandler(func(ctx freighter.Context) (freighter.Context, error) {
			var err error
			res, err = us.Handler(ctx, req)
			return freighter.Context{
				Context:  ctx,
				Target:   us.Address,
				Protocol: us.Protocol,
				Params:   make(freighter.Params),
			}, err
		}),
	)
	return res, oMD, err
}

// UnaryClient implements the freighter.UnaryClient interface using go channels as the
// transport.
type UnaryClient[RQ, RS freighter.Payload] struct {
	// Network is the network the client is connected to.
	Network *Network[RQ, RS]
	freighter.Reporter
	freighter.MiddlewareCollector
}

var _ freighter.UnaryClient[any, any] = (*UnaryClient[any, any])(nil)

// Send implements the freighter.UnaryClient interface.
func (uc *UnaryClient[RQ, RS]) Send(
	ctx context.Context,
	target address.Address,
	req RQ,
) (RS, error) {
	var res RS
	_, err := uc.MiddlewareCollector.Exec(
		freighter.Context{Context: ctx, Target: target, Protocol: uc.Reporter.Protocol},
		freighter.MiddlewareHandler(func(ctx freighter.Context) (freighter.Context, error) {
			var (
				handler func(freighter.Context, RQ) (RS, freighter.Context, error)
				oMD     freighter.Context
				err     error
			)

			if uc.Network != nil {
				route, ok := uc.Network.resolveUnaryTarget(target)
				if !ok || route.Handler == nil {
					return oMD, address.NewErrTargetNotFound(target)
				}
				handler = route.exec
			}
			res, oMD, err = handler(ctx, req)
			if uc.Network != nil {
				uc.Network.appendEntry(target, req, res, err)
			}
			return oMD, err
		}))
	return res, err
}
