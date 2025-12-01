// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
)

var _ = Describe("Writer", Ordered, func() {
	var (
		db *gorp.DB
		tx gorp.Tx
	)
	BeforeAll(func() {
		db = gorp.Wrap(memkv.New())
	})
	AfterAll(func() { Expect(db.Close()).To(Succeed()) })
	BeforeEach(func() {
		tx = db.OpenTx()
	})
	AfterEach(func() { Expect(tx.Close()).To(Succeed()) })
	It("Should wrap a key-value writer with an encoder", func() {
		w := gorp.WrapWriter[int32, entry](tx)
		Expect(w.Set(ctx, entry{ID: 1, Data: "Two"})).To(Succeed())
		Expect(w.Delete(ctx, 1)).To(Succeed())
	})
})
