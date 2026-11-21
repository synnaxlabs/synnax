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

var _ = Describe("Lookup values and holders", func() {
	var (
		l   *index.Lookup[*scope, int, string]
		txn *scope
	)
	BeforeEach(func() {
		l = index.NewLookup[*scope, int, string]()
		insert, finish := l.Populate()
		insert(1, "a")
		insert(2, "b")
		finish(nil)
		txn = &scope{}
	})

	Describe("Value", func() {
		It("Should return the committed value of a key", func() {
			v, ok := MustSucceed2(l.Value(nil, 1))
			Expect(ok).To(BeTrue())
			Expect(v).To(Equal("a"))
		})

		It("Should return false for a key with no value", func() {
			_, ok := MustSucceed2(l.Value(nil, 9))
			Expect(ok).To(BeFalse())
		})

		It("Should return the staged value of a key only to its scope", func() {
			l.Stage(txn, 1, "c")
			staged, ok := MustSucceed2(l.Value(txn, 1))
			Expect(ok).To(BeTrue())
			Expect(staged).To(Equal("c"))
			other, ok := MustSucceed2(l.Value(&scope{}, 1))
			Expect(ok).To(BeTrue())
			Expect(other).To(Equal("a"))
		})

		It("Should return false for a key that its scope deletes", func() {
			l.StageDelete(txn, 1)
			_, ok := MustSucceed2(l.Value(txn, 1))
			Expect(ok).To(BeFalse())
		})

		It("Should return the error of a failed load", func() {
			failed := index.NewLookup[*scope, int, string]()
			_, finish := failed.Populate()
			finish(errLoad)
			Expect(failed.Value(nil, 1)).Error().To(MatchError(errLoad))
		})
	})

	Describe("Holder", func() {
		It("Should return a committed holder of a value", func() {
			k, ok := MustSucceed2(l.Holder(nil, "a", nil))
			Expect(ok).To(BeTrue())
			Expect(k).To(Equal(1))
		})

		It("Should never return the skipped key", func() {
			_, ok := MustSucceed2(l.Holder(nil, "a", new(1)))
			Expect(ok).To(BeFalse())
		})

		It("Should return a key that its scope stages with the value", func() {
			l.Stage(txn, 3, "c")
			k, ok := MustSucceed2(l.Holder(txn, "c", nil))
			Expect(ok).To(BeTrue())
			Expect(k).To(Equal(3))
			_, ok = MustSucceed2(l.Holder(&scope{}, "c", nil))
			Expect(ok).To(BeFalse())
		})

		It("Should leave out a committed holder that its scope moves away", func() {
			l.Stage(txn, 1, "c")
			_, ok := MustSucceed2(l.Holder(txn, "a", nil))
			Expect(ok).To(BeFalse())
		})

		It("Should leave out a committed holder that its scope deletes", func() {
			l.StageDelete(txn, 2)
			_, ok := MustSucceed2(l.Holder(txn, "b", nil))
			Expect(ok).To(BeFalse())
		})

		It("Should return the error of a failed load", func() {
			failed := index.NewLookup[*scope, int, string]()
			_, finish := failed.Populate()
			finish(errLoad)
			Expect(failed.Holder(nil, "a", nil)).Error().To(MatchError(errLoad))
		})
	})
})
