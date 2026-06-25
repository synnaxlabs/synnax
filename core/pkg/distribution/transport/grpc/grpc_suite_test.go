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
	"net"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	transportgrpc "github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc"
	"github.com/synnaxlabs/x/address"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestGRPC(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Distribution Transport gRPC Suite")
}

var (
	transport transportgrpc.Transport
	addr      address.Address
)

var _ = ShouldNotLeakGoroutinesPerSpec()

var _ = BeforeEach(func() {
	lis := MustSucceed(net.Listen("tcp", "localhost:0"))
	addr = address.Address(lis.Addr().String())
	pool := DeferClose(fgrpc.OpenPool(
		"",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	))
	transport = transportgrpc.New(pool)
	grpcServer := grpc.NewServer()
	for _, bt := range transport.BindableTransports() {
		bt.BindTo(grpcServer)
	}
	go func() {
		defer GinkgoRecover()
		Expect(grpcServer.Serve(lis)).To(Succeed())
	}()
	DeferCleanup(grpcServer.GracefulStop)
})
