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

var _ = Describe("Multi", func() {
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

	It("Should map each value to every key that holds it", func() {
		Expect(m.Get(nil, "a")).To(ConsistOf(1))
		Expect(m.Get(nil, "b")).To(ConsistOf(1, 2))
		Expect(m.Get(nil, "a", "b")).To(ConsistOf(1, 2))
		Expect(m.Get(nil)).To(BeEmpty())
	})

	It("Should replace all values of a key on Set", func() {
		m.Set(1, []string{"c"})
		Expect(m.Get(nil, "a")).To(BeEmpty())
		Expect(m.Get(nil, "b")).To(ConsistOf(2))
		Expect(m.Get(nil, "c")).To(ConsistOf(1))
	})

	It("Should show staged changes only to their scope", func() {
		m.Stage(txn, 1, []string{"b"})
		m.Stage(txn, 3, []string{"a"})
		m.StageDelete(txn, 2)
		Expect(m.Get(txn, "a")).To(ConsistOf(3))
		Expect(m.Get(txn, "b")).To(ConsistOf(1))
		Expect(m.Get(nil, "a")).To(ConsistOf(1))
	})

	It("Should apply staged changes when the scope commits", func() {
		m.Stage(txn, 1, nil)
		txn.end(true)
		Expect(m.Get(nil, "a")).To(BeEmpty())
		Expect(m.Get(nil, "b")).To(ConsistOf(2))
	})

	It("Should track many holders of one value", func() {
		for k := 10; k < 110; k++ {
			m.Set(k, []string{"c"})
		}
		Expect(m.Get(nil, "c")).To(HaveLen(100))
		for k := 10; k < 109; k++ {
			m.Delete(k)
		}
		Expect(m.Get(nil, "c")).To(ConsistOf(109))
	})

	It("Should drop a staged delete when the scope does not commit", func() {
		m.StageDelete(txn, 1)
		Expect(m.Get(txn, "a")).To(BeEmpty())
		txn.end(false)
		Expect(m.Get(nil, "a")).To(ConsistOf(1))
		Expect(m.Get(txn, "a")).To(ConsistOf(1))
	})

	It("Should return announced keys until they are withdrawn", func() {
		handle := m.Announce(4, []string{"d", "e"})
		Expect(m.Committing("e")).To(ConsistOf(4))
		m.Withdraw(handle)
		Expect(m.Committing("e")).To(BeEmpty())
	})
})
