// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer_test

import (
	"context"
	"fmt"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer", func() {
	Describe("Happy Path", Ordered, func() {
		scenarios := []func(context.Context) scenario{
			gatewayOnlyScenario,
			peerOnlyScenario,
			mixedScenario,
			freeWriterScenario,
		}
		for i, sF := range scenarios {
			_sF := sF
			var s scenario
			BeforeAll(func(ctx SpecContext) { s = _sF(ctx) })
			AfterAll(func() { Expect(s.closer.Close()).To(Succeed()) })
			Specify(fmt.Sprintf("Scenario: %v - Happy Path", i), func(ctx SpecContext) {
				writer := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
					Keys:  s.keys,
					Start: 10 * telem.SecondTS,
					Sync:  new(true),
				}))
				MustSucceed(writer.Write(frame.NewMulti(
					s.keys,
					[]telem.Series{
						telem.NewSeriesV[int64](1, 2, 3),
						telem.NewSeriesV[int64](3, 4, 5),
						telem.NewSeriesV[int64](5, 6, 7),
					},
				)))
				MustSucceed(writer.Commit())
				MustSucceed(writer.Write(frame.NewMulti(
					s.keys,
					[]telem.Series{
						telem.NewSeriesV[int64](1, 2, 3),
						telem.NewSeriesV[int64](3, 4, 5),
						telem.NewSeriesV[int64](5, 6, 7),
					},
				)))
				MustSucceed(writer.Commit())
				Expect(writer.Close()).To(Succeed())
			})
		}
	})

	Describe("Variable Channels", Ordered, func() {
		var (
			s     scenario
			idxCh channel.Channel
			strCh channel.Channel
		)
		BeforeAll(func(ctx SpecContext) {
			builder := mock.ProvisionCluster(ctx, 1)
			dist := builder.Nodes[1]
			idxCh = channel.Channel{
				Name:     channel.NewRandomName(),
				IsIndex:  true,
				DataType: telem.TimeStampT,
			}
			Expect(dist.Channel.Create(ctx, &idxCh)).To(Succeed())
			strCh = channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.StringT,
				LocalIndex: idxCh.LocalKey,
			}
			Expect(dist.Channel.Create(ctx, &strCh)).To(Succeed())
			s = scenario{
				dist:   dist,
				closer: builder,
				name:   "Variable",
				keys:   []channel.Key{idxCh.Key(), strCh.Key()},
			}
		})
		AfterAll(func() { Expect(s.closer.Close()).To(Succeed()) })
		It("Should write and read persisted string data", func(ctx SpecContext) {
			w := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:  s.keys,
				Start: 10 * telem.SecondTS,
				Sync:  new(true),
			}))
			MustSucceed(w.Write(frame.NewMulti(
				s.keys,
				[]telem.Series{
					telem.NewSeriesSecondsTSV(10, 11, 12),
					telem.NewSeriesV("hello", "world", "foo"),
				},
			)))
			MustSucceed(w.Commit())
			Expect(w.Close()).To(Succeed())
			iter := MustSucceed(s.dist.Framer.OpenIterator(ctx, iterator.Config{
				Keys:   []channel.Key{strCh.Key()},
				Bounds: telem.TimeRangeMax,
			}))
			Expect(iter.SeekFirst()).To(BeTrue())
			Expect(iter.Next(telem.TimeSpanMax)).To(BeTrue())
			Expect(telem.UnmarshalSeries[string](iter.Value().SeriesAt(0))).To(Equal([]string{"hello", "world", "foo"}))
			Expect(iter.Close()).To(Succeed())
		})
		It("Should write mixed fixed and variable channels", func(ctx SpecContext) {
			floatCh := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Float64T,
				LocalIndex: idxCh.LocalKey,
			}
			Expect(s.dist.Channel.Create(ctx, &floatCh)).To(Succeed())
			keys := []channel.Key{idxCh.Key(), floatCh.Key(), strCh.Key()}
			w := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:  keys,
				Start: 20 * telem.SecondTS,
				Sync:  new(true),
			}))
			MustSucceed(w.Write(frame.NewMulti(
				keys,
				[]telem.Series{
					telem.NewSeriesSecondsTSV(20, 21, 22),
					telem.NewSeriesV[float64](1.1, 2.2, 3.3),
					telem.NewSeriesV("a", "b", "c"),
				},
			)))
			MustSucceed(w.Commit())
			Expect(w.Close()).To(Succeed())
			iter := MustSucceed(s.dist.Framer.OpenIterator(ctx, iterator.Config{
				Keys:   []channel.Key{strCh.Key()},
				Bounds: (20 * telem.SecondTS).Range(23 * telem.SecondTS),
			}))
			Expect(iter.SeekFirst()).To(BeTrue())
			Expect(iter.Next(telem.TimeSpanMax)).To(BeTrue())
			Expect(telem.UnmarshalSeries[string](iter.Value().SeriesAt(0))).To(Equal([]string{"a", "b", "c"}))
			Expect(iter.Close()).To(Succeed())
		})
	})

	Describe("Open Errors", Ordered, func() {
		var s scenario
		BeforeAll(func(ctx SpecContext) { s = gatewayOnlyScenario(ctx) })
		AfterAll(func() { Expect(s.closer.Close()).To(Succeed()) })
		It("Should return an error if no keys are provided", func(ctx SpecContext) {
			Expect(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:  []channel.Key{},
				Start: 10 * telem.SecondTS,
				Sync:  new(true),
			})).Error().To(MatchError(ContainSubstring("keys: must be non-empty")))
		})
		It("Should return an error if the channel can't be found", func(ctx SpecContext) {
			Expect(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:  []channel.Key{channel.NewKey(0, 22), s.keys[0]},
				Start: 10 * telem.SecondTS,
				Sync:  new(true),
			})).Error().To(SatisfyAll(
				MatchError(query.ErrNotFound),
				MatchError(ContainSubstring("Channel")),
				MatchError(ContainSubstring("22")),
			))
		})
	})

	Describe("Frame Errors", Ordered, func() {
		var s scenario
		BeforeAll(func(ctx SpecContext) { s = peerOnlyScenario(ctx) })
		AfterAll(func() { Expect(s.closer.Close()).To(Succeed()) })
		It("Should return an error if a key is provided that is not in the list of keys provided to the writer", func(ctx SpecContext) {
			writer := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:  s.keys,
				Start: 10 * telem.SecondTS,
				Sync:  new(true),
			}))
			Expect(writer.Write(frame.NewMulti(
				append(s.keys, channel.NewKey(12, 22)),
				[]telem.Series{
					telem.NewSeriesV[int64](1, 2, 3),
					telem.NewSeriesV[int64](3, 4, 5),
					telem.NewSeriesV[int64](5, 6, 7),
					telem.NewSeriesV[int64](5, 6, 7),
				},
			))).Error().To(MatchError(validate.ErrValidation))
			Expect(writer.Close()).To(MatchError(validate.ErrValidation))
		})
	})

	Describe("Free Write Group Propagation", func() {
		It("Should propagate the writer's group to the streamer response", func(ctx SpecContext) {
			s := freeWriterScenario(ctx)
			defer func() { Expect(s.closer.Close()).To(Succeed()) }()
			streamer := MustSucceed(s.dist.Framer.NewStreamer(ctx, framer.StreamerConfig{
				Keys:        s.keys,
				SendOpenAck: new(true),
			}))
			_, out := confluence.Attach(streamer, 10)
			sCtx, cancel := signal.WithCancel(ctx)
			defer cancel()
			streamer.Flow(sCtx)
			var res framer.StreamerResponse
			Eventually(out.Outlet()).Should(Receive(&res))
			w := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:           s.keys,
				Start:          10 * telem.SecondTS,
				Sync:           new(true),
				ControlSubject: control.Subject{Group: 42},
			}))
			Expect(w.Write(frame.NewMulti(
				s.keys,
				[]telem.Series{
					telem.NewSeriesV[int64](1, 2, 3),
					telem.NewSeriesV[int64](3, 4, 5),
					telem.NewSeriesV[int64](5, 6, 7),
				},
			))).To(BeTrue())
			Eventually(out.Outlet()).Should(Receive(&res))
			Expect(res.Group).To(Equal(uint32(42)))
			Expect(w.Close()).To(Succeed())
		})
		It("Should set group to zero when the writer has no group", func(ctx SpecContext) {
			s := freeWriterScenario(ctx)
			defer func() { Expect(s.closer.Close()).To(Succeed()) }()
			streamer := MustSucceed(s.dist.Framer.NewStreamer(ctx, framer.StreamerConfig{
				Keys:        s.keys,
				SendOpenAck: new(true),
			}))
			_, out := confluence.Attach(streamer, 10)
			sCtx, cancel := signal.WithCancel(ctx)
			defer cancel()
			streamer.Flow(sCtx)
			var res framer.StreamerResponse
			Eventually(out.Outlet()).Should(Receive(&res))
			w := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:  s.keys,
				Start: 10 * telem.SecondTS,
				Sync:  new(true),
			}))
			Expect(w.Write(frame.NewMulti(
				s.keys,
				[]telem.Series{
					telem.NewSeriesV[int64](1, 2, 3),
					telem.NewSeriesV[int64](3, 4, 5),
					telem.NewSeriesV[int64](5, 6, 7),
				},
			))).To(BeTrue())
			Eventually(out.Outlet()).Should(Receive(&res))
			Expect(res.Group).To(Equal(uint32(0)))
			Expect(w.Close()).To(Succeed())
		})
	})

	Describe("Free Write Group Isolation", func() {
		It("Should propagate distinct groups from different writers", func(ctx SpecContext) {
			s := freeWriterScenario(ctx)
			defer func() { Expect(s.closer.Close()).To(Succeed()) }()
			streamer := MustSucceed(s.dist.Framer.NewStreamer(ctx, framer.StreamerConfig{
				Keys:        s.keys,
				SendOpenAck: new(true),
			}))
			_, out := confluence.Attach(streamer, 10)
			sCtx, cancel := signal.WithCancel(ctx)
			defer cancel()
			streamer.Flow(sCtx)
			var res framer.StreamerResponse
			Eventually(out.Outlet()).Should(Receive(&res))
			w1 := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:           s.keys,
				Start:          10 * telem.SecondTS,
				Sync:           new(true),
				ControlSubject: control.Subject{Group: 10},
			}))
			w2 := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:           s.keys,
				Start:          10 * telem.SecondTS,
				Sync:           new(true),
				ControlSubject: control.Subject{Group: 20},
			}))
			Expect(w1.Write(frame.NewMulti(
				s.keys,
				[]telem.Series{
					telem.NewSeriesV[int64](1, 2, 3),
					telem.NewSeriesV[int64](3, 4, 5),
					telem.NewSeriesV[int64](5, 6, 7),
				},
			))).To(BeTrue())
			Eventually(out.Outlet()).Should(Receive(&res))
			Expect(res.Group).To(Equal(uint32(10)))
			Expect(w2.Write(frame.NewMulti(
				s.keys,
				[]telem.Series{
					telem.NewSeriesV[int64](8, 9, 10),
					telem.NewSeriesV[int64](11, 12, 13),
					telem.NewSeriesV[int64](14, 15, 16),
				},
			))).To(BeTrue())
			Eventually(out.Outlet()).Should(Receive(&res))
			Expect(res.Group).To(Equal(uint32(20)))
			Expect(w1.Close()).To(Succeed())
			Expect(w2.Close()).To(Succeed())
		})
	})

	Describe("Free Write Alignment", Ordered, func() {
		It("Should correctly set alignments on indexed free channels", func(ctx SpecContext) {
			var (
				s = gatewayOnlyScenario(ctx)
			)
			defer func() { Expect(s.closer.Close()).To(Succeed()) }()
			var (
				idxCh = channel.Channel{
					Name:        "free_time",
					IsIndex:     true,
					DataType:    telem.TimeStampT,
					Leaseholder: cluster.NodeKeyFree,
					Virtual:     true,
				}
				dataCh = channel.Channel{
					Name:        "free",
					DataType:    telem.Float32T,
					Leaseholder: cluster.NodeKeyFree,
					Virtual:     true,
				}
			)
			Expect(s.dist.Channel.Create(ctx, &idxCh)).To(Succeed())
			dataCh.LocalIndex = idxCh.LocalKey
			Expect(s.dist.Channel.Create(ctx, &dataCh)).To(Succeed())

			keys := []channel.Key{idxCh.Key(), dataCh.Key()}
			streamer := MustSucceed(s.dist.Framer.NewStreamer(ctx, framer.StreamerConfig{
				Keys:        keys,
				SendOpenAck: new(true),
			}))
			_, out := confluence.Attach(streamer, 10)
			sCtx, cancel := signal.WithCancel(ctx)
			defer cancel()
			streamer.Flow(sCtx)
			var res framer.StreamerResponse
			Eventually(out.Outlet()).Should(Receive(&res))
			writer := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:  keys,
				Start: 10 * telem.SecondTS,
				Sync:  new(true),
			}))
			data := telem.NewSeriesV[int64](1, 2)
			idx := telem.NewSeriesSecondsTSV(10*telem.SecondTS, 11*telem.SecondTS)
			MustSucceed(writer.Write(frame.NewMulti(
				keys,
				[]telem.Series{idx, data},
			)))
			Eventually(out.Outlet()).Should(Receive(&res))
			Expect(res.Frame.KeysSlice()).To(Equal(keys))
			writtenData := res.Frame.Get(dataCh.Key()).Series[0]
			Expect(writtenData).To(telem.MatchSeriesData(data))
			writtenIdx := res.Frame.Get(idxCh.Key()).Series[0]
			Expect(writtenIdx).To(telem.MatchSeriesData(idx))
			Expect(writtenData.Alignment.DomainIndex()).To(BeEquivalentTo(cesium.ZeroLeadingAlignment + 1))
			Expect(writtenData.Alignment.SampleIndex()).To(BeEquivalentTo(0))
			Expect(writtenIdx.Alignment.DomainIndex()).To(BeEquivalentTo(cesium.ZeroLeadingAlignment + 1))
			Expect(writtenIdx.Alignment.SampleIndex()).To(BeEquivalentTo(0))
			data = telem.NewSeriesV[int64](3, 4)
			idx = telem.NewSeriesSecondsTSV(12*telem.SecondTS, 13*telem.SecondTS)
			MustSucceed(writer.Write(frame.NewMulti(
				keys,
				[]telem.Series{idx, data},
			)))
			Eventually(out.Outlet()).Should(Receive(&res))
			Expect(res.Frame.KeysSlice()).To(Equal(keys))
			writtenData = res.Frame.Get(dataCh.Key()).Series[0]
			Expect(writtenData).To(telem.MatchSeriesData(data))
			writtenIdx = res.Frame.Get(idxCh.Key()).Series[0]
			Expect(writtenIdx).To(telem.MatchSeriesData(idx))
			Expect(writtenData.Alignment.DomainIndex()).To(BeEquivalentTo(cesium.ZeroLeadingAlignment + 1))
			Expect(writtenData.Alignment.SampleIndex()).To(BeEquivalentTo(2))
			Expect(writtenIdx.Alignment.DomainIndex()).To(BeEquivalentTo(cesium.ZeroLeadingAlignment + 1))
			Expect(writtenIdx.Alignment.SampleIndex()).To(BeEquivalentTo(2))
			Expect(writer.Close()).To(Succeed())
		})
	})
})

