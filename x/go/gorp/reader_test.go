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
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Reader", Ordered, func() {
	var (
		db *gorp.DB
		tx gorp.Tx
	)
	BeforeAll(func() {
		db = gorp.Wrap(memkv.New())
		tx = db.OpenTx()
	})
	AfterAll(func() { Expect(db.Close()).To(Succeed()) })
	Describe("Iterator", func() {
		It("Should iterate over entries matching a type", func() {
			Expect(gorp.NewCreate[int32, entry]().
				Entries(&[]entry{{ID: 1, Data: "data"}, {ID: 2, Data: "data"}}).
				Exec(ctx, tx)).To(Succeed())
			iter := MustSucceed(gorp.WrapReader[int32, entry](tx).OpenIterator(gorp.IterOptions{}))
			Expect(iter.First()).To(BeTrue())
			Expect(iter.Value(ctx).Data).To(Equal("data"))
			Expect(iter.Next()).To(BeTrue())
			Expect(iter.Value(ctx).Data).To(Equal("data"))
			Expect(iter.Next()).To(BeFalse())
			Expect(iter.Close()).To(Succeed())
		})
	})
	Describe("Nexter", func() {
		It("Should iterate over entries matching a type", func() {
			Expect(gorp.NewCreate[int32, entry]().
				Entries(&[]entry{{ID: 1, Data: "data"}, {ID: 2, Data: "data"}}).
				Exec(ctx, tx)).To(Succeed())
			nexter := MustSucceed(gorp.WrapReader[int32, entry](tx).OpenNexter())
			v, ok := nexter.Next(ctx)
			Expect(ok).To(BeTrue())
			Expect(v.Data).To(Equal("data"))
			v, ok = nexter.Next(ctx)
			Expect(ok).To(BeTrue())
			Expect(nexter.Close()).To(Succeed())
		})
		It("Should correctly iterate over entries with a binary key", func() {
			Expect(gorp.NewCreate[[]byte, prefixEntry]().
				Entries(&[]prefixEntry{{ID: 1, Data: "data"}, {ID: 2, Data: "data"}}).
				Exec(ctx, tx)).To(Succeed())
			nexter := MustSucceed(gorp.WrapReader[[]byte, prefixEntry](tx).OpenNexter())
			v, ok := nexter.Next(ctx)
			Expect(ok).To(BeTrue())
			Expect(v.Data).To(Equal("data"))
			v, ok = nexter.Next(ctx)
			Expect(ok).To(BeTrue())
			Expect(nexter.Close()).To(Succeed())
		})
	})
})
