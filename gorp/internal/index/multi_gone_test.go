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

var _ = Describe("Multi gone keys", func() {
	var (
		m   *index.Multi[*scope, int, string]
		txn *scope
	)
	BeforeEach(func() {
		m = index.NewMulti[*scope, int, string]()
		insert, finish := m.Populate()
		insert(1, []string{"a", "b"})
		insert(7, nil)
		insert(8, nil)
		finish(nil)
		txn = &scope{}
	})

	It("deleted then staged with no value then deleted again", func() {
		m.StageDelete(txn, 7)
		m.Stage(txn, 7, nil)
		m.StageDelete(txn, 7)
		Expect(m.Holding(txn, false)).To(ConsistOf(8))
		Expect(m.Current(txn, 7, nil)).To(BeFalse())
		Expect(m.Holding(nil, false)).To(ConsistOf(7, 8))
		txn.end(true)
		Expect(m.Holding(nil, false)).To(ConsistOf(8))
	})

	It("deleted then staged with values then deleted again", func() {
		m.StageDelete(txn, 7)
		m.Stage(txn, 7, []string{"g"})
		m.StageDelete(txn, 7)
		Expect(m.Holding(txn, false)).To(ConsistOf(8))
		Expect(m.Holding(txn, true)).To(ConsistOf(1))
		Expect(m.Get(txn, "g")).To(BeEmpty())
		txn.end(true)
		Expect(m.Holding(nil, false)).To(ConsistOf(8))
		Expect(m.Holding(nil, true)).To(ConsistOf(1))
	})

	It("staged with values then deleted", func() {
		m.Stage(txn, 7, []string{"g"})
		m.StageDelete(txn, 7)
		Expect(m.Holding(txn, false)).To(ConsistOf(8))
		Expect(m.Current(txn, 7, nil)).To(BeFalse())
		txn.end(true)
		Expect(m.Holding(nil, false)).To(ConsistOf(8))
		Expect(m.Get(nil, "g")).To(BeEmpty())
	})

	It("staged with no value then deleted", func() {
		m.Stage(txn, 7, nil)
		m.StageDelete(txn, 7)
		Expect(m.Holding(txn, false)).To(ConsistOf(8))
		txn.end(true)
		Expect(m.Holding(nil, false)).To(ConsistOf(8))
	})

	It("fresh key deleted, aborted and committed", func() {
		m.Stage(txn, 9, nil)
		m.StageDelete(txn, 9)
		Expect(m.Holding(txn, false)).To(ConsistOf(7, 8))
		txn.end(true)
		Expect(m.Holding(nil, false)).To(ConsistOf(7, 8))
	})

	It("two scopes in sequence", func() {
		m.StageDelete(txn, 7)
		txn.end(false)
		b := &scope{}
		m.StageDelete(b, 8)
		Expect(m.Holding(b, false)).To(ConsistOf(7))
		Expect(m.Holding(txn, false)).To(ConsistOf(7, 8))
		Expect(m.Current(b, 7, nil)).To(BeTrue())
		Expect(m.Current(b, 8, nil)).To(BeFalse())
		b.end(true)
		Expect(m.Holding(nil, false)).To(ConsistOf(7))
		c := &scope{}
		m.StageDelete(c, 7)
		Expect(m.Holding(b, false)).To(ConsistOf(7))
		c.end(true)
		Expect(m.Holding(nil, false)).To(BeEmpty())
	})

	It("reads during commit", func() {
		m.StageDelete(txn, 7)
		txn.committing = true
		Expect(m.Holding(nil, false)).To(ConsistOf(7, 8))
		Expect(m.Holding(txn, false)).To(ConsistOf(8))
		Expect(m.CommittingHolding(true)).To(BeEmpty())
		Expect(m.CommittingHolding(false)).NotTo(ContainElement(7))
		Expect(m.Current(txn, 7, nil)).To(BeFalse())
		Expect(m.Current(&scope{}, 7, nil)).To(BeTrue())
		Expect(MustBeOk(m.Held(txn, 7))).To(BeEmpty())
		txn.end(true)
		Expect(m.Holding(nil, false)).To(ConsistOf(8))
		Expect(m.Current(nil, 7, nil)).To(BeFalse())
	})

	It("concurrent reader while a key in none is deleted", func() {
		done := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			defer close(done)
			for range 200 {
				Expect(m.Holding(&scope{}, false)).To(ContainElements(7, 8))
				m.Current(&scope{}, 7, nil)
			}
		}()
		for range 200 {
			m.StageDelete(txn, 7)
			m.Stage(txn, 7, nil)
			m.StageDelete(txn, 8)
		}
		<-done
		txn.end(false)
		Expect(m.Holding(nil, false)).To(ConsistOf(7, 8))
	})

	It("Should panic when a second scope stages while the first is open", func() {
		m.StageDelete(txn, 7)
		Expect(func() { m.StageDelete(&scope{}, 8) }).
			To(PanicWith(ContainSubstring("while another was open")))
	})
})
