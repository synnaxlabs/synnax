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
)

var _ = Describe("Multi keys staged with no value, checker cases", func() {
	var (
		m   *index.Multi[*scope, int, string]
		txn *scope
	)
	BeforeEach(func() {
		m = index.NewMulti[*scope, int, string]()
		insert, finish := m.Populate()
		insert(1, []string{"a", "b"})
		insert(2, []string{"b"})
		finish(nil)
		txn = &scope{}
	})

	It("Should hold none for a key that the scope deleted and then staged", func() {
		m.StageDelete(txn, 5)
		m.Stage(txn, 5, nil)
		Expect(m.Holding(txn, false)).To(ConsistOf(5))
		txn.end(true)
		Expect(m.Holding(nil, false)).To(ConsistOf(5))
		Expect(m.Holding(nil, true)).To(ConsistOf(1, 2))
	})

	It("Should hide the key from every other scope until the commit", func() {
		other := &scope{}
		m.Stage(txn, 5, nil)
		Expect(m.Current(other, 5, nil)).To(BeFalse())
		Expect(m.Current(nil, 5, nil)).To(BeFalse())
		Expect(m.Holding(other, false)).To(BeEmpty())
		txn.committing = true
		Expect(m.CommittingHolding(true)).To(BeEmpty())
		Expect(m.CommittingHolding(false)).To(ConsistOf(5))
		Expect(m.Holding(nil, false)).To(BeEmpty())
		txn.end(true)
		Expect(m.Current(other, 5, nil)).To(BeTrue())
		Expect(m.CommittingHolding(false)).To(BeEmpty())
	})

	It("Should keep each scope's keys apart across commits and failures", func() {
		m.Stage(txn, 5, nil)
		txn.end(true)
		failed := &scope{}
		m.Stage(failed, 6, nil)
		Expect(m.Holding(failed, false)).To(ConsistOf(5, 6))
		failed.end(false)
		next := &scope{}
		m.Stage(next, 7, nil)
		Expect(m.Holding(next, false)).To(ConsistOf(5, 7))
		Expect(m.Current(next, 6, nil)).To(BeFalse())
		next.end(true)
		Expect(m.Holding(nil, false)).To(ConsistOf(5, 7))
	})

	It("Should hold none after no value, a value, and no value again", func() {
		m.Stage(txn, 5, nil)
		m.Stage(txn, 5, []string{"e"})
		m.Stage(txn, 5, nil)
		Expect(m.Holding(txn, false)).To(ConsistOf(5))
		Expect(m.Get(txn, "e")).To(BeEmpty())
		txn.end(true)
		Expect(m.Holding(nil, false)).To(ConsistOf(5))
		Expect(m.Get(nil, "e")).To(BeEmpty())
	})

	It("Should clear the values of a held key that the scope stages with none", func() {
		m.Stage(txn, 1, nil)
		Expect(m.Holding(txn, false)).To(ConsistOf(1))
		Expect(m.Get(txn, "a")).To(BeEmpty())
		txn.end(true)
		Expect(m.Holding(nil, false)).To(ConsistOf(1))
		Expect(m.Get(nil, "a")).To(BeEmpty())
		Expect(m.Get(nil, "b")).To(ConsistOf(2))
	})

	It("Should keep every key of a scope that stages more than fresh retains", func() {
		for k := 100; k < 2100; k++ {
			m.Stage(txn, k, nil)
		}
		txn.end(true)
		next := &scope{}
		m.Stage(next, 5000, nil)
		Expect(m.Holding(next, false)).To(HaveLen(2001))
		next.end(true)
		Expect(m.Holding(nil, false)).To(HaveLen(2001))
	})
})
