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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/gorp"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Reader", func() {
	var (
		ctx context.Context
		tx  gorp.Tx
	)
	BeforeEach(func() {
		ctx = context.Background()
		tx = db.OpenTx()
	})
	Describe("Iterator", func() {
		It("Should iterate over entries matching a type", func() {
			Expect(gorp.NewCreate[int, entry]().
				Entries(&[]entry{{ID: 1, Data: "data"}, {ID: 2, Data: "data"}}).
				Exec(ctx, tx)).To(Succeed())
			iter := MustSucceed(gorp.WrapReader[int, entry](tx).OpenIterator(gorp.IterOptions{}))
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
			Expect(gorp.NewCreate[int, entry]().
				Entries(&[]entry{{ID: 1, Data: "data"}, {ID: 2, Data: "data"}}).
				Exec(ctx, tx)).To(Succeed())
			nexter := MustSucceed(gorp.WrapReader[int, entry](tx).OpenNexter())
			v := MustBeOk(nexter.Next(ctx))
			Expect(v.Data).To(Equal("data"))
			v = MustBeOk(nexter.Next(ctx))
			Expect(nexter.Close()).To(Succeed())
		})
		It("Should correctly iterate over entries with a binary key", func() {
			Expect(gorp.NewCreate[[]byte, prefixEntry]().
				Entries(&[]prefixEntry{{ID: 1, Data: "data"}, {ID: 2, Data: "data"}}).
				Exec(ctx, tx)).To(Succeed())
			nexter := MustSucceed(gorp.WrapReader[[]byte, prefixEntry](tx).OpenNexter())
			v := MustBeOk(nexter.Next(ctx))
			Expect(v.Data).To(Equal("data"))
			v = MustBeOk(nexter.Next(ctx))
			Expect(nexter.Close()).To(Succeed())
		})
	})
})
