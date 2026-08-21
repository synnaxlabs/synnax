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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var _ = Describe("Pool", func() {
	Describe("Acquire", func() {
		It("Should dial the address through the passthrough resolver", func() {
			pool := DeferClose(fgrpc.OpenPool(
				"",
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			))
			conn := MustSucceed(pool.Acquire("localhost:12345"))
			Expect(conn.CanonicalTarget()).To(Equal("passthrough:///localhost:12345"))
		})

		It("Should prepend the target prefix to the address", func() {
			pool := DeferClose(fgrpc.OpenPool(
				"unix",
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			))
			conn := MustSucceed(pool.Acquire("run/synnax.sock"))
			Expect(conn.CanonicalTarget()).To(
				Equal("passthrough:///unix/run/synnax.sock"),
			)
		})
	})
})
