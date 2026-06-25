// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer_test

import (
	"net"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	writergrpc "github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/framer/writer"
	"github.com/synnaxlabs/x/address"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestWriter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Distribution Transport gRPC Framer Writer Suite")
}

var (
	transport writergrpc.Transport
	addr      address.Address
)

var _ = ShouldNotLeakGoroutinesPerSpec()

var _ = BeforeEach(func() {
	lis := MustSucceed(net.Listen("tcp", "localhost:0"))
	addr = address.Address(lis.Addr().String())
	grpcServer := grpc.NewServer()
	pool := fgrpc.OpenPool("", grpc.WithTransportCredentials(insecure.NewCredentials()))
	transport = writergrpc.New(pool)
	transport.BindTo(grpcServer)
	go func() {
		defer GinkgoRecover()
		Expect(grpcServer.Serve(lis)).To(Succeed())
	}()
	DeferCleanup(grpcServer.GracefulStop)
	DeferCleanup(func() { Expect(pool.Close()).To(Succeed()) })
})
