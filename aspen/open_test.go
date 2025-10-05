// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package aspen_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/x/address"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Open", func() {
	var (
		db1 *aspen.DB
		db2 *aspen.DB
	)
	BeforeEach(func() {
		db1 = MustSucceed(aspen.Open(
			ctx,
			"",
			"localhost:22646",
			[]address.Address{},
			aspen.Bootstrap(),
			aspen.InMemory(),
			aspen.WithPropagationConfig(aspen.FastPropagationConfig),
		))
		db2 = MustSucceed(aspen.Open(
			ctx,
			"",
			"localhost:22647",
			[]address.Address{"localhost:22646"},
			aspen.InMemory(),
			aspen.WithPropagationConfig(aspen.FastPropagationConfig),
		))
	})
	AfterEach(func() {
		Expect(db1.Close()).To(Succeed())
		Expect(db2.Close()).To(Succeed())
	})
	It("Should be able to join two clusters", func() {
		Eventually(db1.Cluster.Nodes).Should(HaveLen(2))
		tx := db1.OpenTx()
		for range 10 {
			Expect(tx.Set(ctx, []byte("key"), []byte("value"), aspen.NodeKey(2))).To(Succeed())
		}
		Expect(tx.Commit(ctx)).To(Succeed())
	})
})
