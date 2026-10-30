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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	freightertestutil "github.com/synnaxlabs/freighter/testutil"
	apiframer "github.com/synnaxlabs/synnax/pkg/api/framer"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	svcframer "github.com/synnaxlabs/synnax/pkg/service/framer"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Framer", func() {
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
			iter := MustSucceed(res.OpenIterator(ctx, channel.Keys{data.Key()}))
			Expect(iter.Close()).To(Succeed())
			Expect(res.Channels).To(HaveLen(1))
			Expect(res.Channels[0].Key()).To(Equal(data.Key()))
			Expect(res.Indexes).To(BeEmpty())
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
			iter := MustSucceed(res.OpenIterator(ctx, channel.Keys{data.Key()}))
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
})
