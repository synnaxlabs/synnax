// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package iterator_test

import (
	"fmt"
	"math/rand/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer/iterator"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Scratch clock seeds", Ordered, func() {
	var (
		node        testutil.Node
		iteratorSvc *iterator.Service
		channelSvc  *channel.Service
	)
	BeforeAll(func(ctx SpecContext) {
		node = testutil.NewNode(ctx)
		iteratorSvc, channelSvc = openServices(ctx, node)
	})
	create := func(ctx SpecContext, ch *channel.Channel) {
		GinkgoHelper()
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, ch)
		})).To(Succeed())
	}
	for seed := range 40 {
		It(fmt.Sprintf("Should stamp the clock of every series read, seed %d", seed),
			func(ctx SpecContext) {
				rng := rand.New(rand.NewPCG(uint64(seed), 11))
				p := fmt.Sprintf("s%d_", seed)
				i1 := &channel.Channel{Name: p + "i1", DataType: telem.TimestampT, IsIndex: true}
				i2 := &channel.Channel{Name: p + "i2", DataType: telem.TimestampT, IsIndex: true}
				create(ctx, i1)
				create(ctx, i2)
				a1 := &channel.Channel{Name: p + "a1", DataType: telem.Float32T, LocalIndex: i1.LocalKey}
				b1 := &channel.Channel{Name: p + "b1", DataType: telem.Float32T, LocalIndex: i1.LocalKey}
				a2 := &channel.Channel{Name: p + "a2", DataType: telem.Float32T, LocalIndex: i2.LocalKey}
				create(ctx, a1)
				create(ctx, b1)
				create(ctx, a2)
				sum := &channel.Channel{
					Name: p + "sum", DataType: telem.Float32T,
					Expression: "return " + a1.Name + " + " + b1.Name,
				}
				create(ctx, sum)
				cross := &channel.Channel{
					Name: p + "cross", DataType: telem.Float32T,
					Expression: "return " + a1.Name + " + " + a2.Name,
				}
				create(ctx, cross)
				nested := &channel.Channel{
					Name: p + "nested", DataType: telem.Float32T,
					Expression: "return " + sum.Name + " * 2",
				}
				create(ctx, nested)
				records := map[channel.Key]channel.Channel{}
				for _, ch := range []*channel.Channel{i1, i2, a1, b1, a2, sum, cross, nested} {
					records[ch.Key()] = *ch
				}
				for _, ch := range []*channel.Channel{sum, cross, nested} {
					var idx channel.Channel
					Expect(channelSvc.NewRetrieve().
						Where(channel.MatchKeys(ch.Index())).
						Entry(&idx).
						Exec(ctx, node.DB)).To(Succeed())
					records[idx.Key()] = idx
				}
				write := func(idx *channel.Channel, data ...*channel.Channel) {
					t := telem.TimeStamp(1+rng.IntN(3)) * telem.SecondTS
					for range 1 + rng.IntN(4) {
						n := 1 + rng.IntN(12)
						stamps := make([]telem.TimeStamp, n)
						for j := range n {
							t += telem.TimeStamp(1+rng.IntN(5)) * telem.MillisecondTS * 100
							stamps[j] = t
						}
						keys := channel.Keys{idx.Key()}
						series := []telem.Series{telem.NewSeries(stamps)}
						for _, d := range data {
							vals := make([]float32, n)
							for j := range vals {
								vals[j] = float32(rng.IntN(100))
							}
							keys = append(keys, d.Key())
							series = append(series, telem.NewSeries(vals))
						}
						w := MustSucceed(node.Framer.OpenWriter(ctx, framer.WriterConfig{
							Start:            stamps[0],
							Keys:             keys,
							EnableAutoCommit: new(true),
						}))
						Expect(w.Write(frame.NewMulti(keys, series))).To(BeTrue())
						Expect(w.Close()).To(Succeed())
						t += telem.TimeStamp(rng.IntN(3)) * telem.SecondTS
					}
				}
				write(i1, a1, b1)
				write(i2, a2)
				all := make(channel.Keys, 0, len(records))
				for k := range records {
					all = append(all, k)
				}
				checked := 0
				for read := range 10 {
					var keys channel.Keys
					for _, k := range all {
						if rng.IntN(3) == 0 {
							keys = append(keys, k)
						}
					}
					if len(keys) == 0 {
						keys = channel.Keys{sum.Key()}
					}
					var red telem.Reduction
					switch rng.IntN(4) {
					case 1:
						red = telem.Reduction{Variant: telem.StrideReduction{
							Factor: uint32(2 + rng.IntN(2)),
						}}
					case 2:
						aggs := []telem.Aggregation{
							telem.AggregationMinMax, telem.AggregationAverage,
							telem.AggregationDecimate,
						}
						red = telem.Reduction{Variant: telem.LimitReduction{
							Aggregation: aggs[rng.IntN(len(aggs))],
							PointLimit:  uint32(3 + rng.IntN(8)),
						}}
					}
					desc := fmt.Sprintf("read %d keys %v reduction %+v", read, keys, red)
					iter, err := iteratorSvc.Open(ctx, iterator.Config{
						Keys:      keys,
						Bounds:    telem.TimeRangeMax,
						Reduction: red,
						ChunkSize: int64(1 + rng.IntN(20)),
					})
					Expect(err).ToNot(HaveOccurred(), desc)
					check := func() {
						for k, s := range iter.Value().Frame.Entries() {
							checked++
							Expect(s.Clock).To(
								Equal(uint32(records[k].Index())),
								fmt.Sprintf("%s: key %d (%s)", desc, k, records[k].Name),
							)
						}
					}
					span := iterator.AutoSpan
					if rng.IntN(2) == 0 {
						span = telem.TimeSpan(1+rng.IntN(5)) * telem.Second
					}
					if rng.IntN(2) == 0 {
						if iter.SeekFirst() {
							for iter.Next(span) {
								check()
							}
						}
					} else if iter.SeekLast() {
						for iter.Prev(span) {
							check()
						}
					}
					Expect(iter.Close()).To(Succeed(), desc)
				}
				Expect(checked).To(BeNumerically(">", 0))
				GinkgoWriter.Printf("seed %d checked %d series\n", seed, checked)
			})
	}
})
