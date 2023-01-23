// Copyright 2023 Synnax Labs, Inc.
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
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

func parseMetaData(ctx context.Context, serviceName string) freighter.MD {
	grpcMD, ok := metadata.FromIncomingContext(ctx)
	md := freighter.MD{
		Target:   address.Address(serviceName),
		Protocol: reporter.Protocol,
		Params:   make(freighter.Params),
		Sec:      parseSecurityInfo(ctx),
	}
	if ok {
		for k, v := range grpcMD {
			md.Params[k] = v
		}
	}
	return md
}

func attachMetaData(ctx context.Context, md freighter.MD) {
	var toAppend []string
	for k, v := range md.Params {
		if vStr, ok := v.(string); ok {
			toAppend = append(toAppend, k, vStr)
		}
	}
	metadata.AppendToOutgoingContext(ctx, toAppend...)
}

func parseSecurityInfo(ctx context.Context) (info freighter.SecurityInfo) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return
	}
	if tlsAuth, ok := p.AuthInfo.(credentials.TLSInfo); ok {
		info.TLS.Used = true
		info.TLS.ConnectionState = tlsAuth.State
	}
	return
}
