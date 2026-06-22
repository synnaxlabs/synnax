// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	"net"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	channelgrpc "github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/channel"
	"github.com/synnaxlabs/x/address"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestChannel(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Distribution Transport gRPC Channel Suite")
}

var (
	transport  channelgrpc.Transport
	addr       address.Address
	grpcServer *grpc.Server
)

var _ = BeforeSuite(func() {
	lis := MustSucceed(net.Listen("tcp", "localhost:0"))
	addr = address.Address(lis.Addr().String())
	grpcServer = grpc.NewServer()
	pool := fgrpc.NewPool("", grpc.WithTransportCredentials(insecure.NewCredentials()))
	transport = channelgrpc.New(pool)
	transport.BindTo(grpcServer)
	go func() {
		defer GinkgoRecover()
		Expect(grpcServer.Serve(lis)).To(Succeed())
	}()
	DeferCleanup(grpcServer.GracefulStop)
})
