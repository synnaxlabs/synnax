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
	"context"
	"maps"
	"math/rand/v2"
	"slices"
	"strconv"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// oddKeys start the keys of holders, to stress the text of their resource IDs.
var oddKeys = []string{
	":", "::", "12:schematic:7", "ключ✓\x00\n", "9:9:9:", " ", "a\nb", "😀", "-1", "+3:",
}

// link is one reference of a holder to a target, as resource ID text.
type link struct{ reference, target string }

func channelText(k channel.Key) string {
	return resource.ID{Type: resource.TypeChannel, Key: gorp.FormatKey(k)}.String()
}

// linksOf returns the links of h. A holder with no key has none.
func linksOf(h refHolder) set.Set[link] {
	links := make(set.Set[link])
	if h.Key == "" {
		return links
	}
	for _, k := range h.Channels {
		links.Add(link{reference: "channels", target: channelText(k)})
	}
	for _, l := range h.Labels {
		id := resource.ID{Type: resource.TypeLabel, Key: gorp.FormatKey(l)}
		links.Add(link{reference: "labels", target: id.String()})
	}
	if h.Output != 0 {
		links.Add(link{reference: "output", target: channelText(h.Output)})
	}
	return links
}

// modelOf returns, for each target that a link of a touched holder starts or stops
// pointing at, the holders whose links to it changed, sorted.
func modelOf(
	before, after map[string]refHolder,
	touched []string,
) map[string][]string {
	out := make(map[string][]string)
	add := func(target, holder string) {
		if !slices.Contains(out[target], holder) {
			out[target] = append(out[target], holder)
		}
	}
	for _, key := range touched {
		holder := holderOf(gorp.Target{Table: "signals_test_holder", Key: key}).String()
		old, next := linksOf(before[key]), linksOf(after[key])
		for l := range old {
			if !next.Contains(l) {
				add(l.target, holder)
			}
		}
		for l := range next {
			if !old.Contains(l) {
				add(l.target, holder)
			}
		}
	}
	for _, holders := range out {
		slices.Sort(holders)
	}
	return out
}

var _ = Describe("Reference changes over random commits", Ordered, func() {
	var (
		holders  *gorp.Table[string, refHolder]
		channels []channel.Key
		labels   []uuid.UUID
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		holders = MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[string, refHolder]{
			DB:         db,
			Name:       "signals_test_holder",
			References: refHolderReferences(),
		}))
		DeferCleanup(func() { Expect(holders.Close()).To(Succeed()) })
		for range 5 {
			ch := channel.Channel{
				Name:     nextName(),
				DataType: telem.Float32T,
				Virtual:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Create(ctx, &ch)
			})).To(Succeed())
			channels = append(channels, ch.Key())
		}
		for range 3 {
			l := label.Label{Key: uuid.New(), Name: nextName(), Color: labelColor}
			Expect(gorp.NewCreate[uuid.UUID, label.Label]().Entry(&l).Exec(ctx, db)).
				To(Succeed())
			labels = append(labels, l.Key)
		}
	})

	randomHolder := func(r *rand.Rand, key string) refHolder {
		h := refHolder{Key: key}
		for _, i := range r.Perm(len(channels))[:r.IntN(4)] {
			h.Channels = append(h.Channels, channels[i])
		}
		for _, i := range r.Perm(len(labels))[:r.IntN(3)] {
			h.Labels = append(h.Labels, labels[i])
		}
		if r.IntN(2) == 0 {
			h.Output = channels[r.IntN(len(channels))]
		}
		return h
	}

	It("Should deliver and stream the targets that each commit changes", func(
		ctx SpecContext,
	) {
		sigs := MustSucceed(signals.New(signals.Config{
			DB:      db,
			Channel: channelSvc,
			Framer:  nodeFramer,
		}))
		Expect(sigs.PublishReferenceChanges(ctx, holderOf)).To(Succeed())
		MustOpen(sigs.Start(ctx))
		hook := make(chan []signals.ReferenceChange, 1)
		DeferCleanup(sigs.ReferenceChanges().OnChange(
			func(_ context.Context, changes []signals.ReferenceChange) {
				select {
				case hook <- changes:
				default:
				}
			},
		))
		responses := openStreamer(ctx, sigs.ReferenceChanges().Key)
		state := make(map[string]refHolder)
		created := 0
		for seed := range uint64(4) {
			r := rand.New(rand.NewPCG(seed, 7))
			for step := range 30 {
				n := 1 + r.IntN(4)
				if step%9 == 4 {
					n = 300 + r.IntN(200)
				}
				before := maps.Clone(state)
				existing := slices.Sorted(maps.Keys(state))
				r.Shuffle(len(existing), func(i, j int) {
					existing[i], existing[j] = existing[j], existing[i]
				})
				var (
					writes  []refHolder
					deletes []string
					touched []string
				)
				for i := range n {
					var key string
					if i < len(existing) && r.IntN(3) > 0 {
						key = existing[i]
					} else {
						created++
						key = oddKeys[r.IntN(len(oddKeys))] + strconv.Itoa(created)
					}
					touched = append(touched, key)
					if _, ok := state[key]; ok && r.IntN(3) == 0 {
						deletes = append(deletes, key)
						delete(state, key)
						continue
					}
					h := randomHolder(r, key)
					writes = append(writes, h)
					state[key] = h
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					err := holders.NewCreate().Entries(&writes).Exec(ctx, tx)
					if err != nil || len(deletes) == 0 {
						return err
					}
					return holders.NewDelete().
						Where(gorp.MatchKeys[string, refHolder](deletes...)).
						Exec(ctx, tx)
				})).To(Succeed())
				want := modelOf(before, state, touched)
				if len(want) == 0 {
					continue
				}
				var changes []signals.ReferenceChange
				Eventually(hook).
					Should(Receive(&changes), "seed %d step %d", seed, step)
				got := make(map[string][]string, len(changes))
				targets := make([]string, len(changes))
				for i, c := range changes {
					targets[i] = c.Target.String()
					for _, h := range c.Holders {
						got[targets[i]] = append(got[targets[i]], h.String())
					}
					slices.Sort(got[targets[i]])
				}
				Expect(got).To(Equal(want), "seed %d step %d", seed, step)
				streamed := make([]string, 0, len(targets))
				for _, s := range receiveSamples(responses) {
					streamed = append(streamed, string(s))
				}
				Expect(streamed).To(Equal(targets), "seed %d step %d", seed, step)
			}
		}
	})
})
