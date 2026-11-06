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
	apiframer "github.com/synnaxlabs/synnax/pkg/api/framer"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	svcframer "github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
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
			Expect(channelWriter.Create(ctx, &index)).To(Succeed())
		})

		It("Should release an iterator whose open ack fails", func(ctx SpecContext) {
			Expect(apiSvc.Iterate(rootCtx(ctx), goneStream[
				apiframer.IteratorRequest,
				apiframer.IteratorResponse,
			]{request: apiframer.IteratorRequest{
				Keys:   channel.Keys{index.Key()},
				Bounds: telem.TimeRangeMax,
			}})).To(MatchError(errClientGone))
			Expect(channelWriter.Delete(ctx, index.Key(), false)).To(Succeed())
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

	Describe("Iterate", func() {
		It("Should return the error of an iterator that fails", func(ctx SpecContext) {
			index := channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.TimestampT,
				IsIndex:  true,
			}
			Expect(channelWriter.Create(ctx, &index)).To(Succeed())
			data := channel.Channel{
				Name:       UniqueChannelName(),
				DataType:   telem.Float32T,
				LocalIndex: index.LocalKey,
			}
			Expect(channelWriter.Create(ctx, &data)).To(Succeed())
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
			Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
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
			Expect(channelWriter.Create(ctx, &index)).To(Succeed())
			data = channel.Channel{
				Name:       UniqueChannelName(),
				DataType:   telem.Float32T,
				LocalIndex: index.LocalKey,
			}
			Expect(channelWriter.Create(ctx, &data)).To(Succeed())
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
					svcframer.OntologyIDs(channel.Keys{data.Key()})...,
				)
				Expect(apiSvc.Read(
					subjectCtx(ctx, u.OntologyID()),
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
						svcframer.OntologyIDs(channel.Keys{data.Key()})...,
					)
					res := MustSucceed(apiSvc.Read(
						subjectCtx(ctx, u.OntologyID()),
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
					svcframer.OntologyIDs(channel.Keys{data.Key(), index.Key()})...,
				)
				res := MustSucceed(apiSvc.Read(
					subjectCtx(ctx, u.OntologyID()),
					apiframer.ReadRequest{
						Keys:               channel.Keys{data.Key()},
						Bounds:             telem.TimeRangeMax,
						TimestampsIncluded: true,
					},
				))
				Expect(res.Indexes).To(HaveLen(1))
			})
		})
	})

	Describe("Stream", func() {
		var (
			readable = ontology.ID{Type: ontology.ResourceTypeTask, Key: "readable"}
			denied   = ontology.ID{Type: ontology.ResourceTypeTask, Key: "denied"}
			records  channel.Channel
			other    channel.Channel
			writer   *svcframer.Writer
			client   freighter.ClientStream[
				apiframer.StreamerRequest,
				apiframer.StreamerResponse,
			]
		)
		samples := func(ids ...ontology.ID) telem.Series {
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
			Expect(channelWriter.Create(ctx, &records)).To(Succeed())
			other = channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Float32T,
				Virtual:  true,
			}
			Expect(channelWriter.Create(ctx, &other)).To(Succeed())
			keys := channel.Keys{records.Key(), other.Key()}
			svc := MustSucceed(apiframer.NewService(apiframer.ServiceConfig{
				Framer:  framerSvc,
				Channel: channelSvc,
				RBAC:    rbacSvc,
				Records: map[channel.Key]apiframer.RecordOf{
					records.Key(): func(sample []byte) (ontology.ID, error) {
						id, err := strconv.Unquote(string(sample))
						if err != nil {
							return ontology.ID{}, err
						}
						return ontology.ParseID(id)
					},
				},
			}))
			u := createUserGranted(
				ctx,
				access.ActionRetrieve,
				append(svcframer.OntologyIDs(keys), readable)...,
			)
			var server apiframer.StreamerStream
			client, server = freightertestutil.NewStreams[
				apiframer.StreamerRequest,
				apiframer.StreamerResponse,
			](ctx)
			done := make(chan error, 1)
			go func() { done <- svc.Stream(subjectCtx(ctx, u.OntologyID()), server) }()
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
	})
})
