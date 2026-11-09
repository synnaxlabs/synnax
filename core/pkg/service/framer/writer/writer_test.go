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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/gorp"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

func malformedVariableSeries() telem.Series {
	s := telem.NewSeriesV("ok")
	s.Data = append(s.Data, 0xFF, 0xFF)
	return s
}

var _ = Describe("Writer", func() {
	createIndexed := func(
		ctx SpecContext, dataType telem.DataType,
	) (channel.Channel, channel.Channel) {
		GinkgoHelper()
		idxCh := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.TimestampT,
			IsIndex:  true,
		}
		Expect(mockNode.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &idxCh)
		})).To(Succeed())
		dataCh := channel.Channel{
			Name:       UniqueChannelName(),
			DataType:   dataType,
			LocalIndex: idxCh.LocalKey,
		}
		Expect(mockNode.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &dataCh)
		})).To(Succeed())
		return idxCh, dataCh
	}

	createVirtual := func(ctx SpecContext) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Int64T,
			Virtual:  true,
		}
		Expect(mockNode.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}

	It("Should return no end from a commit that fails", func(ctx SpecContext) {
		// Commits walk a map of indexes, so most commit before the one that fails.
		keys := make(channel.Keys, 8)
		series := make([]telem.Series, len(keys))
		for i := range keys {
			ch := channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.TimestampT,
				IsIndex:  true,
			}
			Expect(mockNode.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Create(ctx, &ch)
			})).To(Succeed())
			keys[i] = ch.Key()
			series[i] = telem.NewSeriesSecondsTSV(1, 30)
		}
		live := MustSucceed(writerSvc.Open(ctx, writer.Config{
			Keys:  keys[:1],
			Start: 10 * telem.SecondTS,
			Sync:  new(true),
		}))
		fill := MustSucceed(writerSvc.Open(ctx, writer.Config{
			Keys:             keys,
			Start:            1 * telem.SecondTS,
			Mode:             writer.ModePersistOnly,
			EnableAutoCommit: new(false),
			Sync:             new(true),
		}))
		Expect(fill.Write(frame.NewMulti(keys, series))).To(BeTrue())
		Expect(fill.Commit()).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("cannot write to time range")),
		))
		Expect(fill.Close()).To(MatchError(validate.ErrValidation))
		Expect(live.Close()).To(Succeed())
	})

	DescribeTable("Should return ErrClosed when a canceled context ends the pipeline",
		func(ctx SpecContext, exec func(*writer.Writer, frame.Frame)) {
			idxCh, dataCh := createIndexed(ctx, telem.Float64T)
			keys := channel.Keys{idxCh.Key(), dataCh.Key()}
			cfg := writer.Config{
				Keys:              keys,
				Start:             telem.SecondTS,
				Authorities:       []control.Authority{control.AuthorityAbsolute},
				ErrOnUnauthorized: new(true),
				Sync:              new(true),
			}
			cCtx, cancel := context.WithCancel(ctx)
			defer cancel()
			w := MustSucceed(writerSvc.Open(cCtx, cfg))
			cancel()
			Eventually(func() error {
				next, err := writerSvc.Open(ctx, cfg)
				if err != nil {
					return err
				}
				return next.Close()
			}).Should(Succeed())
			exec(w, frame.NewMulti(keys, []telem.Series{
				telem.NewSeriesSecondsTSV(1),
				telem.NewSeriesV(1.0),
			}))
			Expect(w.Close()).To(Succeed())
		},
		Entry("Write", func(w *writer.Writer, fr frame.Frame) {
			GinkgoHelper()
			Expect(w.Write(fr)).Error().To(MatchError(writer.ErrClosed))
		}),
		Entry("Commit", func(w *writer.Writer, _ frame.Frame) {
			GinkgoHelper()
			Expect(w.Commit()).Error().To(MatchError(writer.ErrClosed))
		}),
	)

	It(
		"Should report a commit without control",
		func(ctx SpecContext) {
			ch := channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.TimestampT,
				IsIndex:  true,
			}
			Expect(mockNode.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Create(ctx, &ch)
			})).To(Succeed())
			keys := channel.Keys{ch.Key()}
			w := MustSucceed(writerSvc.Open(ctx, writer.Config{
				ControlSubject:   control.Subject{Key: "low"},
				Keys:             keys,
				Start:            10 * telem.SecondTS,
				Authorities:      []control.Authority{1},
				EnableAutoCommit: new(false),
				Sync:             new(true),
			}))
			Expect(w.Write(frame.NewUnary(
				ch.Key(),
				telem.NewSeriesSecondsTSV(10),
			))).To(BeTrue())
			holder := MustSucceed(writerSvc.Open(ctx, writer.Config{
				ControlSubject: control.Subject{Key: "high"},
				Keys:           keys,
				Start:          10 * telem.SecondTS,
				Authorities:    []control.Authority{control.AuthorityAbsolute},
				Sync:           new(true),
			}))
			end, authorized := MustSucceed2(w.Commit())
			Expect(authorized).To(BeFalse())
			Expect(end).To(BeZero())
			Expect(holder.Close()).To(Succeed())
			Expect(w.Write(frame.NewUnary(
				ch.Key(),
				telem.NewSeriesSecondsTSV(11),
			))).To(BeTrue())
			Expect(MustCommit(w)).To(Equal(11*telem.SecondTS + 1))
			Expect(w.Close()).To(Succeed())
		},
	)

	Describe("Open Errors", func() {
		It(
			"Should return a validation error when opening with no keys",
			func(ctx SpecContext) {
				Expect(writerSvc.Open(ctx, writer.Config{Start: telem.SecondTS})).
					Error().To(MatchError(ContainSubstring("keys: must be non-empty")))
			},
		)
		It(
			"Should return a validation error when opening a stream with no keys",
			func(ctx SpecContext) {
				Expect(writerSvc.NewStream(ctx, writer.Config{Start: telem.SecondTS})).
					Error().To(MatchError(ContainSubstring("keys: must be non-empty")))
			},
		)
		It(
			"Should return an error when opening with a nonexistent channel",
			func(ctx SpecContext) {
				Expect(writerSvc.Open(ctx, writer.Config{
					Keys: channel.Keys{
						distchannel.NewKey(mockNode.Identity.HostKey, 9999),
					},
					Start: telem.SecondTS,
				})).Error().To(MatchError(query.ErrNotFound))
			},
		)
		It(
			"Should return an error when opening a stream with a nonexistent channel",
			func(ctx SpecContext) {
				Expect(writerSvc.NewStream(ctx, writer.Config{
					Keys: channel.Keys{
						distchannel.NewKey(mockNode.Identity.HostKey, 9999),
					},
					Start: telem.SecondTS,
				})).Error().To(MatchError(query.ErrNotFound))
			},
		)
		It(
			"Should return an error when opening with a nonexistent free channel",
			func(ctx SpecContext) {
				Expect(writerSvc.Open(ctx, writer.Config{
					Keys:  channel.Keys{distchannel.NewKey(node.KeyFree, 9999)},
					Start: telem.SecondTS,
				})).Error().To(MatchError(query.ErrNotFound))
			},
		)
		It(
			"Should return an error when a nonexistent free channel is mixed with valid keys",
			func(ctx SpecContext) {
				ch := createVirtual(ctx)
				Expect(writerSvc.NewStream(ctx, writer.Config{
					Keys: channel.Keys{
						ch.Key(),
						distchannel.NewKey(node.KeyFree, 9999),
					},
					Start: telem.SecondTS,
				})).Error().To(MatchError(query.ErrNotFound))
			},
		)
	})

	Describe("Reserved Channels", func() {
		It("Should refuse a writer on a reserved channel", func(ctx SpecContext) {
			ch := createVirtual(ctx)
			svc := MustSucceed(writer.NewService(writer.ServiceConfig{
				Framer:   mockNode.Framer,
				Channel:  channelSvc,
				DB:       mockNode.DB,
				Reserved: func(k channel.Key) bool { return k == ch.Key() },
			}))
			Expect(svc.NewStream(ctx, writer.Config{
				Keys:  channel.Keys{ch.Key()},
				Start: telem.SecondTS,
			})).Error().To(MatchError(fmt.Sprintf(
				"channel %d holds changes that only the Core writes: validation error",
				ch.Key(),
			)))
		})
	})

	Describe("Frame Errors", func() {
		It(
			"Should return an error when a frame key is not in the writer's key set",
			func(ctx SpecContext) {
				ch, other := createVirtual(ctx), createVirtual(ctx)
				w := MustSucceed(writerSvc.Open(ctx, writer.Config{
					Keys:  channel.Keys{ch.Key()},
					Start: 10 * telem.SecondTS,
					Sync:  new(true),
				}))
				Expect(w.Write(frame.NewMulti(
					[]channel.Key{ch.Key(), other.Key()},
					[]telem.Series{
						telem.NewSeriesV[int64](1, 2, 3),
						telem.NewSeriesV[int64](4, 5, 6),
					},
				))).Error().To(And(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("invalid key")),
				))
				Expect(w.Close()).To(MatchError(validate.ErrValidation))
			},
		)
		It("Should reject an out-of-bounds command", func(ctx SpecContext) {
			ch := createVirtual(ctx)
			s := MustSucceed(writerSvc.NewStream(ctx, writer.Config{
				Keys:  channel.Keys{ch.Key()},
				Start: 10 * telem.SecondTS,
			}))
			sCtx, cancel := signal.Isolated()
			defer cancel()
			inlet, outlet := confluence.Attach(s)
			s.Flow(
				sCtx,
				confluence.CloseOutputInletsOnExit(),
				confluence.CancelOnFail(),
			)
			inlet.Inlet() <- writer.Request{Command: writer.CommandEndDomain + 1}
			Eventually(outlet.Outlet()).Should(BeClosed())
			Expect(sCtx.Wait()).To(MatchError(validate.ErrValidation))
		})
	})

	Describe("End Domain", func() {
		frameAt := func(keys channel.Keys, seconds ...telem.TimeStamp) frame.Frame {
			stamps := make([]telem.TimeStamp, len(seconds))
			values := make([]int64, len(seconds))
			for i, s := range seconds {
				stamps[i] = s * telem.SecondTS
				values[i] = int64(s)
			}
			return frame.NewMulti(keys, []telem.Series{
				telem.NewSeries(stamps),
				telem.NewSeries(values),
			})
		}
		openWriter := func(ctx SpecContext, keys channel.Keys) *writer.Writer {
			GinkgoHelper()
			return MustSucceed(writerSvc.Open(ctx, writer.Config{
				Keys:  keys,
				Start: 10 * telem.SecondTS,
				Sync:  new(true),
			}))
		}
		It("Should free the time between two domains", func(ctx SpecContext) {
			idx, data := createIndexed(ctx, telem.Int64T)
			keys := channel.Keys{idx.Key(), data.Key()}
			w := openWriter(ctx, keys)
			Expect(w.Write(frameAt(keys, 10, 11))).To(BeTrue())
			MustCommit(w)
			Expect(w.EndDomain(channel.Keys{idx.Key()})).To(BeTrue())
			Expect(w.Write(frameAt(keys, 20, 21))).To(BeTrue())
			MustCommit(w)
			fill := MustSucceed(writerSvc.Open(ctx, writer.Config{
				Keys:  keys,
				Start: 12 * telem.SecondTS,
				Mode:  ts.WriterModePersistOnly,
				Sync:  new(true),
			}))
			Expect(fill.Write(frameAt(keys, 12, 13))).To(BeTrue())
			MustCommit(fill)
			Expect(fill.Close()).To(Succeed())
			Expect(w.Close()).To(Succeed())
		})
		It("Should report an index it does not control", func(ctx SpecContext) {
			idx, data := createIndexed(ctx, telem.Int64T)
			keys := channel.Keys{idx.Key(), data.Key()}
			holder := openWriter(ctx, keys)
			other := openWriter(ctx, keys)
			Expect(holder.Write(frameAt(keys, 10, 11))).To(BeTrue())
			MustCommit(holder)
			Expect(other.EndDomain(channel.Keys{idx.Key()})).To(BeFalse())
			Expect(other.Close()).To(Succeed())
			Expect(holder.Close()).To(Succeed())
		})
		It("Should report a commit that fails", func(ctx SpecContext) {
			idx, data := createIndexed(ctx, telem.Int64T)
			keys := channel.Keys{idx.Key(), data.Key()}
			live := openWriter(ctx, keys)
			fill := MustSucceed(writerSvc.Open(ctx, writer.Config{
				Keys:             keys,
				Start:            1 * telem.SecondTS,
				Mode:             ts.WriterModePersistOnly,
				EnableAutoCommit: new(false),
				Sync:             new(true),
			}))
			Expect(fill.Write(frameAt(keys, 1, 30))).To(BeTrue())
			Expect(fill.EndDomain(channel.Keys{idx.Key()})).Error().To(And(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("cannot write to time range")),
			))
			Expect(fill.Close()).To(MatchError(validate.ErrValidation))
			Expect(live.Close()).To(Succeed())
		})
		DescribeTable(
			"Should fail for keys that are not indexes of the writer",
			func(
				ctx SpecContext,
				indexes func(data, other channel.Channel) channel.Keys,
				message string,
			) {
				idx, data := createIndexed(ctx, telem.Int64T)
				other, _ := createIndexed(ctx, telem.Int64T)
				w := openWriter(ctx, channel.Keys{idx.Key(), data.Key()})
				Expect(w.EndDomain(indexes(data, other))).Error().To(And(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(message)),
				))
				Expect(w.Close()).To(MatchError(validate.ErrValidation))
			},
			Entry(
				"a data channel",
				func(data, _ channel.Channel) channel.Keys {
					return channel.Keys{data.Key()}
				},
				"is not an index",
			),
			Entry(
				"a key the writer was not opened with",
				func(_, other channel.Channel) channel.Keys {
					return channel.Keys{other.Key()}
				},
				"invalid key",
			),
			Entry(
				"no keys",
				func(_, _ channel.Channel) channel.Keys { return nil },
				"no indexes",
			),
		)
	})

	Describe("Series Validation", func() {
		DescribeTable("Should reject series that are invalid for their channel",
			func(ctx SpecContext, dataType telem.DataType, series telem.Series) {
				idxCh, dataCh := createIndexed(ctx, dataType)
				w := MustSucceed(writerSvc.Open(ctx, writer.Config{
					Keys:  channel.Keys{idxCh.Key(), dataCh.Key()},
					Start: 10 * telem.SecondTS,
					Sync:  new(true),
				}))
				Expect(w.Write(frame.NewMulti(
					[]channel.Key{idxCh.Key(), dataCh.Key()},
					[]telem.Series{telem.NewSeriesSecondsTSV(10), series},
				))).Error().To(MatchError(validate.ErrValidation))
				Expect(w.Close()).To(MatchError(validate.ErrValidation))
			},
			Entry(
				"misaligned density",
				telem.Int64T,
				telem.Series{DataType: telem.Int64T, Data: make([]byte, 7)},
			),
			Entry("wrong data type", telem.Int64T, telem.NewSeriesV(1.0)),
			Entry(
				"invalid JSON",
				telem.JSONT,
				telem.Series{
					DataType: telem.JSONT,
					Data:     telem.MarshalVariableSample([]byte(`{not json}`)),
				},
			),
			Entry(
				"invalid UTF-8",
				telem.StringT,
				telem.Series{
					DataType: telem.StringT,
					Data:     telem.MarshalVariableSample([]byte{0xFF, 0xFE}),
				},
			),
			Entry(
				"malformed variable-length prefix",
				telem.StringT,
				malformedVariableSeries(),
			),
		)

		It(
			"Should treat int64 and timestamp series as equivalent",
			func(ctx SpecContext) {
				idxCh, dataCh := createIndexed(ctx, telem.Int64T)
				w := MustOpen(writerSvc.Open(ctx, writer.Config{
					Keys:  channel.Keys{idxCh.Key(), dataCh.Key()},
					Start: 10 * telem.SecondTS,
					Sync:  new(true),
				}))
				Expect(w.Write(frame.NewMulti(
					[]channel.Key{idxCh.Key(), dataCh.Key()},
					[]telem.Series{
						telem.NewSeriesV(
							int64(10*telem.SecondTS),
							int64(11*telem.SecondTS),
						),
						telem.NewSeriesSecondsTSV(1, 2),
					},
				))).To(BeTrue())
			},
		)
	})

	Describe("Free Channel Writes", func() {
		It(
			"Should stamp alignments on free writes and stream them through the relay",
			func(ctx SpecContext) {
				idxCh := channel.Channel{
					Name:        UniqueChannelName(),
					IsIndex:     true,
					DataType:    telem.TimestampT,
					Leaseholder: node.KeyFree,
					Virtual:     true,
				}
				Expect(mockNode.DB.WithTx(ctx, func(tx gorp.Tx) error {
					return channelSvc.NewWriter(tx).Create(ctx, &idxCh)
				})).To(Succeed())
				dataCh := channel.Channel{
					Name:        UniqueChannelName(),
					DataType:    telem.Float32T,
					Leaseholder: node.KeyFree,
					Virtual:     true,
					LocalIndex:  idxCh.LocalKey,
				}
				Expect(mockNode.DB.WithTx(ctx, func(tx gorp.Tx) error {
					return channelSvc.NewWriter(tx).Create(ctx, &dataCh)
				})).To(Succeed())
				keys := channel.Keys{idxCh.Key(), dataCh.Key()}

				streamer := MustSucceed(
					mockNode.Framer.NewStreamer(framer.StreamerConfig{
						Keys:        keys,
						SendOpenAck: new(true),
					}),
				)
				_, out := confluence.Attach(streamer, 10)
				sCtx, cancel := signal.WithCancel(ctx)
				defer cancel()
				streamer.Flow(sCtx)
				var res framer.StreamerResponse
				Eventually(out.Outlet()).Should(Receive(&res))

				w := MustOpen(writerSvc.Open(ctx, writer.Config{
					Keys:  keys,
					Start: 10 * telem.SecondTS,
					Sync:  new(true),
				}))
				data := telem.NewSeriesV[float32](1, 2)
				idx := telem.NewSeriesSecondsTSV(10, 11)
				Expect(
					w.Write(frame.NewMulti(keys, []telem.Series{idx, data})),
				).To(BeTrue())
				Eventually(out.Outlet()).Should(Receive(&res))
				writtenData := res.Frame.Get(dataCh.Key()).Series[0]
				Expect(writtenData).To(telem.MatchSeriesData(data))
				writtenIdx := res.Frame.Get(idxCh.Key()).Series[0]
				Expect(writtenIdx).To(telem.MatchSeriesData(idx))
				firstAlignment := writtenData.Alignment
				groupDomain := firstAlignment.DomainIndex()
				Expect(groupDomain).To(BeNumerically(
					">=",
					MustSucceed(cesium.NewAlignmentCounter().Next()).DomainIndex(),
				))
				Expect(firstAlignment.SampleIndex()).To(BeEquivalentTo(0))
				Expect(writtenIdx.Alignment).To(Equal(firstAlignment))

				data = telem.NewSeriesV[float32](3, 4)
				idx = telem.NewSeriesSecondsTSV(12, 13)
				Expect(
					w.Write(frame.NewMulti(keys, []telem.Series{idx, data})),
				).To(BeTrue())
				Eventually(out.Outlet()).Should(Receive(&res))
				writtenData = res.Frame.Get(dataCh.Key()).Series[0]
				Expect(writtenData).To(telem.MatchSeriesData(data))
				writtenIdx = res.Frame.Get(idxCh.Key()).Series[0]
				Expect(writtenIdx).To(telem.MatchSeriesData(idx))
				Expect(writtenData.Alignment.DomainIndex()).To(Equal(groupDomain))
				Expect(writtenData.Alignment.SampleIndex()).To(BeEquivalentTo(2))
				Expect(writtenData.Alignment).To(BeNumerically(">", firstAlignment))
				Expect(writtenIdx.Alignment).To(Equal(writtenData.Alignment))
			},
		)
	})
})
