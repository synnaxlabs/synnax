// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation_test

import (
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	disttestutil "github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("SetReads checks", func() {
	// The graph's start pass calls SetReads for every record, so a start must not
	// write or broadcast a record whose reads did not change.
	It("Should write and broadcast nothing when the record holds the reads", func(
		ctx SpecContext,
	) {
		node := disttestutil.OpenNode(ctx)
		DeferCleanup(func() { Expect(node.Close()).To(Succeed()) })
		statuses := MustSucceed(
			status.OpenService(ctx, status.ServiceConfig{DB: node.DB}),
		)
		DeferCleanup(func() { Expect(statuses.Close()).To(Succeed()) })
		channels := MustSucceed(channel.OpenService(ctx, channel.ServiceConfig{
			Channel: node.Channel,
			TS:      node.Storage.TS,
			DB:      node.DB,
			HostKey: node.Identity.HostKey,
			Status:  statuses,
		}))
		DeferCleanup(func() { Expect(channels.Close()).To(Succeed()) })
		svc := MustSucceed(calculation.OpenService(ctx, calculation.ServiceConfig{
			DB:      node.DB,
			Channel: channels,
			Framer:  node.Framer,
			Status:  statuses,
		}))
		DeferCleanup(func() { Expect(svc.Close()).To(Succeed()) })
		virtual := func(name string) channel.Channel {
			ch := channel.Channel{Name: name, DataType: telem.Float32T, Virtual: true}
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return channels.NewWriter(tx).Create(ctx, &ch)
			})).To(Succeed())
			return ch
		}
		a, b, out := virtual("a"), virtual("b"), virtual("sum")
		c := calculation.Calculation{Output: out.Key(), Expression: "return a + b"}
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &c)
		})).To(Succeed())
		var stored calculation.Calculation
		Expect(svc.NewRetrieve().
			Where(calculation.MatchKeys(c.Key)).
			Entry(&stored).
			Exec(ctx, node.DB)).To(Succeed())
		Expect(stored.Reads).To(ConsistOf(a.Key(), b.Key()))
		rec := &Recorder[calculation.Key, calculation.Action]{}
		DeferCleanup(svc.Actions().OnChange(rec.Record))
		readsSets := func() [][]channel.Key {
			var sets [][]channel.Key
			for _, s := range rec.Snapshot() {
				for _, act := range s.Actions {
					if act.Set == nil ||
						!slices.Equal(act.Set.Path, []string{"reads"}) {
						continue
					}
					keys, ok := act.Set.Value.([]channel.Key)
					Expect(ok).To(BeTrue(), "a set of reads holds channel keys")
					sets = append(sets, keys)
				}
			}
			return sets
		}
		// The same reads, in another order and with a repeat, as a compile finds them.
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).SetReads(
				ctx,
				stored,
				[]channel.Key{b.Key(), a.Key(), b.Key()},
			)
		})).To(Succeed())
		// A change after it marks the point where every earlier broadcast arrived.
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).SetReads(ctx, stored, []channel.Key{a.Key()})
		})).To(Succeed())
		Eventually(readsSets).Should(ContainElement([]channel.Key{a.Key()}))
		Expect(readsSets()).To(Equal([][]channel.Key{{a.Key()}}))
	})
})
