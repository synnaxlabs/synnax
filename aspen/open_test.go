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
)

var _ = Describe("Open", func() {
	var (
		db1 *aspen.DB
		db2 *aspen.DB
	)
	BeforeEach(func() {
		var err error
		db1, err = aspen.Open(
			ctx,
			"",
			"localhost:22646",
			[]address.Address{},
			aspen.Bootstrap(),
			aspen.MemBacked(),
			aspen.WithPropagationConfig(aspen.FastPropagationConfig),
		)
		Expect(err).ToNot(HaveOccurred())
		db2, err = aspen.Open(
			ctx,
			"",
			"localhost:22647",
			[]address.Address{"localhost:22646"},
			aspen.MemBacked(),
			aspen.WithPropagationConfig(aspen.FastPropagationConfig),
		)

	})
	AfterEach(func() {
		Expect(db1.Close()).To(Succeed())
		Expect(db2.Close()).To(Succeed())
	})
	It("Should be able to join two clusters", func() {
		Eventually(db1.Cluster.Nodes).Should(HaveLen(2))
		tx := db1.OpenTx()
		for i := 0; i < 10; i++ {
			Expect(tx.Set(ctx, []byte("key"), []byte("value"), aspen.NodeKey(2))).To(Succeed())
		}
		Expect(tx.Commit(ctx)).To(Succeed())
	})
})
