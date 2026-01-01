// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fgrpc

import (
	"context"

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"go.uber.org/zap"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

func parseServerContext(
	ctx context.Context,
	serviceName string,
	variant freighter.Variant,
) freighter.Context {

	oCtx := freighter.Context{
		Context:  ctx,
		Role:     freighter.Server,
		Target:   address.Address(serviceName),
		Protocol: Reporter.Protocol,
		Params:   make(freighter.Params),
		Variant:  variant,
	}

	p, ok := peer.FromContext(ctx)
	if !ok {
		zap.L().DPanic(
			"failed to get peer from context",
			zap.String("service", serviceName),
		)
		return oCtx
	}
	if tlsAuth, ok := p.AuthInfo.(credentials.TLSInfo); ok {
		oCtx.Sec.TLS.Used = true
		oCtx.Sec.TLS.ConnectionState = tlsAuth.State
	}
	grpcMD, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		zap.L().DPanic(
			"failed to get metadata from context",
			zap.String("service", serviceName),
		)
		return oCtx
	}

	for k, v := range grpcMD {
		oCtx.Params[k] = v[0]
	}

	return oCtx
}

func attachContext(ctx freighter.Context) freighter.Context {
	var toAppend []string
	for k, v := range ctx.Params {
		if vStr, ok := v.(string); ok {
			toAppend = append(toAppend, k, vStr)
		}
	}
	ctx.Context = metadata.AppendToOutgoingContext(ctx.Context, toAppend...)
	return ctx
}
