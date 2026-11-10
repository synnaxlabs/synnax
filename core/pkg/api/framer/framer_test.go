// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer_test

import (
	"fmt"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	freightertestutil "github.com/synnaxlabs/freighter/testutil"
	"github.com/synnaxlabs/gorp"
	apiframer "github.com/synnaxlabs/synnax/pkg/api/framer"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	svcframer "github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var errClientGone = errors.New("client gone")

type goneStream[RQ, RS freighter.Payload] struct{ request RQ }

func (s goneStream[RQ, RS]) Receive() (RQ, error) { return s.request, nil }

func (s goneStream[RQ, RS]) Send(RS) error { return errClientGone }

var _ = Describe("Framer", func() {
	Describe("Open acks", func() {
		var index channel.Channel
		BeforeEach(func(ctx SpecContext) {
			index = channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.TimestampT,
				IsIndex:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Channel.NewWriter(tx).Create(ctx, &index)
			})).To(Succeed())
		})

		It("Should release an iterator whose open ack fails", func(ctx SpecContext) {
			Expect(apiSvc.Iterate(rootCtx(ctx), goneStream[
				apiframer.IteratorRequest,
				apiframer.IteratorResponse,
			]{request: apiframer.IteratorRequest{
				Keys:   channel.Keys{index.Key()},
				Bounds: telem.TimeRangeMax,
			}})).To(MatchError(errClientGone))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Channel.NewWriter(tx).Delete(ctx, index.Key(), false)
			})).To(Succeed())
		})

		It("Should release a writer whose open ack fails", func(ctx SpecContext) {
			keys := channel.Keys{index.Key()}
			Expect(apiSvc.Write(rootCtx(ctx), goneStream[
				apiframer.WriterRequest,
				apiframer.WriterResponse,
			]{request: apiframer.WriterRequest{Config: apiframer.WriterConfig{
				ControlSubject: control.Subject{Key: "gone"},
				Keys:           keys,
				Start:          telem.SecondTS,
				Authorities:    []uint32{uint32(control.AuthorityAbsolute)},
			}}})).To(MatchError(errClientGone))
			w := MustSucceed(framerSvc.OpenWriter(ctx, svcframer.WriterConfig{
				Start:             telem.SecondTS,
				Keys:              keys,
				Authorities:       []control.Authority{control.AuthorityAbsolute},
				ErrOnUnauthorized: new(true),
			}))
			Expect(w.Close()).To(Succeed())
		})
	})

	createVirtual := func(ctx SpecContext) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Float32T,
			Virtual:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Channel.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}
	createCalculated := func(ctx SpecContext, input channel.Channel) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{
			Name:       UniqueChannelName(),
			DataType:   telem.Float32T,
			Expression: fmt.Sprintf("return %s * 2", input.Name),
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Channel.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}
	// readerCtx returns a context whose subject may retrieve only the given channels.
	readerCtx := func(
		ctx SpecContext,
		readable ...channel.Channel,
	) freighter.Context {
		GinkgoHelper()
		keys := channel.KeysFromChannels(readable)
		u := createUserGranted(
			ctx,
			access.ActionRetrieve,
			svcframer.ResourceIDs(keys)...,
		)
		return subjectCtx(ctx, u.ResourceID())
	}

	Describe("Iterate", func() {
		It(
			"Should deny a calculated channel whose input the subject cannot read",
			func(ctx SpecContext) {
				calc := createCalculated(ctx, createVirtual(ctx))
				client, server := freightertestutil.NewStreams[
					apiframer.IteratorRequest,
					apiframer.IteratorResponse,
				](ctx)
				DeferCleanup(client.CloseSend)
				done := make(chan error, 1)
				go func() { done <- apiSvc.Iterate(readerCtx(ctx, calc), server) }()
				Expect(client.Send(apiframer.IteratorRequest{
					Keys:   channel.Keys{calc.Key()},
					Bounds: telem.TimeRangeMax,
				})).To(Succeed())
				Eventually(done).WithTimeout(5 * time.Second).
					Should(Receive(MatchError(access.ErrDenied)))
			},
		)

		It("Should return the error of an iterator that fails", func(ctx SpecContext) {
			index := channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.TimestampT,
				IsIndex:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Channel.NewWriter(tx).Create(ctx, &index)
			})).To(Succeed())
			data := channel.Channel{
				Name:       UniqueChannelName(),
				DataType:   telem.Float32T,
				LocalIndex: index.LocalKey,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Channel.NewWriter(tx).Create(ctx, &data)
			})).To(Succeed())
			keys := channel.Keys{index.Key(), data.Key()}
			w := MustSucceed(framerSvc.OpenWriter(ctx, svcframer.WriterConfig{
				Start: telem.SecondTS,
				Keys:  keys,
			}))
			Expect(w.Write(frame.NewMulti(keys, []telem.Series{
				telem.NewSeriesSecondsTSV(1, 2, 3, 4),
				telem.NewSeriesV[float32](1, 2, 3, 4),
			}))).To(BeTrue())
			Expect(w.Close()).To(Succeed())
			calc := channel.Channel{
				Name:       UniqueChannelName(),
				DataType:   telem.Float32T,
				Expression: fmt.Sprintf("return %s * 2", data.Name),
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Channel.NewWriter(tx).Create(ctx, &calc)
			})).To(Succeed())
			client, server := freightertestutil.NewStreams[
				apiframer.IteratorRequest,
				apiframer.IteratorResponse,
			](ctx)
			DeferCleanup(client.CloseSend)
			done := make(chan error, 1)
			go func() { done <- apiSvc.Iterate(rootCtx(ctx), server) }()
			Expect(client.Send(apiframer.IteratorRequest{
				Keys:   channel.Keys{calc.Key()},
				Bounds: telem.TimeRangeMax,
				Reduction: telem.Reduction{Variant: telem.LimitReduction{
					Aggregation: telem.AggregationMinMax,
					PointLimit:  2,
				}},
			})).To(Succeed())
			Expect(client.Receive()).To(HaveField("Ack", BeTrue()))
			faultyFS.SetOptions(WithFailReadAt())
			DeferCleanup(func() { faultyFS.SetOptions() })
			// Setting bounds counts the samples in them, and the count fails.
			Expect(client.Send(apiframer.IteratorRequest{
				Command: apiframer.IteratorCommandSetBounds,
				Bounds:  telem.TimeRangeMax,
			})).To(Succeed())
			Eventually(done).WithTimeout(5 * time.Second).
				Should(Receive(MatchError(ContainSubstring(ErrFault.Error()))))
		})
	})

	Describe("Read", func() {
		var (
			index channel.Channel
			data  channel.Channel
		)
		BeforeEach(func(ctx SpecContext) {
			index = channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.TimestampT,
				IsIndex:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Channel.NewWriter(tx).Create(ctx, &index)
			})).To(Succeed())
			data = channel.Channel{
				Name:       UniqueChannelName(),
				DataType:   telem.Float32T,
				LocalIndex: index.LocalKey,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Channel.NewWriter(tx).Create(ctx, &data)
			})).To(Succeed())
		})

		It("Should open iterators over the requested channels", func(
			ctx SpecContext,
		) {
			res := MustSucceed(apiSvc.Read(rootCtx(ctx), apiframer.ReadRequest{
				Keys:   channel.Keys{data.Key()},
				Bounds: telem.TimeRangeMax,
			}))
			iter := MustSucceed(
				res.OpenIterator(ctx, channel.Keys{data.Key()}, res.Bounds),
			)
			Expect(iter.Close()).To(Succeed())
			Expect(res.Channels).To(HaveLen(1))
			Expect(res.Channels[0].Key()).To(Equal(data.Key()))
			Expect(res.Indexes).To(BeEmpty())
		})

		It("Should reject a read with both latest and earliest", func(
			ctx SpecContext,
		) {
			Expect(apiSvc.Read(rootCtx(ctx), apiframer.ReadRequest{
				Keys:     channel.Keys{data.Key()},
				Bounds:   telem.TimeRangeMax,
				Latest:   10,
				Earliest: 10,
			})).Error().To(MatchError(validate.ErrValidation))
		})

		It("Should reject a read with both earliest and a reduction", func(
			ctx SpecContext,
		) {
			Expect(apiSvc.Read(rootCtx(ctx), apiframer.ReadRequest{
				Keys:     channel.Keys{data.Key()},
				Bounds:   telem.TimeRangeMax,
				Earliest: 10,
				Reduction: telem.Reduction{
					Variant: telem.LimitReduction{PointLimit: 10},
				},
			})).Error().To(MatchError(validate.ErrValidation))
		})

		It("Should reject a read with both latest and a reduction", func(
			ctx SpecContext,
		) {
			Expect(apiSvc.Read(rootCtx(ctx), apiframer.ReadRequest{
				Keys:   channel.Keys{data.Key()},
				Bounds: telem.TimeRangeMax,
				Latest: 10,
				Reduction: telem.Reduction{
					Variant: telem.LimitReduction{PointLimit: 10},
				},
			})).Error().To(MatchError(validate.ErrValidation))
		})

		It(
			"Should pull in the index channels an indexes-included request left out",
			func(ctx SpecContext) {
				res := MustSucceed(apiSvc.Read(rootCtx(ctx), apiframer.ReadRequest{
					Keys:               channel.Keys{data.Key()},
					Bounds:             telem.TimeRangeMax,
					TimestampsIncluded: true,
				}))
				Expect(res.Indexes).To(HaveLen(1))
				Expect(res.Indexes[0].Key()).To(Equal(index.Key()))
			},
		)

		It("Should not repeat an index the request already asked for", func(
			ctx SpecContext,
		) {
			res := MustSucceed(apiSvc.Read(rootCtx(ctx), apiframer.ReadRequest{
				Keys:               channel.Keys{data.Key(), index.Key()},
				Bounds:             telem.TimeRangeMax,
				TimestampsIncluded: true,
			}))
			Expect(res.Channels).To(HaveLen(2))
			Expect(res.Indexes).To(BeEmpty())
		})

		It("Should open iterators under the request's reduction", func(
			ctx SpecContext,
		) {
			keys := channel.Keys{index.Key(), data.Key()}
			w := MustSucceed(framerSvc.OpenWriter(ctx, svcframer.WriterConfig{
				Start: telem.SecondTS,
				Keys:  keys,
			}))
			Expect(w.Write(frame.NewMulti(keys, []telem.Series{
				telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5, 6, 7, 8),
				telem.NewSeriesV[float32](1, 2, 3, 4, 5, 6, 7, 8),
			}))).To(BeTrue())
			Expect(w.Close()).To(Succeed())
			reduction := telem.Reduction{Variant: telem.LimitReduction{
				Aggregation: telem.AggregationDecimate,
				PointLimit:  4,
			}}
			res := MustSucceed(apiSvc.Read(rootCtx(ctx), apiframer.ReadRequest{
				Keys:      channel.Keys{data.Key()},
				Bounds:    telem.TimeRangeMax,
				Reduction: reduction,
			}))
			Expect(res.Reduction).To(Equal(reduction))
			iter := MustSucceed(
				res.OpenIterator(ctx, channel.Keys{data.Key()}, res.Bounds),
			)
			Expect(iter.SeekFirst()).To(BeTrue())
			Expect(iter.Next(apiframer.IteratorAutoSpan)).To(BeTrue())
			series := iter.Value().Get(data.Key()).Series
			Expect(iter.Close()).To(Succeed())
			Expect(series).To(HaveLen(1))
			Expect(series[0].AlignmentMultiple).To(BeEquivalentTo(2))
			Expect(series[0]).To(telem.MatchSeriesDataV[float32](1, 3, 5, 7))
		})

		It("Should return an error when a key has no channel", func(ctx SpecContext) {
			Expect(apiSvc.Read(rootCtx(ctx), apiframer.ReadRequest{
				Keys:   channel.Keys{distchannel.NewKey(index.Leaseholder, 9999)},
				Bounds: telem.TimeRangeMax,
			})).Error().To(MatchError(query.ErrNotFound))
		})

		Describe("Access control", func() {
			It("Should deny a subject that cannot retrieve the pulled-in index", func(
				ctx SpecContext,
			) {
				u := createUserGranted(
					ctx,
					access.ActionRetrieve,
					svcframer.ResourceIDs(channel.Keys{data.Key()})...,
				)
				Expect(apiSvc.Read(
					subjectCtx(ctx, u.ResourceID()),
					apiframer.ReadRequest{
						Keys:               channel.Keys{data.Key()},
						Bounds:             telem.TimeRangeMax,
						TimestampsIncluded: true,
					},
				)).Error().To(MatchError(access.ErrDenied))
			})

			It(
				"Should allow a subject granted only the channel when indexes are left out",
				func(ctx SpecContext) {
					u := createUserGranted(
						ctx,
						access.ActionRetrieve,
						svcframer.ResourceIDs(channel.Keys{data.Key()})...,
					)
					res := MustSucceed(apiSvc.Read(
						subjectCtx(ctx, u.ResourceID()),
						apiframer.ReadRequest{
							Keys:   channel.Keys{data.Key()},
							Bounds: telem.TimeRangeMax,
						},
					))
					Expect(res.Indexes).To(BeEmpty())
				},
			)

			It("Should allow a subject granted both the channel and its index", func(
				ctx SpecContext,
			) {
				u := createUserGranted(
					ctx,
					access.ActionRetrieve,
					svcframer.ResourceIDs(channel.Keys{data.Key(), index.Key()})...,
				)
				res := MustSucceed(apiSvc.Read(
					subjectCtx(ctx, u.ResourceID()),
					apiframer.ReadRequest{
						Keys:               channel.Keys{data.Key()},
						Bounds:             telem.TimeRangeMax,
						TimestampsIncluded: true,
					},
				))
				Expect(res.Indexes).To(HaveLen(1))
			})

			It(
				"Should deny a calculated channel whose input the subject cannot read",
				func(ctx SpecContext) {
					calc := createCalculated(ctx, data)
					Expect(apiSvc.Read(readerCtx(ctx, calc), apiframer.ReadRequest{
						Keys:   channel.Keys{calc.Key()},
						Bounds: telem.TimeRangeMax,
					})).Error().To(MatchError(access.ErrDenied))
				},
			)
		})
	})

	Describe("Write", func() {
		newStreams := func(ctx SpecContext) (
			freighter.ClientStream[apiframer.WriterRequest, apiframer.WriterResponse],
			apiframer.WriterStream,
		) {
			return freightertestutil.NewStreams[
				apiframer.WriterRequest,
				apiframer.WriterResponse,
			](ctx)
		}

		It("Should refuse a writer on a change channel", func(ctx SpecContext) {
			var ch channel.Channel
			Expect(channelSvc.NewRetrieve().
				Where(channel.MatchNames("sy_range_set")).
				Entry(&ch).
				Exec(ctx, db)).To(Succeed())
			client, server := newStreams(ctx)
			Expect(client.Send(apiframer.WriterRequest{
				Config: apiframer.WriterConfig{Keys: channel.Keys{ch.Key()}},
			})).To(Succeed())
			Expect(apiSvc.Write(rootCtx(ctx), server)).To(MatchError(fmt.Sprintf(
				"channel %d holds changes that only the Core writes: validation error",
				ch.Key(),
			)))
		})

		It("Should open a writer on a channel that holds no changes", func(
			ctx SpecContext,
		) {
			c := channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Float32T,
				Virtual:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Channel.NewWriter(tx).Create(ctx, &c)
			})).To(Succeed())
			client, server := newStreams(ctx)
			done := make(chan error, 1)
			go func() { done <- apiSvc.Write(rootCtx(ctx), server) }()
			Expect(client.Send(apiframer.WriterRequest{
				Config: apiframer.WriterConfig{Keys: channel.Keys{c.Key()}},
			})).To(Succeed())
			Expect(client.Receive()).
				To(HaveField("Command", svcframer.WriterCommandOpen))
			Expect(client.CloseSend()).To(Succeed())
			Eventually(done).WithTimeout(5 * time.Second).Should(Receive(BeNil()))
		})
	})

	Describe("Delete", func() {
		It("Should delete data by exact name and not read a name as a pattern", func(
			ctx SpecContext,
		) {
			index := channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.TimestampT,
				IsIndex:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Channel.NewWriter(tx).Create(ctx, &index)
			})).To(Succeed())
			data := channel.Channel{
				Name:       UniqueChannelName(),
				DataType:   telem.Float32T,
				LocalIndex: index.LocalKey,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Channel.NewWriter(tx).Create(ctx, &data)
			})).To(Succeed())
			keys := channel.Keys{index.Key(), data.Key()}
			w := MustSucceed(framerSvc.OpenWriter(ctx, svcframer.WriterConfig{
				Start: telem.SecondTS,
				Keys:  keys,
			}))
			Expect(w.Write(frame.NewMulti(keys, []telem.Series{
				telem.NewSeriesSecondsTSV(1, 2, 3),
				telem.NewSeriesV[float32](1, 2, 3),
			}))).To(BeTrue())
			Expect(w.Close()).To(Succeed())
			hasData := func() bool {
				GinkgoHelper()
				iter := MustOpen(framerSvc.OpenIterator(ctx, svcframer.IteratorConfig{
					Keys:   keys,
					Bounds: telem.TimeRangeMax,
				}))
				return iter.SeekFirst()
			}
			Expect(apiSvc.Delete(rootCtx(ctx), db, apiframer.DeleteRequest{
				Names:  []string{index.Name + ".*", data.Name + ".*"},
				Bounds: telem.TimeRangeMax,
			})).Error().ToNot(HaveOccurred())
			Expect(hasData()).To(BeTrue())
			Expect(apiSvc.Delete(rootCtx(ctx), db, apiframer.DeleteRequest{
				Names:  []string{index.Name, data.Name},
				Bounds: telem.TimeRangeMax,
			})).Error().ToNot(HaveOccurred())
			Expect(hasData()).To(BeFalse())
		})
	})

	Describe("Stream", func() {
		var (
			readable = resource.ID{Type: resource.TypeTask, Key: "readable"}
			denied   = resource.ID{Type: resource.TypeTask, Key: "denied"}
			records  channel.Channel
			other    channel.Channel
			writer   *svcframer.Writer
			client   freighter.ClientStream[
				apiframer.StreamerRequest,
				apiframer.StreamerResponse,
			]
		)
		samples := func(ids ...resource.ID) telem.Series {
			series := telem.Series{DataType: telem.JSONT}
			for _, id := range ids {
				series.Data = append(
					series.Data,
					telem.MarshalVariableSample([]byte(strconv.Quote(id.String())))...,
				)
			}
			return series
		}
		received := func() []string {
			GinkgoHelper()
			res := MustSucceed(client.Receive())
			var out []string
			for _, series := range res.Frame.Get(records.Key()).Series {
				for sample := range series.Samples() {
					out = append(out, MustSucceed(strconv.Unquote(string(sample))))
				}
			}
			return out
		}
		// open runs in each spec, since a BeforeEach's context ends with the
		// BeforeEach.
		open := func(ctx SpecContext) {
			GinkgoHelper()
			records = channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.JSONT,
				Virtual:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Channel.NewWriter(tx).Create(ctx, &records)
			})).To(Succeed())
			other = channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Float32T,
				Virtual:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Channel.NewWriter(tx).Create(ctx, &other)
			})).To(Succeed())
			keys := channel.Keys{records.Key(), other.Key()}
			svc := MustSucceed(apiframer.NewService(apiframer.ServiceConfig{
				DB:      db,
				Framer:  framerSvc,
				Channel: channelSvc,
				RBAC:    rbacSvc,
				Records: map[channel.Key]signals.Record{
					records.Key(): {Of: func(sample []byte) (resource.ID, error) {
						id, err := strconv.Unquote(string(sample))
						if err != nil {
							return resource.ID{}, err
						}
						return resource.ParseID(id)
					}},
				},
			}))
			u := createUserGranted(
				ctx,
				access.ActionRetrieve,
				append(svcframer.ResourceIDs(keys), readable)...,
			)
			var server apiframer.StreamerStream
			client, server = freightertestutil.NewStreams[
				apiframer.StreamerRequest,
				apiframer.StreamerResponse,
			](ctx)
			done := make(chan error, 1)
			go func() { done <- svc.Stream(subjectCtx(ctx, u.ResourceID()), server) }()
			DeferCleanup(func() {
				Expect(client.CloseSend()).To(Succeed())
				Eventually(done).WithTimeout(5 * time.Second).Should(Receive())
			})
			Expect(client.Send(apiframer.StreamerRequest{Keys: keys})).To(Succeed())
			Expect(MustSucceed(client.Receive()).Frame.Empty()).To(BeTrue())
			writer = MustSucceed(framerSvc.OpenWriter(ctx, svcframer.WriterConfig{
				Start: telem.SecondTS,
				Keys:  keys,
			}))
			DeferCleanup(writer.Close)
		}

		It(
			"Should send only the samples that the subject can read",
			func(ctx SpecContext) {
				open(ctx)
				Expect(writer.Write(frame.NewUnary(
					records.Key(), samples(denied, readable, denied),
				))).To(BeTrue())
				Expect(received()).To(Equal([]string{readable.String()}))
			},
		)

		It("Should send no frame when the subject can read none of its samples", func(
			ctx SpecContext,
		) {
			open(ctx)
			Expect(writer.Write(frame.NewUnary(
				records.Key(), samples(denied),
			))).To(BeTrue())
			Expect(writer.Write(frame.NewUnary(
				records.Key(), samples(readable),
			))).To(BeTrue())
			Expect(received()).To(Equal([]string{readable.String()}))
		})

		It("Should keep the other channels of a frame", func(ctx SpecContext) {
			open(ctx)
			Expect(writer.Write(frame.NewMulti(
				channel.Keys{records.Key(), other.Key()},
				[]telem.Series{samples(denied), telem.NewSeriesV[float32](1)},
			))).To(BeTrue())
			res := MustSucceed(client.Receive())
			Expect(res.Frame.Get(records.Key()).Series).To(BeEmpty())
			Expect(res.Frame.Get(other.Key()).Series).To(HaveLen(1))
		})

		It(
			"Should leave a frame with no record channel as it is",
			func(ctx SpecContext) {
				open(ctx)
				Expect(writer.Write(frame.NewUnary(
					other.Key(), telem.NewSeriesV[float32](1, 2),
				))).To(BeTrue())
				res := MustSucceed(client.Receive())
				Expect(res.Frame.Get(other.Key()).Series).To(HaveLen(1))
			},
		)

		Describe("Access control", func() {
			type streamClient = freighter.ClientStream[
				apiframer.StreamerRequest,
				apiframer.StreamerResponse,
			]
			// stream opens a stream on keys, and returns its client and a channel that
			// receives the error Stream returns.
			stream := func(
				ctx freighter.Context,
				keys channel.Keys,
			) (streamClient, <-chan error) {
				GinkgoHelper()
				client, server := freightertestutil.NewStreams[
					apiframer.StreamerRequest,
					apiframer.StreamerResponse,
				](ctx)
				done := make(chan error, 1)
				stopped := make(chan struct{})
				go func() {
					defer close(stopped)
					done <- apiSvc.Stream(ctx, server)
				}()
				DeferCleanup(func() {
					Expect(client.CloseSend()).To(Succeed())
					Eventually(stopped).WithTimeout(5 * time.Second).Should(BeClosed())
				})
				Expect(client.Send(apiframer.StreamerRequest{Keys: keys})).To(Succeed())
				return client, done
			}

			It(
				"Should deny a calculated channel whose input the subject cannot read",
				func(ctx SpecContext) {
					calc := createCalculated(ctx, createVirtual(ctx))
					_, done := stream(readerCtx(ctx, calc), channel.Keys{calc.Key()})
					Eventually(done).WithTimeout(5 * time.Second).
						Should(Receive(MatchError(access.ErrDenied)))
				},
			)

			It(
				"Should end the stream on a later request for an unreadable channel",
				func(ctx SpecContext) {
					readable, secret := createVirtual(ctx), createVirtual(ctx)
					streamer, done := stream(
						readerCtx(ctx, readable),
						channel.Keys{readable.Key()},
					)
					Expect(
						MustSucceed(streamer.Receive()).Frame.Empty(),
					).To(BeTrue())
					Expect(streamer.Send(apiframer.StreamerRequest{
						Keys: channel.Keys{readable.Key(), secret.Key()},
					})).To(Succeed())
					Eventually(done).WithTimeout(5 * time.Second).
						Should(Receive(MatchError(access.ErrDenied)))
				},
			)

			It(
				"Should apply a later request that adds a channel the subject can read",
				func(ctx SpecContext) {
					first, second := createVirtual(ctx), createVirtual(ctx)
					streamer, _ := stream(
						readerCtx(ctx, first, second),
						channel.Keys{first.Key()},
					)
					Expect(
						MustSucceed(streamer.Receive()).Frame.Empty(),
					).To(BeTrue())
					Expect(streamer.Send(apiframer.StreamerRequest{
						Keys:         channel.Keys{first.Key(), second.Key()},
						AckRequested: true,
					})).To(Succeed())
					Expect(MustSucceed(streamer.Receive()).Ack).To(BeTrue())
					w := MustOpen(framerSvc.OpenWriter(ctx, svcframer.WriterConfig{
						Start: telem.SecondTS,
						Keys:  channel.Keys{second.Key()},
					}))
					Expect(w.Write(frame.NewUnary(
						second.Key(), telem.NewSeriesV[float32](1),
					))).To(BeTrue())
					res := MustSucceed(streamer.Receive())
					Expect(res.Frame.Get(second.Key()).Series).To(HaveLen(1))
				},
			)
		})
	})
})
