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
)

// edit is a generated mutation that can be replayed against any replica.
type edit func(*crdt.Text)

// randomEdit applies a random insert or delete to d and returns an edit that replays the
// generated operations against another replica.
func randomEdit(rng *rand.Rand, d *crdt.Text) edit {
	const alphabet = "abcdefghijklmnopqrstuvwxyz"
	n := d.Len()
	if n > 0 && rng.Intn(3) == 0 {
		index := rng.Intn(n)
		ops := d.Delete(index, 1+rng.Intn(n-index))
		return func(t *crdt.Text) { t.ApplyDelete(ops...) }
	}
	index := rng.Intn(n + 1)
	runes := make([]rune, 1+rng.Intn(4))
	for i := range runes {
		runes[i] = rune(alphabet[rng.Intn(len(alphabet))])
	}
	ops := d.Insert(index, string(runes))
	return func(t *crdt.Text) { t.ApplyInsert(ops...) }
}

var _ = Describe("Convergence property", func() {
	DescribeTable("random interleavings converge across replicas and delivery orders",
		func(seed int64) {
			rng := rand.New(rand.NewSource(seed))
			const replicas, rounds = 4, 60
			docs := make([]*crdt.Text, replicas)
			for i := range docs {
				docs[i] = crdt.New(uint32(i + 1))
			}
			var allEdits []edit
			for range rounds {
				roundEdits := make([]edit, replicas)
				for i, d := range docs {
					roundEdits[i] = randomEdit(rng, d)
				}
				allEdits = append(allEdits, roundEdits...)
				for origin, e := range roundEdits {
					for j, d := range docs {
						if j != origin {
							e(d)
						}
					}
				}
			}
			want := docs[0].String()
			for _, d := range docs[1:] {
				Expect(d.String()).To(Equal(want))
			}

			shuffled := make([]edit, len(allEdits))
			copy(shuffled, allEdits)
			rng.Shuffle(len(shuffled), func(i, j int) {
				shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
			})
			fresh := crdt.New(999)
			for _, e := range shuffled {
				e(fresh)
			}
			Expect(fresh.String()).To(Equal(want))
		},
		Entry("seed 1", int64(1)),
		Entry("seed 2", int64(2)),
		Entry("seed 7", int64(7)),
		Entry("seed 42", int64(42)),
		Entry("seed 99", int64(99)),
		Entry("seed 1000", int64(1000)),
		Entry("seed 31337", int64(31337)),
		Entry("seed 808080", int64(808080)),
	)
})
