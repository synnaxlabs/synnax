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
	"go/types"
)

var (
	_ freighter.UnaryClient[types.Nil, any] = (*UnaryClient[types.Nil, any])(nil)
	_ freighter.UnaryServer[types.Nil, any] = (*UnaryServer[types.Nil, any])(nil)
)

// NewUnaryPair creates a new unary client and server pair that are directly linked to
// one another i.e. dialing any target on the client will call the server's handler.
func NewUnaryPair[RQ, RS freighter.Payload]() (*UnaryServer[RQ, RS], *UnaryClient[RQ, RS]) {
	us := &UnaryServer[RQ, RS]{}
	uc := &UnaryClient[RQ, RS]{server: us}
	return us, uc
}

// UnaryServer implements the freighter.UnaryServer interface using go channels as
// the transport.
type UnaryServer[RQ, RS freighter.Payload] struct {
	// Network is the network the server is listening on. In the case where a server
	// is directly connected to a client (i.e. via NewUnaryPair), this is nil.
	Network *Network[RQ, RS]
	// Address of the server on the network. This field is only defined if network is
	// not nil.
	Address address.Address
	// Handler is the handler that is called when a request is received.
	Handler func(context.Context, RQ) (RS, error)
	freighter.Reporter
	freighter.MiddlewareCollector
}

// BindHandler implements the freighter.Unary interface.
func (u *UnaryServer[RQ, RS]) BindHandler(handler func(context.Context, RQ) (RS, error)) {
	u.Handler = handler
}

func (u *UnaryServer[RQ, RS]) exec(ctx freighter.Context, req RQ) (res RS, oMD freighter.Context, err error) {
	oMD, err = u.MiddlewareCollector.Exec(
		ctx,
		freighter.FinalizerFunc(func(ctx freighter.Context) (oCtx freighter.Context, err error) {
			res, err = u.Handler(ctx, req)
			return freighter.Context{
				Context:  ctx,
				Target:   u.Address,
				Protocol: u.Protocol,
				Params:   make(freighter.Params),
			}, err
		}),
	)
	return res, oMD, err
}

// UnaryClient implements the freighter.UnaryCLinet interface using go channels as the
// transport.
type UnaryClient[RQ, RS freighter.Payload] struct {
	// Network is the network the client is connected to. In the case where a client
	// is directly connected to a server (i.e. via NewUnaryPair), this is nil.
	Network *Network[RQ, RS]
	server  *UnaryServer[RQ, RS]
	freighter.Reporter
	freighter.MiddlewareCollector
}

// Send implements the freighter.Unary interface.
func (u *UnaryClient[RQ, RS]) Send(
	ctx context.Context,
	target address.Address,
	req RQ,
) (res RS, err error) {
	_, err = u.MiddlewareCollector.Exec(
		freighter.Context{Context: ctx, Target: target, Protocol: u.Reporter.Protocol},
		freighter.FinalizerFunc(func(ctx freighter.Context) (freighter.Context, error) {
			var (
				handler func(freighter.Context, RQ) (RS, freighter.Context, error)
				oMD     freighter.Context
			)

			// A non nil server means we're tied up in a unary pair, so we can just
			// use the server's handler.
			if u.server != nil {
				handler = u.server.exec
			} else if u.Network != nil {
				route, ok := u.Network.resolveUnaryTarget(target)
				if !ok || route.Handler == nil {
					return oMD, address.TargetNotFound(target)
				}
				handler = route.exec
			}
			res, oMD, err = handler(ctx, req)
			if u.Network != nil {
				u.Network.appendEntry(target, req, res, err)
			}
			return oMD, err
		}))
	return res, err
}
