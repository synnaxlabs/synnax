// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/x/address"
)

var _ = Describe("Open", func() {
	var (
		db1 aspen.DB
		db2 aspen.DB
	)
	BeforeEach(func() {
		var err error
		db1, err = aspen.Open(
			alamos.Dev("aspen", false, "aspen.test.open.db1"),
			"",
			"localhost:22646",
			[]address.Address{},
			aspen.Bootstrap(),
			aspen.MemBacked(),
			aspen.WithPropagationConfig(aspen.FastPropagationConfig),
		)
		Expect(err).ToNot(HaveOccurred())
		db2, err = aspen.Open(
			alamos.Dev("aspen", false, "aspen.test.open.db2"),
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
	FIt("Should be able to join two clusters", func() {
		Eventually(db1.Nodes).Should(HaveLen(2))
		b := db1.NewBatch()
		ctx := alamos.Dev("aspen", false, "aspen.test.open.db1.batch")
		for i := 0; i < 10; i++ {
			Expect(b.Set(ctx, []byte("key"), []byte("value"), aspen.NodeID(2))).To(Succeed())
		}
		Expect(b.Commit(ctx)).To(Succeed())
	})
})
