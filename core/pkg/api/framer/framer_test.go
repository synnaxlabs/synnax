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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiframer "github.com/synnaxlabs/synnax/pkg/api/framer"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	svcframer "github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Framer", func() {
	Describe("Read", func() {
		var (
			index channel.Channel
			data  channel.Channel
		)
		BeforeEach(func(ctx SpecContext) {
			index = channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.TimeStampT,
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

		It("Should open an iterator over the requested channels", func(
			ctx SpecContext,
		) {
			res := MustSucceed(apiSvc.Read(rootCtx(ctx), apiframer.ReadRequest{
				Keys:   channel.Keys{data.Key()},
				Bounds: telem.TimeRangeMax,
			}))
			DeferCleanup(func() { Expect(res.Iterator.Close()).To(Succeed()) })
			Expect(res.Channels).To(HaveLen(1))
			Expect(res.Channels[0].Key()).To(Equal(data.Key()))
		})

		It("Should pull in the index channels the request left out", func(
			ctx SpecContext,
		) {
			res := MustSucceed(apiSvc.Read(rootCtx(ctx), apiframer.ReadRequest{
				Keys:   channel.Keys{data.Key()},
				Bounds: telem.TimeRangeMax,
			}))
			DeferCleanup(func() { Expect(res.Iterator.Close()).To(Succeed()) })
			Expect(res.Indexes).To(HaveLen(1))
			Expect(res.Indexes[0].Key()).To(Equal(index.Key()))
		})

		It("Should not repeat an index the request already asked for", func(
			ctx SpecContext,
		) {
			res := MustSucceed(apiSvc.Read(rootCtx(ctx), apiframer.ReadRequest{
				Keys:   channel.Keys{data.Key(), index.Key()},
				Bounds: telem.TimeRangeMax,
			}))
			DeferCleanup(func() { Expect(res.Iterator.Close()).To(Succeed()) })
			Expect(res.Channels).To(HaveLen(2))
			Expect(res.Indexes).To(BeEmpty())
		})

		It("Should return an error when a key has no channel", func(ctx SpecContext) {
			Expect(apiSvc.Read(rootCtx(ctx), apiframer.ReadRequest{
				Keys:   channel.Keys{channel.NewKey(index.Leaseholder, 9999)},
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
						Keys:   channel.Keys{data.Key()},
						Bounds: telem.TimeRangeMax,
					},
				)).Error().To(MatchError(access.ErrDenied))
			})

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
						Keys:   channel.Keys{data.Key()},
						Bounds: telem.TimeRangeMax,
					},
				))
				DeferCleanup(func() { Expect(res.Iterator.Close()).To(Succeed()) })
				Expect(res.Indexes).To(HaveLen(1))
			})
		})
	})
})
