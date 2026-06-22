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
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// openWriter opens a writer against n, attaching the supplied channel metadata to the
// config. The distribution writer no longer resolves channels itself — its caller (the
// service layer in production, this helper in tests) supplies them. The writer maps
// channels by key and only uses those whose key is in cfg.Keys, so passing a superset is
// fine.
func openWriter(ctx context.Context, n mock.Node, channels []channel.Channel, cfg writer.Config) (*writer.Writer, error) {
	cfg.Channels = channels
	return n.Framer.OpenWriter(ctx, cfg)
}

var _ = Describe("Writer", func() {
	Describe("Happy Path", Ordered, func() {
		scenarios := []func(context.Context) scenario{
			gatewayOnlyScenario,
			peerOnlyScenario,
			mixedScenario,
			freeWriterScenario,
		}
		for i, sF := range scenarios {
			var s scenario
			BeforeAll(func(ctx SpecContext) { s = DeferClose(sF(ctx)) })
			Specify(fmt.Sprintf("Scenario: %v - Happy Path", i), func(ctx SpecContext) {
				writer := MustOpen(openWriter(ctx, s.dist, s.channels, writer.Config{
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
			builder := mock.OpenCluster(ctx, 1)
			dist := builder.Nodes[1]
			idxCh = channel.Channel{
				Name:     "variable_time",
				IsIndex:  true,
				DataType: telem.TimeStampT,
			}
			idxCh = MustSucceed(dist.Channel.Create(ctx, []channel.Channel{idxCh}))[0]
			strCh = channel.Channel{
				Name:       "variable_str",
				DataType:   telem.StringT,
				LocalIndex: idxCh.LocalKey,
			}
			strCh = MustSucceed(dist.Channel.Create(ctx, []channel.Channel{strCh}))[0]
			s = DeferClose(scenario{
				dist:     dist,
				closer:   builder,
				name:     "Variable",
				keys:     []channel.Key{idxCh.Key(), strCh.Key()},
				channels: []channel.Channel{idxCh, strCh},
			})
		})
		It("Should write and read persisted string data", func(ctx SpecContext) {
			w := MustOpen(openWriter(ctx, s.dist, s.channels, writer.Config{
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
			iter := MustOpen(s.dist.Framer.OpenIterator(ctx, iterator.Config{
				Keys:   []channel.Key{strCh.Key()},
				Bounds: telem.TimeRangeMax,
			}))
			Expect(iter.SeekFirst()).To(BeTrue())
			Expect(iter.Next(telem.TimeSpanMax)).To(BeTrue())
			Expect(telem.UnmarshalSeries[string](iter.Value().SeriesAt(0))).To(Equal([]string{"hello", "world", "foo"}))
		})
		It("Should write mixed fixed and variable channels", func(ctx SpecContext) {
			floatCh := channel.Channel{
				Name:       "variable_float",
				DataType:   telem.Float64T,
				LocalIndex: idxCh.LocalKey,
			}
			floatCh = MustSucceed(s.dist.Channel.Create(ctx, []channel.Channel{floatCh}))[0]
			keys := []channel.Key{idxCh.Key(), floatCh.Key(), strCh.Key()}
			w := MustOpen(openWriter(ctx, s.dist, []channel.Channel{idxCh, floatCh, strCh}, writer.Config{
				Keys:  keys,
				Start: 20 * telem.SecondTS,
				Sync:  new(true),
			}))
			MustSucceed(w.Write(frame.NewMulti(
				keys,
				[]telem.Series{
					telem.NewSeriesSecondsTSV(20, 21, 22),
					telem.NewSeriesV(1.1, 2.2, 3.3),
					telem.NewSeriesV("a", "b", "c"),
				},
			)))
			MustSucceed(w.Commit())
			iter := MustOpen(s.dist.Framer.OpenIterator(ctx, iterator.Config{
				Keys:   []channel.Key{strCh.Key()},
				Bounds: (20 * telem.SecondTS).Range(23 * telem.SecondTS),
			}))
			Expect(iter.SeekFirst()).To(BeTrue())
			Expect(iter.Next(telem.TimeSpanMax)).To(BeTrue())
			Expect(telem.UnmarshalSeries[string](iter.Value().SeriesAt(0))).To(Equal([]string{"a", "b", "c"}))
		})
	})

	Describe("Open Errors", Ordered, func() {
		var s scenario
		BeforeAll(func(ctx SpecContext) { s = DeferClose(gatewayOnlyScenario(ctx)) })
		It("Should return an error if no keys are provided", func(ctx SpecContext) {
			Expect(openWriter(ctx, s.dist, s.channels, writer.Config{
				Keys:  []channel.Key{},
				Start: 10 * telem.SecondTS,
				Sync:  new(true),
			})).Error().To(MatchError(ContainSubstring("keys: must be non-empty")))
		})
		It("Should return an error if metadata is missing for a provided key", func(ctx SpecContext) {
			Expect(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:     []channel.Key{channel.NewKey(0, 22), s.keys[0]},
				Channels: s.channels[:1],
				Start:    10 * telem.SecondTS,
				Sync:     new(true),
			})).Error().To(SatisfyAll(
				MatchError(ContainSubstring("missing channel metadata")),
				MatchError(ContainSubstring("22")),
			))
		})
	})

	Describe("Frame Errors", Ordered, func() {
		var s scenario
		BeforeAll(func(ctx SpecContext) { s = DeferClose(peerOnlyScenario(ctx)) })
		It("Should return an error if a key is provided that is not in the list of keys provided to the writer", func(ctx SpecContext) {
			writer := MustSucceed(openWriter(ctx, s.dist, s.channels, writer.Config{
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

	Describe("Series Validation", func() {
		Describe("Misaligned Fixed Density", func() {
			It("Should return an error when series data is not aligned to density", func(ctx SpecContext) {
				s := DeferClose(gatewayOnlyScenario(ctx))
				w := MustSucceed(openWriter(ctx, s.dist, s.channels, writer.Config{
					Keys:  s.keys,
					Start: 10 * telem.SecondTS,
					Sync:  new(true),
				}))
				Expect(w.Write(frame.NewMulti(
					s.keys,
					[]telem.Series{
						{DataType: telem.Int64T, Data: make([]byte, 7)},
						telem.NewSeriesV[int64](1),
						telem.NewSeriesV[int64](1),
					},
				))).Error().To(MatchError(validate.ErrValidation))
				Expect(w.Close()).To(MatchError(validate.ErrValidation))
			})
		})

		Describe("Wrong Data Type", func() {
			It("Should return an error when series data type does not match channel", func(ctx SpecContext) {
				s := DeferClose(gatewayOnlyScenario(ctx))
				w := MustSucceed(openWriter(ctx, s.dist, s.channels, writer.Config{
					Keys:  s.keys,
					Start: 10 * telem.SecondTS,
					Sync:  new(true),
				}))
				Expect(w.Write(frame.NewMulti(
					s.keys,
					[]telem.Series{
						telem.NewSeriesV(1.0),
						telem.NewSeriesV[int64](1),
						telem.NewSeriesV[int64](1),
					},
				))).Error().To(MatchError(validate.ErrValidation))
				Expect(w.Close()).To(MatchError(validate.ErrValidation))
			})
		})

		Describe("Invalid JSON", Ordered, func() {
			var s scenario
			BeforeAll(func(ctx SpecContext) {
				builder := mock.OpenCluster(ctx, 1)
				dist := builder.Nodes[1]
				idxCh := channel.Channel{
					Name:     "invalid_json_time",
					IsIndex:  true,
					DataType: telem.TimeStampT,
				}
				idxCh = MustSucceed(dist.Channel.Create(ctx, []channel.Channel{idxCh}))[0]
				jsonCh := channel.Channel{
					Name:       "invalid_json",
					DataType:   telem.JSONT,
					LocalIndex: idxCh.LocalKey,
				}
				jsonCh = MustSucceed(dist.Channel.Create(ctx, []channel.Channel{jsonCh}))[0]
				s = DeferClose(scenario{
					dist:     dist,
					closer:   builder,
					keys:     []channel.Key{idxCh.Key(), jsonCh.Key()},
					channels: []channel.Channel{idxCh, jsonCh},
				})
			})
			It("Should return an error when JSON series contains invalid JSON", func(ctx SpecContext) {
				w := MustSucceed(openWriter(ctx, s.dist, s.channels, writer.Config{
					Keys:  s.keys,
					Start: 10 * telem.SecondTS,
					Sync:  new(true),
				}))
				invalidJSON := telem.MarshalVariableSample([]byte(`{not json}`))
				Expect(w.Write(frame.NewMulti(
					s.keys,
					[]telem.Series{
						telem.NewSeriesSecondsTSV(10),
						{DataType: telem.JSONT, Data: invalidJSON},
					},
				))).Error().To(MatchError(validate.ErrValidation))
				Expect(w.Close()).To(MatchError(validate.ErrValidation))
			})
		})

		Describe("Invalid UTF-8", Ordered, func() {
			var s scenario
			BeforeAll(func(ctx SpecContext) {
				builder := mock.OpenCluster(ctx, 1)
				dist := builder.Nodes[1]
				idxCh := channel.Channel{
					Name:     "invalid_utf8_time",
					IsIndex:  true,
					DataType: telem.TimeStampT,
				}
				idxCh = MustSucceed(dist.Channel.Create(ctx, []channel.Channel{idxCh}))[0]
				strCh := channel.Channel{
					Name:       "invalid_utf8_str",
					DataType:   telem.StringT,
					LocalIndex: idxCh.LocalKey,
				}
				strCh = MustSucceed(dist.Channel.Create(ctx, []channel.Channel{strCh}))[0]
				s = DeferClose(scenario{
					dist:     dist,
					closer:   builder,
					keys:     []channel.Key{idxCh.Key(), strCh.Key()},
					channels: []channel.Channel{idxCh, strCh},
				})
			})
			It("Should return an error when string series contains invalid UTF-8", func(ctx SpecContext) {
				w := MustSucceed(openWriter(ctx, s.dist, s.channels, writer.Config{
					Keys:  s.keys,
					Start: 10 * telem.SecondTS,
					Sync:  new(true),
				}))
				invalidUTF8 := telem.MarshalVariableSample([]byte{0xFF, 0xFE})
				Expect(w.Write(frame.NewMulti(
					s.keys,
					[]telem.Series{
						telem.NewSeriesSecondsTSV(10),
						{DataType: telem.StringT, Data: invalidUTF8},
					},
				))).Error().To(MatchError(validate.ErrValidation))
				Expect(w.Close()).To(MatchError(validate.ErrValidation))
			})
		})

		Describe("Malformed Variable Prefix", Ordered, func() {
			var s scenario
			BeforeAll(func(ctx SpecContext) {
				builder := mock.OpenCluster(ctx, 1)
				dist := builder.Nodes[1]
				idxCh := channel.Channel{
					Name:     "malformed_prefix_time",
					IsIndex:  true,
					DataType: telem.TimeStampT,
				}
				idxCh = MustSucceed(dist.Channel.Create(ctx, []channel.Channel{idxCh}))[0]
				strCh := channel.Channel{
					Name:       "malformed_prefix_str",
					DataType:   telem.StringT,
					LocalIndex: idxCh.LocalKey,
				}
				strCh = MustSucceed(dist.Channel.Create(ctx, []channel.Channel{strCh}))[0]
				s = DeferClose(scenario{
					dist:     dist,
					closer:   builder,
					keys:     []channel.Key{idxCh.Key(), strCh.Key()},
					channels: []channel.Channel{idxCh, strCh},
				})
			})
			It("Should return an error when variable-density prefix is malformed", func(ctx SpecContext) {
				w := MustSucceed(openWriter(ctx, s.dist, s.channels, writer.Config{
					Keys:  s.keys,
					Start: 10 * telem.SecondTS,
					Sync:  new(true),
				}))
				valid := telem.NewSeriesV("ok")
				valid.Data = append(valid.Data, 0xFF, 0xFF)
				Expect(w.Write(frame.NewMulti(
					s.keys,
					[]telem.Series{
						telem.NewSeriesSecondsTSV(10),
						valid,
					},
				))).Error().To(MatchError(validate.ErrValidation))
				Expect(w.Close()).To(MatchError(validate.ErrValidation))
			})
		})
	})

	Describe("Free Write Group Propagation", func() {
		It("Should propagate the writer's group to the streamer response", func(ctx SpecContext) {
			s := DeferClose(freeWriterScenario(ctx))
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
			w := MustOpen(openWriter(ctx, s.dist, s.channels, writer.Config{
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
		})
		It("Should set group to zero when the writer has no group", func(ctx SpecContext) {
			s := DeferClose(freeWriterScenario(ctx))
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
			w := MustOpen(openWriter(ctx, s.dist, s.channels, writer.Config{
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
		})
	})

	Describe("Free Write Group Isolation", func() {
		It("Should propagate distinct groups from different writers", func(ctx SpecContext) {
			s := DeferClose(freeWriterScenario(ctx))
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
			w1 := MustOpen(openWriter(ctx, s.dist, s.channels, writer.Config{
				Keys:           s.keys,
				Start:          10 * telem.SecondTS,
				Sync:           new(true),
				ControlSubject: control.Subject{Group: 10},
			}))
			w2 := MustOpen(openWriter(ctx, s.dist, s.channels, writer.Config{
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
		})
	})

	Describe("Auto-Index", func() {
		It("Should auto-stamp from the leaseholder when the data channel lives on a peer", func(ctx SpecContext) {
			builder := DeferClose(mock.OpenCluster(ctx, 2))
			gw := builder.Nodes[1]
			peer := node.Key(2)

			idx := channel.Channel{
				Name:        "auto_index_time",
				IsIndex:     true,
				DataType:    telem.TimeStampT,
				Leaseholder: peer,
			}
			idx = MustSucceed(gw.Channel.Create(ctx, []channel.Channel{idx}))[0]
			data := channel.Channel{
				Name:        "auto_index_data",
				DataType:    telem.Float64T,
				LocalIndex:  idx.LocalKey,
				Leaseholder: peer,
			}
			data = MustSucceed(gw.Channel.Create(ctx, []channel.Channel{data}))[0]

			before := telem.Now()
			w := MustSucceed(openWriter(ctx, gw, []channel.Channel{idx, data}, writer.Config{
				Keys:      []channel.Key{data.Key()},
				Sync:      new(true),
				AutoIndex: new(true),
				Start:     1 * telem.SecondTS,
			}))
			Expect((w.Write(frame.NewUnary(
				data.Key(),
				telem.NewSeriesV(1.1, 2.2, 3.3),
			)))).To(BeTrue())
			MustSucceed(w.Commit())
			Expect(w.Close()).To(Succeed())
			after := telem.Now()

			iter := MustSucceed(gw.Framer.OpenIterator(ctx, iterator.Config{
				Keys:   []channel.Key{idx.Key()},
				Bounds: telem.TimeRangeMax,
			}))
			Expect(iter.SeekFirst()).To(BeTrue())
			Expect(iter.Next(telem.TimeSpanMax)).To(BeTrue())
			ts := telem.UnmarshalSeries[telem.TimeStamp](iter.Value().SeriesAt(0))
			Expect(iter.Close()).To(Succeed())

			Expect(ts).To(HaveLen(3))
			Expect(ts[0]).To(BeNumerically(">=", before))
			Expect(ts[2]).To(BeNumerically("<=", after+telem.TimeStamp(3)))
			Expect(ts[1]).To(Equal(ts[0] + 1))
			Expect(ts[2]).To(Equal(ts[0] + 2))
		})
	})

	Describe("Free Write Alignment", Ordered, func() {
		It("Should correctly set alignments on indexed free channels", func(ctx SpecContext) {
			var s = DeferClose(gatewayOnlyScenario(ctx))
			var (
				idxCh = channel.Channel{
					Name:        "free_time",
					IsIndex:     true,
					DataType:    telem.TimeStampT,
					Leaseholder: node.KeyFree,
					Virtual:     true,
				}
				dataCh = channel.Channel{
					Name:        "free",
					DataType:    telem.Float32T,
					Leaseholder: node.KeyFree,
					Virtual:     true,
				}
			)
			idxCh = MustSucceed(s.dist.Channel.Create(ctx, []channel.Channel{idxCh}))[0]
			dataCh.LocalIndex = idxCh.LocalKey
			dataCh = MustSucceed(s.dist.Channel.Create(ctx, []channel.Channel{dataCh}))[0]

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
			writer := MustOpen(openWriter(ctx, s.dist, []channel.Channel{idxCh, dataCh}, writer.Config{
				Keys:  keys,
				Start: 10 * telem.SecondTS,
				Sync:  new(true),
			}))
			data := telem.NewSeriesV[float32](1, 2)
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
			data = telem.NewSeriesV[float32](3, 4)
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
		})
	})
})

type scenario struct {
	dist     mock.Node
	closer   io.Closer
	name     string
	keys     channel.Keys
	channels []channel.Channel
}

func (s scenario) Close() error { return s.closer.Close() }

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
	builder := mock.OpenCluster(ctx, 1)
	dist := builder.Nodes[1]
	channels = MustSucceed(dist.Channel.Create(ctx, channels))
	keys := channel.KeysFromChannels(channels)
	return scenario{name: "Gateway Only", keys: keys, dist: dist, closer: builder, channels: channels}
}

func peerOnlyScenario(ctx context.Context) scenario {
	channels := newChannelSet()
	builder := mock.OpenCluster(ctx, 4)
	dist := builder.Nodes[1]
	for i, ch := range channels {
		ch.Leaseholder = node.Key(i + 2)
		channels[i] = ch
	}
	channels = MustSucceed(dist.Channel.Create(ctx, channels))
	keys := channel.KeysFromChannels(channels)
	return scenario{name: "Peer Only", keys: keys, dist: dist, closer: builder, channels: channels}
}

func mixedScenario(ctx context.Context) scenario {
	channels := newChannelSet()
	builder := mock.OpenCluster(ctx, 3)
	svc := builder.Nodes[1]
	for i, ch := range channels {
		ch.Leaseholder = node.Key(i + 1)
		channels[i] = ch
	}
	channels = MustSucceed(svc.Channel.Create(ctx, channels))
	keys := channel.KeysFromChannels(channels)
	return scenario{name: "Mixed Gateway and Peer", keys: keys, dist: svc, closer: builder, channels: channels}
}

func freeWriterScenario(ctx context.Context) scenario {
	channels := newChannelSet()
	builder := mock.OpenCluster(ctx, 3)
	svc := builder.Nodes[1]
	for i, ch := range channels {
		ch.Leaseholder = node.KeyFree
		ch.Virtual = true
		channels[i] = ch
	}
	channels = MustSucceed(svc.Channel.Create(ctx, channels))
	keys := channel.KeysFromChannels(channels)
	return scenario{name: "Free Writes", keys: keys, dist: svc, closer: builder, channels: channels}
}
