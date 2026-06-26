// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package grpc_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	. "github.com/synnaxlabs/freighter/grpc/testutil"
	transportgrpc "github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc"
	"github.com/synnaxlabs/x/address"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/grpc"
)

func TestGRPC(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Distribution Transport gRPC Suite")
}

var (
	transport transportgrpc.Transport
	addr      address.Address
)

var _ = BeforeSuite(func() {
	ShouldNotLeakGoroutines()
	addr = StartServer(func(reg grpc.ServiceRegistrar, pool *fgrpc.Pool) {
		transport = transportgrpc.New(pool)
		transport.BindTo(reg)
	})
})