type scenario struct {
	dist   mock.Node
	closer io.Closer
	name   string
	keys   channel.Keys
}

func newChannelSet() []channel.Channel {
	return []channel.Channel{
		{
			Name:     "test1",
			Virtual:  true,
			DataType: telem.Int64T,
		},
		{
			Name:     "test2",
			Virtual:  true,
			DataType: telem.Int64T,
		},
		{
			Name:     "test3",
			Virtual:  true,
			DataType: telem.Int64T,
		},
	}
}

func gatewayOnlyScenario(ctx context.Context) scenario {
	channels := newChannelSet()
	builder := mock.ProvisionCluster(ctx, 1)
	dist := builder.Nodes[1]
	Expect(dist.Channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
	keys := channel.KeysFromChannels(channels)
	return scenario{name: "Gateway Only", keys: keys, dist: dist, closer: builder}
}

func peerOnlyScenario(ctx context.Context) scenario {
	channels := newChannelSet()
	builder := mock.ProvisionCluster(ctx, 4)
	dist := builder.Nodes[1]
	for i, ch := range channels {
		ch.Leaseholder = cluster.NodeKey(i + 2)
		channels[i] = ch
	}
	Expect(dist.Channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
	Eventually(func(g Gomega) {
		var chs []channel.Channel
		err := dist.Channel.NewRetrieve().Entries(&chs).WhereKeys(channel.KeysFromChannels(channels)...).Exec(ctx, nil)
		g.Expect(err).To(Succeed())
		g.Expect(chs).To(HaveLen(len(channels)))
	}).Should(Succeed())
	keys := channel.KeysFromChannels(channels)
	return scenario{name: "Peer Only", keys: keys, dist: dist, closer: builder}
}

func mixedScenario(ctx context.Context) scenario {
	channels := newChannelSet()
	builder := mock.ProvisionCluster(ctx, 3)
	svc := builder.Nodes[1]
	for i, ch := range channels {
		ch.Leaseholder = cluster.NodeKey(i + 1)
		channels[i] = ch
	}
	Expect(svc.Channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
	Eventually(func(g Gomega) {
		var chs []channel.Channel
		err := svc.Channel.NewRetrieve().Entries(&chs).WhereKeys(channel.KeysFromChannels(channels)...).Exec(ctx, nil)
		g.Expect(err).To(Succeed())
		g.Expect(chs).To(HaveLen(len(channels)))
	}).Should(Succeed())
	keys := channel.KeysFromChannels(channels)
	return scenario{name: "Mixed Gateway and Peer", keys: keys, dist: svc, closer: builder}
}

func freeWriterScenario(ctx context.Context) scenario {
	channels := newChannelSet()
	builder := mock.ProvisionCluster(ctx, 3)
	svc := builder.Nodes[1]
	for i, ch := range channels {
		ch.Leaseholder = cluster.NodeKeyFree
		ch.Virtual = true
		channels[i] = ch
	}
	Expect(svc.Channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
	Eventually(func(g Gomega) {
		var chs []channel.Channel
		err := svc.Channel.NewRetrieve().Entries(&chs).WhereKeys(channel.KeysFromChannels(channels)...).Exec(ctx, nil)
		g.Expect(err).To(Succeed())
		g.Expect(chs).To(HaveLen(len(channels)))
	}).Should(Succeed())
	keys := channel.KeysFromChannels(channels)
	return scenario{name: "Free Writes", keys: keys, dist: svc, closer: builder}
}
