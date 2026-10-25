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
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
)

// engine is the method set that Lookup and Sorted share.
type engine interface {
	Populate() (insert func(int, string), finish func(error))
	Wait(ctx context.Context) error
	Set(key int, value string)
	Delete(key int)
	Stage(s *scope, key int, value string)
	StageDelete(s *scope, key int)
	Get(s *scope, values ...string) ([]int, error)
}

// populated returns an index loaded with key 1 -> a, 2 -> a, and 3 -> b.
func populated(newEngine func() engine) engine {
	e := newEngine()
	insert, finish := e.Populate()
	insert(1, "a")
	insert(2, "a")
	insert(3, "b")
	finish(nil)
	return e
}

var errLoad = errors.New("load failed")

var _ = Describe("Index", func() {
	engines := map[string]func() engine{
		"Lookup": func() engine { return index.NewLookup[*scope, int, string]() },
		"Sorted": func() engine { return index.NewSorted[*scope, int, string]() },
	}
	for name, newEngine := range engines {
		Describe(name, func() {
			Describe("Bulk load", func() {
				It("Should serve the loaded mappings", func() {
					e := populated(newEngine)
					Expect(e.Get(nil, "a")).To(ConsistOf(1, 2))
					Expect(e.Get(nil, "a", "b")).To(ConsistOf(1, 2, 3))
					Expect(e.Get(nil, "missing")).To(BeEmpty())
				})

				It("Should block reads until the load finishes", func() {
					e := newEngine()
					insert, finish := e.Populate()
					insert(1, "a")
					read := make(chan []int)
					go func() { read <- MustSucceed(e.Get(nil, "a")) }()
					Consistently(read).ShouldNot(Receive())
					finish(nil)
					Eventually(read).Should(Receive(ConsistOf(1)))
				})

				It("Should return ctx.Err from Wait when ctx ends first", func() {
					e := newEngine()
					_, finish := e.Populate()
					defer finish(nil)
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					Expect(e.Wait(ctx)).To(MatchError(context.Canceled))
				})

				It(
					"Should return the load error from every read",
					func(ctx SpecContext) {
						e := newEngine()
						insert, finish := e.Populate()
						insert(1, "a")
						finish(errLoad)
						Expect(e.Wait(ctx)).To(MatchError(errLoad))
						Expect(e.Get(nil, "a")).Error().To(MatchError(errLoad))
					},
				)

				It("Should return an empty slice for no values", func() {
					e := newEngine()
					keys := MustSucceed(e.Get(nil))
					Expect(keys).To(BeEmpty())
					Expect(keys).ToNot(BeNil())
				})
			})

			Describe("Committed writes", func() {
				It("Should move a key to its new value", func() {
					e := populated(newEngine)
					e.Set(1, "b")
					Expect(e.Get(nil, "a")).To(ConsistOf(2))
					Expect(e.Get(nil, "b")).To(ConsistOf(1, 3))
				})

				It("Should ignore a write of the current value", func() {
					e := populated(newEngine)
					e.Set(1, "a")
					Expect(e.Get(nil, "a")).To(ConsistOf(1, 2))
				})

				It("Should remove a deleted key", func() {
					e := populated(newEngine)
					e.Delete(3)
					e.Delete(99)
					Expect(e.Get(nil, "b")).To(BeEmpty())
				})
			})

			Describe("Staged writes", func() {
				var (
					e   engine
					txn *scope
				)
				BeforeEach(func() {
					e = populated(newEngine)
					txn = &scope{}
				})

				It("Should show staged changes only to their scope", func() {
					e.Stage(txn, 4, "a")
					e.StageDelete(txn, 1)
					Expect(e.Get(txn, "a")).To(ConsistOf(2, 4))
					Expect(e.Get(&scope{}, "a")).To(ConsistOf(1, 2))
					Expect(e.Get(nil, "a")).To(ConsistOf(1, 2))
				})

				It("Should move a key to its staged value within the scope", func() {
					e.Stage(txn, 1, "b")
					Expect(e.Get(txn, "a")).To(ConsistOf(2))
					Expect(e.Get(txn, "b")).To(ConsistOf(1, 3))
				})

				It("Should keep the last staged change of a key", func() {
					e.Stage(txn, 4, "a")
					e.StageDelete(txn, 4)
					Expect(e.Get(txn, "a")).To(ConsistOf(1, 2))
					e.Stage(txn, 4, "b")
					Expect(e.Get(txn, "b")).To(ConsistOf(3, 4))
				})

				It("Should apply staged changes when the scope commits", func() {
					e.Stage(txn, 4, "a")
					e.StageDelete(txn, 3)
					txn.end(true)
					Expect(e.Get(nil, "a")).To(ConsistOf(1, 2, 4))
					Expect(e.Get(nil, "b")).To(BeEmpty())
				})

				It("Should drop staged changes when the scope does not commit", func() {
					e.Stage(txn, 4, "a")
					e.StageDelete(txn, 3)
					txn.end(false)
					Expect(e.Get(nil, "a")).To(ConsistOf(1, 2))
					Expect(e.Get(nil, "b")).To(ConsistOf(3))
					Expect(e.Get(txn, "a")).To(ConsistOf(1, 2))
				})

				It("Should apply changes against the zero scope at once", func() {
					e.Stage(nil, 4, "a")
					e.StageDelete(nil, 3)
					Expect(e.Get(nil, "a")).To(ConsistOf(1, 2, 4))
					Expect(e.Get(nil, "b")).To(BeEmpty())
				})
			})
		})
	}

	Describe("Sorted walk", func() {
		var s *index.Sorted[*scope, int, int]
		BeforeEach(func() {
			s = index.NewSorted[*scope, int, int]()
			insert, finish := s.Populate()
			for _, k := range []int{5, 3, 1, 4, 2} {
				insert(k, k*10)
			}
			finish(nil)
		})

		It("Should walk in ascending value order", func() {
			Expect(s.Walk(false, nil, 0)).To(Equal([]int{1, 2, 3, 4, 5}))
		})

		It("Should walk in descending value order", func() {
			Expect(s.Walk(true, nil, 0)).To(Equal([]int{5, 4, 3, 2, 1}))
		})

		It("Should stop at the limit", func() {
			Expect(s.Walk(false, nil, 2)).To(Equal([]int{1, 2}))
			Expect(s.Walk(true, nil, 2)).To(Equal([]int{5, 4}))
		})

		It("Should return nil for a negative limit", func() {
			Expect(s.Walk(false, nil, -1)).To(BeNil())
		})

		It("Should resume strictly past the cursor", func() {
			after := 20
			Expect(s.Walk(false, &after, 2)).To(Equal([]int{3, 4}))
			Expect(s.Walk(true, &after, 0)).To(Equal([]int{1}))
		})

		It("Should keep value order through committed writes", func() {
			s.Set(1, 35)
			s.Set(6, 0)
			s.Delete(4)
			Expect(s.Walk(false, nil, 0)).To(Equal([]int{6, 2, 3, 1, 5}))
		})

		It("Should not see staged changes", func() {
			txn := &scope{}
			s.Stage(txn, 6, 0)
			s.StageDelete(txn, 1)
			Expect(s.Walk(false, nil, 0)).To(Equal([]int{1, 2, 3, 4, 5}))
			txn.end(true)
			Expect(s.Walk(false, nil, 0)).To(Equal([]int{6, 2, 3, 4, 5}))
		})

		It("Should return nil when the load failed", func() {
			failed := index.NewSorted[*scope, int, int]()
			insert, finish := failed.Populate()
			insert(2, 20)
			insert(1, 10)
			finish(errLoad)
			Expect(failed.Walk(false, nil, 0)).To(BeNil())
		})

		It("Should return nil for an empty index", func() {
			empty := index.NewSorted[*scope, int, int]()
			_, finish := empty.Populate()
			finish(nil)
			Expect(empty.Walk(false, nil, 0)).To(BeNil())
		})
	})
})
