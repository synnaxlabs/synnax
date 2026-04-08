// Copyright 2026 Synnax Labs, Inc.
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
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Reader", func() {
	var tx gorp.Tx
	BeforeEach(func() {
		tx = db.OpenTx()
	})
	Describe("Iterator", func() {
		It("Should iterate over entries matching a type", func(ctx SpecContext) {
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
		It("Should return nil and accumulate an error when decoding fails", func(ctx SpecContext) {
			Expect(gorp.NewCreate[int32, entry]().
				Entries(&[]entry{{ID: 99, Data: "valid"}}).
				Exec(ctx, tx)).To(Succeed())
			Expect(tx.Commit(ctx)).To(Succeed())

			iter := MustSucceed(gorp.WrapReader[int32, entry](tx).OpenIterator(gorp.IterOptions{}))
			var rawKey []byte
			for iter.First(); iter.Valid(); iter.Next() {
				v := iter.Value(ctx)
				if v != nil && v.ID == 99 {
					rawKey = make([]byte, len(iter.Key()))
					copy(rawKey, iter.Key())
					break
				}
			}
			Expect(rawKey).ToNot(BeNil())
			Expect(iter.Close()).To(Succeed())

			kvTx := kvDB.OpenTx()
			Expect(kvTx.Set(ctx, rawKey, []byte("corrupt"))).To(Succeed())
			Expect(kvTx.Commit(ctx)).To(Succeed())
			Expect(kvTx.Close()).To(Succeed())

			tx2 := db.OpenTx()
			iter2 := MustSucceed(gorp.WrapReader[int32, entry](tx2).OpenIterator(gorp.IterOptions{}))
			found := false
			for iter2.First(); iter2.Valid(); iter2.Next() {
				v := iter2.Value(ctx)
				if v == nil && iter2.Error() != nil {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
			Expect(iter2.Error()).To(HaveOccurred())
			Expect(iter2.Close()).To(Succeed())

			cleanupTx := kvDB.OpenTx()
			Expect(cleanupTx.Delete(ctx, rawKey)).To(Succeed())
			Expect(cleanupTx.Commit(ctx)).To(Succeed())
			Expect(cleanupTx.Close()).To(Succeed())
		})
	})
	Describe("Nexter", func() {
		It("Should iterate over entries matching a type", func(ctx SpecContext) {
			Expect(gorp.NewCreate[int32, entry]().
				Entries(&[]entry{{ID: 1, Data: "data"}, {ID: 2, Data: "data"}}).
				Exec(ctx, tx)).To(Succeed())
			nexter, closer := MustSucceed2(gorp.WrapReader[int32, entry](tx).OpenNexter(ctx))
			for v := range nexter {
				Expect(v.Data).To(Equal("data"))
			}
			Expect(closer.Close()).To(Succeed())
		})
		It("Should correctly iterate over entries with a binary key", func(ctx SpecContext) {
			Expect(gorp.NewCreate[[]byte, prefixEntry]().
				Entries(&[]prefixEntry{{ID: 1, Data: "data"}, {ID: 2, Data: "data"}}).
				Exec(ctx, tx)).To(Succeed())
			nexter, closer := MustSucceed2(gorp.WrapReader[[]byte, prefixEntry](tx).OpenNexter(ctx))
			for v := range nexter {
				Expect(v.Data).To(Equal("data"))
			}
			Expect(closer.Close()).To(Succeed())
		})
	})
})
