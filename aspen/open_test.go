// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package aspen_test

import (
	"context"
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/x/address"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Open", func() {
	var (
		db1 *aspen.DB
		db2 *aspen.DB
	)
	BeforeEach(func() {
		db1 = MustSucceed(aspen.Open(
			context.Background(),
			"",
			"localhost:0",
			[]address.Address{},
			aspen.Bootstrap(),
			aspen.InMemory(),
			aspen.WithPropagationConfig(aspen.FastPropagationConfig),
		))
		db2 = MustSucceed(aspen.Open(
			context.Background(),
			"",
			"localhost:0",
			[]address.Address{db1.Cluster.Host().Address},
			aspen.InMemory(),
			aspen.WithPropagationConfig(aspen.FastPropagationConfig),
		))
	})
	AfterEach(func() {
		Expect(db1.Close()).To(Succeed())
		Expect(db2.Close()).To(Succeed())
	})
	It("Should be able to join two clusters", func(ctx SpecContext) {
		Eventually(db1.Cluster.Nodes).Should(HaveLen(2))
		tx := db1.OpenTx()
		for range 10 {
			Expect(
				tx.Set(ctx, []byte("key"), []byte("value"), aspen.NodeKey(2)),
			).To(Succeed())
		}
		Expect(tx.Commit(ctx)).To(Succeed())
	})
})

// portlessListener reports an address without a port, like a unix socket listener.
type portlessListener struct{ net.Listener }

func (p portlessListener) Addr() net.Addr {
	return &net.UnixAddr{Name: "aspen.sock", Net: "unix"}
}

var _ = Describe("WithListener", func() {
	It(
		"Should serve on the pre-bound listener and keep the configured host",
		func(ctx SpecContext) {
			lis := MustSucceed(net.Listen("tcp", "localhost:0"))
			port := lis.Addr().(*net.TCPAddr).Port
			db := MustSucceed(aspen.Open(
				ctx,
				"",
				"localhost:0",
				[]address.Address{},
				aspen.Bootstrap(),
				aspen.InMemory(),
				aspen.WithListener(lis),
			))
			addr := db.Cluster.Host().Address
			Expect(addr).To(Equal(address.Newf("localhost:%d", port)))
			Expect(MustSucceed(net.Dial("tcp", addr.String())).Close()).To(Succeed())
			Expect(db.Close()).To(Succeed())
		},
	)
	It(
		"Should return an error when the listener address has no port",
		func(ctx SpecContext) {
			lis := DeferClose(portlessListener{
				Listener: MustSucceed(net.Listen("tcp", "localhost:0")),
			})
			Expect(aspen.Open(
				ctx,
				"",
				"localhost:0",
				[]address.Address{},
				aspen.Bootstrap(),
				aspen.InMemory(),
				aspen.WithListener(lis),
			)).Error().To(MatchError(validate.ErrValidation))
		},
	)
})
