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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp/internal/index"
	. "github.com/synnaxlabs/x/testutil"
)

// keys returns the keys that s matches for term under scope, closest match first.
func keys(s *index.Search[*scope, int], scope *scope, term string) []int {
	GinkgoHelper()
	return MustSucceed(s.Get(scope, term)).Keys
}

var _ = Describe("Search", func() {
	var s *index.Search[*scope, int]
	BeforeEach(func() {
		s = index.NewSearch[*scope, int]()
		insert, finish := s.Populate()
		insert(1, []string{"pressure"})
		insert(2, []string{"pressure feed"})
		insert(3, []string{"valve"})
		finish(nil)
	})

	Describe("Bulk load", func() {
		It("Should serve the loaded text, closest match first", func() {
			Expect(keys(s, nil, "pressure")).To(Equal([]int{1, 2}))
			Expect(keys(s, nil, "valve")).To(Equal([]int{3}))
			Expect(keys(s, nil, "missing")).To(BeEmpty())
		})

		It("Should block reads until the load finishes", func() {
			loading := index.NewSearch[*scope, int]()
			insert, finish := loading.Populate()
			insert(1, []string{"valve"})
			read := make(chan []int)
			go func() { read <- keys(loading, nil, "valve") }()
			Consistently(read).ShouldNot(Receive())
			finish(nil)
			Eventually(read).Should(Receive(Equal([]int{1})))
		})

		It("Should return ctx.Err from Wait when ctx ends first", func() {
			loading := index.NewSearch[*scope, int]()
			_, finish := loading.Populate()
			defer finish(nil)
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			Expect(loading.Wait(ctx)).To(MatchError(context.Canceled))
		})

		It("Should return the load error from every read", func(ctx SpecContext) {
			failed := index.NewSearch[*scope, int]()
			insert, finish := failed.Populate()
			insert(1, []string{"valve"})
			finish(errLoad)
			Expect(failed.Wait(ctx)).To(MatchError(errLoad))
			Expect(failed.Get(nil, "valve")).Error().To(MatchError(errLoad))
		})
	})

	Describe("Committed writes", func() {
		It("Should replace the text of a key", func() {
			s.Set(3, []string{"pressure valve"})
			Expect(keys(s, nil, "valve")).To(Equal([]int{3}))
			Expect(keys(s, nil, "pressure")).To(Equal([]int{1, 2, 3}))
		})

		It("Should remove a deleted key", func() {
			s.Delete(1)
			s.Delete(99)
			Expect(keys(s, nil, "pressure")).To(Equal([]int{2}))
		})
	})

	Describe("Staged writes", func() {
		var txn *scope
		BeforeEach(func() { txn = &scope{} })

		It("Should show staged changes only to their scope", func() {
			s.Stage(txn, 4, []string{"pressure"})
			s.StageDelete(txn, 1)
			Expect(keys(s, txn, "pressure")).To(Equal([]int{4, 2}))
			Expect(keys(s, &scope{}, "pressure")).To(Equal([]int{1, 2}))
			Expect(keys(s, nil, "pressure")).To(Equal([]int{1, 2}))
		})

		It("Should hide the committed text of a key the scope changed", func() {
			s.Stage(txn, 1, []string{"valve"})
			Expect(keys(s, txn, "pressure")).To(Equal([]int{2}))
			Expect(keys(s, txn, "valve")).To(ConsistOf(1, 3))
		})

		It("Should match any word when only changed keys matched every word", func() {
			s.Stage(txn, 2, []string{"valve"})
			Expect(keys(s, nil, "pressure feed")).To(Equal([]int{2}))
			Expect(keys(s, txn, "pressure feed")).To(Equal([]int{1}))
		})

		It("Should rank staged and committed keys together", func() {
			s.Stage(txn, 4, []string{"pressure inlet feed"})
			s.Stage(txn, 5, []string{"pressures"})
			Expect(keys(s, txn, "pressure")).To(Equal([]int{1, 2, 4, 5}))
		})

		It("Should keep the last staged change of a key", func() {
			s.Stage(txn, 4, []string{"valve"})
			s.StageDelete(txn, 4)
			Expect(keys(s, txn, "valve")).To(Equal([]int{3}))
			s.Stage(txn, 4, []string{"valve"})
			Expect(keys(s, txn, "valve")).To(ConsistOf(3, 4))
		})

		It("Should stage a key with no text as matching nothing", func() {
			s.Stage(txn, 1, nil)
			Expect(keys(s, txn, "pressure")).To(Equal([]int{2}))
			txn.end(true)
			Expect(keys(s, nil, "pressure")).To(Equal([]int{2}))
		})

		It("Should apply staged changes when the scope commits", func() {
			s.Stage(txn, 4, []string{"valve"})
			s.StageDelete(txn, 3)
			txn.end(true)
			Expect(keys(s, nil, "valve")).To(Equal([]int{4}))
		})

		It("Should drop staged changes when the scope does not commit", func() {
			s.Stage(txn, 4, []string{"valve"})
			s.StageDelete(txn, 3)
			txn.end(false)
			Expect(keys(s, nil, "valve")).To(Equal([]int{3}))
			Expect(keys(s, txn, "valve")).To(Equal([]int{3}))
		})

		It("Should apply changes against the zero scope at once", func() {
			s.Stage(nil, 4, []string{"valve"})
			s.StageDelete(nil, 3)
			Expect(keys(s, nil, "valve")).To(Equal([]int{4}))
		})
	})
})
