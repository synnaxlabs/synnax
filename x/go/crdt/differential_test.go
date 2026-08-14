// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package crdt_test

import (
	"math/rand"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/crdt"
	"github.com/synnaxlabs/x/set"
)

// pair drives the production document and the frozen per-character reference model
// through identical edits, asserting that both generate identical operations.
type pair struct {
	p *crdt.Text
	r *refText
}

func newPair(replica uint32) *pair {
	return &pair{p: crdt.New(replica), r: newRef(replica)}
}

func (pr *pair) insert(index int, text string) []crdt.Insert {
	GinkgoHelper()
	ops := pr.p.Insert(index, text)
	Expect(pr.r.Insert(index, text)).To(Equal(ops))
	return ops
}

func (pr *pair) delete(index, length int) []crdt.Delete {
	GinkgoHelper()
	ops := pr.p.Delete(index, length)
	Expect(pr.r.Delete(index, length)).To(Equal(ops))
	return ops
}

func (pr *pair) applyInsert(ops ...crdt.Insert) {
	pr.p.ApplyInsert(ops...)
	pr.r.ApplyInsert(ops...)
}

func (pr *pair) applyDelete(ops ...crdt.Delete) {
	pr.p.ApplyDelete(ops...)
	pr.r.ApplyDelete(ops...)
}

// expectEquivalent asserts that the production document is observably identical to the
// reference model: value, length, per-index ids, collectable set, and snapshots that
// round-trip across the two implementations in both directions.
func expectEquivalent(pr *pair) {
	GinkgoHelper()
	Expect(pr.p.String()).To(Equal(pr.r.String()))
	Expect(pr.p.Len()).To(Equal(pr.r.Len()))
	for i := range pr.p.Len() {
		pid, pok := pr.p.IndexToID(i)
		rid, rok := pr.r.IndexToID(i)
		Expect(pok).To(BeTrue())
		Expect(rok).To(BeTrue())
		Expect(pid).To(Equal(rid))
	}
	Expect(set.New(pr.p.Collectable()...)).To(Equal(set.New(pr.r.Collectable()...)))
	inserts, deletes := pr.p.Snapshot()
	intoRef := newRef(99)
	intoRef.Load(inserts, deletes)
	Expect(intoRef.String()).To(Equal(pr.r.String()))
	refInserts, refDeletes := pr.r.Snapshot()
	intoProd := crdt.New(99)
	intoProd.Load(refInserts, refDeletes)
	Expect(intoProd.String()).To(Equal(pr.p.String()))
}

// recorded is an operation batch owned by one replica, replayable against any pair.
type recorded struct {
	owner   int
	inserts []crdt.Insert
	deletes []crdt.Delete
}

func (r recorded) applyTo(pr *pair) {
	if len(r.inserts) > 0 {
		pr.applyInsert(r.inserts...)
	} else {
		pr.applyDelete(r.deletes...)
	}
}

var _ = Describe("Differential against the per-character reference", func() {
	DescribeTable("random schedules match the reference model exactly",
		func(seed int64) {
			rng := rand.New(rand.NewSource(seed))
			const replicas, rounds = 3, 30
			const alphabet = "abcdefghijklmnopqrstuvwxyz"
			pairs := make([]*pair, replicas)
			for i := range pairs {
				pairs[i] = newPair(uint32(i + 1))
			}
			var log []recorded
			for round := range rounds {
				var roundOps []recorded
				for i, pr := range pairs {
					n := pr.p.Len()
					if n > 0 && rng.Intn(3) == 0 {
						index := rng.Intn(n)
						length := 1 + rng.Intn(min(n-index, 5))
						roundOps = append(roundOps,
							recorded{owner: i, deletes: pr.delete(index, length)})
						continue
					}
					runes := make([]rune, 1+rng.Intn(4))
					for j := range runes {
						runes[j] = rune(alphabet[rng.Intn(len(alphabet))])
					}
					roundOps = append(
						roundOps,
						recorded{
							owner:   i,
							inserts: pr.insert(rng.Intn(n+1), string(runes)),
						},
					)
				}
				log = append(log, roundOps...)
				for i, pr := range pairs {
					delivery := rng.Perm(len(roundOps))
					for _, j := range delivery {
						if roundOps[j].owner != i {
							roundOps[j].applyTo(pr)
						}
					}
				}
				if round%10 == 9 {
					for _, pr := range pairs {
						expectEquivalent(pr)
					}
				}
			}
			want := pairs[0].p.String()
			for _, pr := range pairs {
				expectEquivalent(pr)
				Expect(pr.p.String()).To(Equal(want))
			}

			// A fresh pair replays the entire shuffled history, exercising pending-op
			// buffering and tombstone-before-insert arrival in both implementations.
			shuffled := make([]recorded, len(log))
			copy(shuffled, log)
			rng.Shuffle(len(shuffled), func(i, j int) {
				shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
			})
			fresh := newPair(999)
			for _, r := range shuffled {
				r.applyTo(fresh)
			}
			expectEquivalent(fresh)
			Expect(fresh.p.String()).To(Equal(want))
		},
		Entry("seed 3", int64(3)),
		Entry("seed 11", int64(11)),
		Entry("seed 17", int64(17)),
		Entry("seed 23", int64(23)),
		Entry("seed 51", int64(51)),
		Entry("seed 77", int64(77)),
		Entry("seed 123", int64(123)),
		Entry("seed 5150", int64(5150)),
		Entry("seed 90210", int64(90210)),
		Entry("seed 424242", int64(424242)),
	)

	It("Should match on a concurrent insert into the middle of a run", func() {
		a, b := newPair(1), newPair(2)
		seed := a.insert(0, "abcdef")
		b.applyInsert(seed...)
		mid := b.insert(3, "XY")
		a.applyInsert(mid...)
		expectEquivalent(a)
		expectEquivalent(b)
		Expect(a.p.String()).To(Equal(b.p.String()))
	})

	It("Should match when a run's operations are redelivered after a split", func() {
		a, b := newPair(1), newPair(2)
		seed := a.insert(0, "abcdef")
		b.applyInsert(seed...)
		mid := b.insert(3, "XY")
		a.applyInsert(mid...)
		a.applyInsert(seed...)
		b.applyInsert(seed...)
		expectEquivalent(a)
		expectEquivalent(b)
		Expect(a.p.String()).To(Equal(b.p.String()))
	})

	It("Should match when deletes arrive before the run they tombstone", func() {
		a := newPair(1)
		inserts := a.insert(0, "hello world")
		deletes := a.delete(2, 6)
		late := newPair(2)
		late.applyDelete(deletes...)
		late.applyInsert(inserts...)
		expectEquivalent(late)
		Expect(late.p.String()).To(Equal(a.p.String()))
	})

	It("Should match on deletes spanning a split boundary", func() {
		a, b := newPair(1), newPair(2)
		seed := a.insert(0, "abcdef")
		b.applyInsert(seed...)
		mid := b.insert(3, "XY")
		a.applyInsert(mid...)
		wipe := a.delete(1, 6)
		b.applyDelete(wipe...)
		expectEquivalent(a)
		expectEquivalent(b)
		Expect(a.p.String()).To(Equal(b.p.String()))
	})
})
