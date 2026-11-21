// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package signals_test

import (
	"math/rand/v2"
	"slices"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
)

type modelSample struct {
	target  resource.ID
	holders []resource.ID
}

var oddKeys = []string{
	"", ":", "::", "0", "12:schematic:7", "ключ✓\x00\n", "9:9:9:", "\x00", " ",
	strings.Repeat("x", 127), strings.Repeat("y", 128), strings.Repeat("9:", 300),
	"a\nb", "😀", "-1", "+3:", "1e3",
}

func randomKey(r *rand.Rand) string {
	if r.IntN(3) == 0 {
		return oddKeys[r.IntN(len(oddKeys))]
	}
	return strings.Repeat(string(rune('a'+r.IntN(26))), 1+r.IntN(4))
}

func randomCommit(r *rand.Rand, size int) gorp.Commit {
	tables := []string{"channel", "secret", "range", "schematic", "task"}
	links := func() []gorp.Link {
		out := make([]gorp.Link, r.IntN(4))
		for i := range out {
			out[i] = gorp.Link{
				Target: gorp.Target{
					Table: tables[r.IntN(len(tables))],
					Key:   randomKey(r),
				},
				Kind: gorp.LinkKind(1 + r.IntN(2)),
				Tag:  uint8(r.IntN(3)),
			}
		}
		return out
	}
	changes := make(gorp.Changes, size)
	for i := range changes {
		changes[i] = gorp.Change{
			Record:  gorp.Target{Table: tables[r.IntN(len(tables))], Key: randomKey(r)},
			Kind:    gorp.ChangeKind(1 + r.IntN(3)),
			Added:   links(),
			Removed: links(),
		}
	}
	return gorp.Commit{Seq: 1, Changes: changes}
}

func idOf(t gorp.Target) resource.ID {
	return resource.ID{Type: resource.Type(t.Table), Key: t.Key}
}

// modelOf returns, for each target in the order of its first link, the holders whose
// link to it changed, once each.
func modelOf(c gorp.Commit) []modelSample {
	var out []modelSample
	at := map[gorp.Target]int{}
	for _, ch := range c.Changes {
		holder := idOf(ch.Record)
		for _, l := range append(slices.Clone(ch.Added), ch.Removed...) {
			i, ok := at[l.Target]
			if !ok {
				i = len(out)
				at[l.Target] = i
				out = append(out, modelSample{target: idOf(l.Target)})
			}
			if !slices.Contains(out[i].holders, holder) {
				out[i].holders = append(out[i].holders, holder)
			}
		}
	}
	return out
}

var _ = Describe("Reference feed model", func() {
	It(
		"Should match a model over seeded commits, bulk commits between small ones",
		func() {
			for seed := range uint64(64) {
				r := rand.New(rand.NewPCG(seed, 7))
				appendCommit := signals.NewReferenceFeedAppend(idOf)
				for step := range 40 {
					size := r.IntN(5)
					if step%9 == 4 {
						size = 2000 + r.IntN(2000)
					}
					c := randomCommit(r, size)
					want := modelOf(c)
					fr, changes := appendCommit(c)
					var sent, wantSent []string
					for s := range fr.Series() {
						Expect(s.Validate()).To(Succeed())
						for sample := range s.Samples() {
							sent = append(sent, string(sample))
						}
					}
					var got []modelSample
					for _, ch := range changes {
						got = append(got, modelSample{
							target:  ch.Target,
							holders: ch.Holders,
						})
					}
					for _, w := range want {
						wantSent = append(wantSent, w.target.String())
					}
					Expect(got).To(Equal(want), "seed %d step %d", seed, step)
					Expect(sent).To(Equal(wantSent), "seed %d step %d", seed, step)
				}
			}
		},
	)
})
