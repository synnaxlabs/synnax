// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package index_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp/internal/index"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Lookup", func() {
	var l *index.Lookup[*scope, int, string]
	BeforeEach(func() {
		l = index.NewLookup[*scope, int, string]()
		insert, finish := l.Populate()
		insert(1, "a")
		insert(2, "b")
		finish(nil)
	})

	Describe("StageRelease", func() {
		It("Should hide a released key from Get and Holder but not from Value", func() {
			txn := &scope{}
			Expect(l.StageRelease(txn, 1)).To(Succeed())
			Expect(l.Get(txn, "a")).To(BeEmpty())
			_, held := MustSucceed2(l.Holder(txn, "a", nil))
			Expect(held).To(BeFalse())
			v, ok := MustSucceed2(l.Value(txn, 1))
			Expect(ok).To(BeTrue())
			Expect(v).To(Equal("a"))
			Expect(l.Get(nil, "a")).To(ConsistOf(1))
		})

		It("Should delete a released key when its scope commits", func() {
			txn := &scope{}
			Expect(l.StageRelease(txn, 1)).To(Succeed())
			txn.end(true)
			_, ok := MustSucceed2(l.Value(nil, 1))
			Expect(ok).To(BeFalse())
			Expect(l.Get(nil, "a")).To(BeEmpty())
		})

		It("Should keep a released key when its scope does not commit", func() {
			txn := &scope{}
			Expect(l.StageRelease(txn, 1)).To(Succeed())
			txn.end(false)
			Expect(l.Get(nil, "a")).To(ConsistOf(1))
		})

		It("Should end a release when the scope stages the key again", func() {
			txn := &scope{}
			Expect(l.StageRelease(txn, 1)).To(Succeed())
			l.Stage(txn, 1, "c")
			v, ok := MustSucceed2(l.Value(txn, 1))
			Expect(ok).To(BeTrue())
			Expect(v).To(Equal("c"))
			Expect(l.Get(txn, "c")).To(ConsistOf(1))
		})

		It("Should delete the key at once against the zero scope", func() {
			Expect(l.StageRelease(nil, 1)).To(Succeed())
			_, ok := MustSucceed2(l.Value(nil, 1))
			Expect(ok).To(BeFalse())
		})
	})

	Describe("Value", func() {
		It("Should read the changes of a scope while it commits", func() {
			txn := &scope{}
			l.Stage(txn, 3, "c")
			l.StageDelete(txn, 2)
			_, ok := MustSucceed2(l.Value(nil, 3))
			Expect(ok).To(BeFalse())
			txn.committing = true
			v, ok := MustSucceed2(l.Value(nil, 3))
			Expect(ok).To(BeTrue())
			Expect(v).To(Equal("c"))
			_, ok = MustSucceed2(l.Value(nil, 2))
			Expect(ok).To(BeFalse())
		})

		It("Should read the scope's own changes over a commit in flight", func() {
			committing, txn := &scope{committing: true}, &scope{}
			l.Stage(committing, 1, "c")
			l.Stage(txn, 1, "d")
			v, ok := MustSucceed2(l.Value(txn, 1))
			Expect(ok).To(BeTrue())
			Expect(v).To(Equal("d"))
		})
	})
})
