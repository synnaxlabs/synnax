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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	. "github.com/synnaxlabs/freighter/grpc/testutil"
	"github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/framer/writer"
	"github.com/synnaxlabs/x/address"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/grpc"
)

func TestWriter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Distribution Transport gRPC Framer Writer Suite")
}

var (
	transport writer.Transport
	addr      address.Address
)

var _ = BeforeSuite(func() {
	ShouldNotLeakGoroutines()
	addr = StartServer(func(reg grpc.ServiceRegistrar, pool *fgrpc.Pool) {
		transport = writer.New(pool)
		transport.BindTo(reg)
	})
})

var _ = ShouldNotLeakGoroutinesPerSpec()
