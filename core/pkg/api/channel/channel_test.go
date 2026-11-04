// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func createChannel(ctx SpecContext, dt telem.DataType) channel.Channel {
	GinkgoHelper()
	ch := channel.Channel{
		Name:     UniqueChannelName(),
		DataType: dt,
		Virtual:  true,
	}
	Expect(channelSvc.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
	return ch
}

var _ = Describe("Service", func() {
	Describe("RetrieveDataTypes", func() {
		It(
			"Should resolve the data types of the channels in key order",
			func(ctx SpecContext) {
				ch1 := createChannel(ctx, telem.Float32T)
				ch2 := createChannel(ctx, telem.Int64T)
				ch3 := createChannel(ctx, telem.Uint8T)
				Expect(apiSvc.RetrieveDataTypes(
					ctx, channel.Keys{ch1.Key(), ch2.Key(), ch3.Key()},
				)).To(Equal([]telem.DataType{telem.Float32T, telem.Int64T, telem.Uint8T}))
			},
		)

		It("Should resolve the data type of a single channel", func(ctx SpecContext) {
			ch := createChannel(ctx, telem.Float64T)
			Expect(apiSvc.RetrieveDataTypes(ctx, channel.Keys{ch.Key()})).
				To(Equal([]telem.DataType{telem.Float64T}))
		})

		It(
			"Should return an empty slice when no keys are provided",
			func(ctx SpecContext) {
				Expect(apiSvc.RetrieveDataTypes(ctx, channel.Keys{})).To(BeEmpty())
			},
		)

		It(
			"Should return a not found error when a key does not resolve to a channel",
			func(ctx SpecContext) {
				ch := createChannel(ctx, telem.Int32T)
				Expect(apiSvc.RetrieveDataTypes(
					ctx, channel.Keys{ch.Key(), channel.Key(999999)},
				)).Error().To(MatchError(query.ErrNotFound))
			},
		)
	})

	Describe("RetrieveName", func() {
		It("Should resolve the name of an existing channel", func(ctx SpecContext) {
			ch := createChannel(ctx, telem.Float32T)
			Expect(apiSvc.RetrieveName(ctx, ch.Key())).To(Equal(ch.Name))
		})

		It(
			"Should return an empty string when no channel has the key",
			func(ctx SpecContext) {
				Expect(apiSvc.RetrieveName(ctx, channel.Key(999999))).To(BeEmpty())
			},
		)
	})

	Describe("Retrieve", func() {
		Describe("IncludeStatus", func() {
			BeforeEach(func(ctx SpecContext) {
				grantOn(
					ctx,
					author.ResourceID(),
					[]access.Action{access.ActionRetrieve},
					resource.ID{Type: resource.TypeChannel},
				)
			})

			setStatus := func(ctx SpecContext, ch channel.Channel) {
				GinkgoHelper()
				Expect(statusSvc.NewWriter(nil).Set(ctx, &apichannel.Status{
					Key:     channel.ResourceID(ch.Key()).String(),
					Name:    ch.Name,
					Variant: status.VariantSuccess,
					Message: "ok",
					Time:    telem.Now(),
				})).To(Succeed())
			}

			retrieve := func(ctx SpecContext, keys ...channel.Key) []apichannel.Channel {
				GinkgoHelper()
				return MustSucceed(apiSvc.Retrieve(
					AuthedCtx(ctx, author),
					apichannel.RetrieveRequest{Keys: keys, IncludeStatus: true},
				)).Channels
			}

			It(
				"Should attach the status of each channel that has one",
				func(ctx SpecContext) {
					ch := createChannel(ctx, telem.Float32T)
					setStatus(ctx, ch)
					res := retrieve(ctx, ch.Key())
					Expect(res).To(HaveLen(1))
					Expect(res[0].Status).ToNot(BeNil())
					Expect(res[0].Status.Message).To(Equal("ok"))
				},
			)

			It(
				"Should leave the status empty for a channel that has none",
				func(ctx SpecContext) {
					withStatus := createChannel(ctx, telem.Float32T)
					setStatus(ctx, withStatus)
					withoutStatus := createChannel(ctx, telem.Int64T)
					res := retrieve(ctx, withStatus.Key(), withoutStatus.Key())
					Expect(res).To(HaveLen(2))
					Expect(res[0].Status).ToNot(BeNil())
					Expect(res[1].Status).To(BeNil())
				},
			)
		})

		It("Should reject a negative offset", func(ctx SpecContext) {
			Expect(apiSvc.Retrieve(
				AuthedCtx(ctx, author),
				apichannel.RetrieveRequest{Offset: -1},
			)).Error().To(MatchError("offset: must be greater than or equal to 0"))
		})

		Describe("SearchTerm", func() {
			const term = "zqsearch"
			var rng ranger.Range
			BeforeEach(func(ctx SpecContext) {
				grantOn(
					ctx,
					author.ResourceID(),
					[]access.Action{access.ActionRetrieve},
					resource.ID{Type: resource.TypeChannel},
				)
				rng = ranger.Range{Name: term, TimeRange: telem.NewRangeSeconds(1, 5)}
				Expect(rangerSvc.NewWriter(nil).Create(ctx, &rng)).To(Succeed())
				DeferCleanup(func(ctx SpecContext) {
					Expect(rangerSvc.NewWriter(nil).Delete(ctx, rng.Key)).To(Succeed())
				})
			})

			named := func(ctx SpecContext, name string) channel.Channel {
				GinkgoHelper()
				ch := channel.Channel{
					Name:     name,
					DataType: telem.Float32T,
					Virtual:  true,
				}
				Expect(channelSvc.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
				DeferCleanup(func(ctx SpecContext) {
					Expect(channelSvc.NewWriter(nil).Delete(ctx, ch.Key(), false)).
						To(Succeed())
				})
				return ch
			}

			aliased := func(ctx SpecContext, name string) channel.Channel {
				GinkgoHelper()
				ch := named(ctx, name)
				Expect(
					aliasSvc.NewWriter(nil).Set(ctx, rng.Key, ch.Key(), term+" feed"),
				).
					To(Succeed())
				DeferCleanup(func(ctx SpecContext) {
					Expect(aliasSvc.NewWriter(nil).Delete(ctx, rng.Key, ch.Key())).
						To(Succeed())
				})
				return ch
			}

			search := func(ctx SpecContext, req apichannel.RetrieveRequest) []channel.Key {
				GinkgoHelper()
				req.SearchTerm = term
				req.RangeKey = rng.Key
				res := MustSucceed(apiSvc.Retrieve(AuthedCtx(ctx, author), req))
				return lo.Map(
					res.Channels,
					func(ch apichannel.Channel, _ int) channel.Key {
						return ch.Key
					},
				)
			}

			It(
				"Should rank alias matches first and page through every match",
				func(ctx SpecContext) {
					byAlias := aliased(ctx, UniqueChannelName())
					inlet := named(ctx, term+"_inlet")
					outlet := named(ctx, term+"_outlet")
					first := search(ctx, apichannel.RetrieveRequest{Limit: 2})
					Expect(first).To(HaveLen(2))
					Expect(first[0]).To(Equal(byAlias.Key()))
					Expect(
						search(ctx, apichannel.RetrieveRequest{Limit: 2, Offset: 1}),
					).
						To(ConsistOf(inlet.Key(), outlet.Key()))
					Expect(
						search(ctx, apichannel.RetrieveRequest{Limit: 2, Offset: 2}),
					).
						To(HaveLen(1))
				},
			)

			It(
				"Should apply the other filters to alias matches",
				func(ctx SpecContext) {
					aliased(ctx, UniqueChannelName())
					Expect(search(ctx, apichannel.RetrieveRequest{
						DataTypes: []telem.DataType{telem.Int64T},
					})).To(BeEmpty())
				},
			)

			It(
				"Should return a channel whose alias and name both match once",
				func(ctx SpecContext) {
					ch := aliased(ctx, term+"_inlet")
					Expect(search(ctx, apichannel.RetrieveRequest{})).
						To(Equal([]channel.Key{ch.Key()}))
				},
			)
		})
	})
})
