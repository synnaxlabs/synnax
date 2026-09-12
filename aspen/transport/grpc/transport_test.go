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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/aspen/transport/grpc"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Transport", func() {
	var tr *grpc.Transport
	BeforeEach(func() { tr = grpc.New(DeferClose(fgrpc.OpenPool(""))) })
	Describe("Serve", func() {
		It("Should accept connections until Close", func() {
			Expect(tr.Configure(alamos.Instrumentation{})).To(Succeed())
			lis := MustSucceed(net.Listen("tcp", "localhost:0"))
			Expect(tr.Serve(lis)).To(Succeed())
			Eventually(func() error {
				conn, err := net.Dial("tcp", lis.Addr().String())
				if err != nil {
					return err
				}
				return conn.Close()
			}).Should(Succeed())

			By("Releasing the listener on Close")
			Expect(tr.Close()).To(Succeed())
			Expect(lis.Close()).To(MatchError(net.ErrClosed))
		})
	})
	Describe("Close", func() {
		It("Should succeed on a Transport that never served", func() {
			Expect(tr.Configure(alamos.Instrumentation{})).To(Succeed())
			Expect(tr.Close()).To(Succeed())
		})
	})
})
