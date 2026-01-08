// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package start

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/security"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	insecureGRPC "google.golang.org/grpc/credentials/insecure"
)

func configureClientGRPC(
	sec security.Provider,
	insecure bool,
) *fgrpc.Pool {
	return fgrpc.NewPool(
		"",
		grpc.WithTransportCredentials(getClientGRPCTransportCredentials(sec, insecure)),
	)
}

func getClientGRPCTransportCredentials(
	sec security.Provider,
	insecure bool,
) credentials.TransportCredentials {
	return lo.Ternary(
		insecure,
		insecureGRPC.NewCredentials(),
		credentials.NewTLS(sec.TLS()),
	)
}
