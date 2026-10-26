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
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
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
					author.OntologyID(),
					[]access.Action{access.ActionRetrieve},
					ontology.ID{Type: ontology.ResourceTypeChannel},
				)
			})

			setStatus := func(ctx SpecContext, ch channel.Channel) {
				GinkgoHelper()
				Expect(statusSvc.NewWriter(nil).Set(ctx, &apichannel.Status{
					Key:     channel.OntologyID(ch.Key()).String(),
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
					apichannel.RetrieveRequest{
						Where: &apichannel.Where{
							Key: &filter.Set[channel.Key]{In: keys},
						},
						IncludeStatus: new(true),
					},
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

		It("Should treat a missing key as a non-match", func(ctx SpecContext) {
			grantOn(
				ctx,
				author.OntologyID(),
				[]access.Action{access.ActionRetrieve},
				ontology.ID{Type: ontology.ResourceTypeChannel},
			)
			ch := createChannel(ctx, telem.Float32T)
			res := MustSucceed(apiSvc.Retrieve(
				AuthedCtx(ctx, author),
				apichannel.RetrieveRequest{Where: &apichannel.Where{
					Key: &filter.Set[channel.Key]{In: []channel.Key{ch.Key(), 999999}},
				}},
			))
			Expect(res.Channels).To(HaveLen(1))
			Expect(res.Channels[0].Key).To(Equal(ch.Key()))
		})

		It("Should return a path error for an invalid regex", func(ctx SpecContext) {
			Expect(apiSvc.Retrieve(
				AuthedCtx(ctx, author),
				apichannel.RetrieveRequest{Where: &apichannel.Where{
					Name: &filter.String{Regex: new("(a")},
				}},
			)).Error().To(SatisfyAll(
				BeAssignableToTypeOf(validate.PathError{}),
				MatchError(
					"where.name.regex: error parsing regexp: missing closing ): `(a`: validation error",
				),
			))
		})
	})
})
