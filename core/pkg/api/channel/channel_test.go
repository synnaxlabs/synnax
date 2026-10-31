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
	"encoding/json/v2"
	"fmt"
	"math/rand/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
)

func createChannel(ctx SpecContext, dt telem.DataType) channel.Channel {
	GinkgoHelper()
	return createNamedChannel(ctx, UniqueChannelName(), dt)
}

func createNamedChannel(
	ctx SpecContext,
	name string,
	dt telem.DataType,
) channel.Channel {
	GinkgoHelper()
	ch := channel.Channel{Name: name, DataType: dt, Virtual: true}
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
					apichannel.AliasedRetrieveRequest{
						Request: apichannel.RetrieveRequest{
							Where: &channel.Where{
								Key: &filter.Set[channel.Key]{In: keys},
							},
							IncludeStatus: true,
						},
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

		Describe("RangeKey", func() {
			var (
				rng     ranger.Range
				term    string
				byName  channel.Channel
				byAlias channel.Channel
				both    *channel.Where
			)
			BeforeEach(func(ctx SpecContext) {
				grantOn(
					ctx,
					author.OntologyID(),
					[]access.Action{access.ActionRetrieve},
					ontology.ID{Type: ontology.ResourceTypeChannel},
					ontology.ID{Type: ontology.ResourceTypeRangeAlias},
				)
				rng = ranger.Range{
					Name: "range",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(5 * telem.Second),
						End:   telem.TimeStamp(10 * telem.Second),
					},
				}
				Expect(rangeSvc.NewWriter(nil).Create(ctx, &rng)).To(Succeed())
				term = fmt.Sprintf("probe%09d", rand.IntN(999999999))
				byName = createNamedChannel(ctx, term, telem.Float32T)
				byAlias = createChannel(ctx, telem.Float32T)
				Expect(aliasSvc.NewWriter(nil).Set(
					ctx, rng.Key, byAlias.Key(), term+" inlet",
				)).To(Succeed())
				both = &channel.Where{Key: &filter.Set[channel.Key]{
					In: []channel.Key{byName.Key(), byAlias.Key()},
				}}
			})

			retrieve := func(ctx SpecContext, req apichannel.RetrieveRequest) []channel.Key {
				GinkgoHelper()
				res := MustSucceed(apiSvc.Retrieve(
					AuthedCtx(ctx, author),
					apichannel.AliasedRetrieveRequest{Request: req, RangeKey: &rng.Key},
				))
				return lo.Map(
					res.Channels,
					func(ch apichannel.Channel, _ int) channel.Key {
						return ch.Key
					},
				)
			}

			It(
				"Should fill the alias of each channel on the range",
				func(ctx SpecContext) {
					res := MustSucceed(apiSvc.Retrieve(
						AuthedCtx(ctx, author),
						apichannel.AliasedRetrieveRequest{
							Request:  apichannel.RetrieveRequest{Where: both},
							RangeKey: &rng.Key,
						},
					))
					Expect(res.Channels).To(ConsistOf(
						SatisfyAll(
							HaveField("Key", byAlias.Key()),
							HaveField("Alias", HaveValue(Equal(term+" inlet"))),
						),
						SatisfyAll(
							HaveField("Key", byName.Key()),
							HaveField("Alias", BeNil()),
						),
					))
				},
			)

			It("Should rank the channels whose alias matches the term first", func(
				ctx SpecContext,
			) {
				Expect(retrieve(ctx, apichannel.RetrieveRequest{
					Where:      both,
					SearchTerm: term,
				})).To(Equal([]channel.Key{byAlias.Key(), byName.Key()}))
			})

			It("Should page the merged list of alias and name matches", func(
				ctx SpecContext,
			) {
				Expect(retrieve(ctx, apichannel.RetrieveRequest{
					Where:      both,
					SearchTerm: term,
					Limit:      new(uint32(1)),
					Offset:     1,
				})).To(Equal([]channel.Key{byName.Key()}))
			})

			It("Should return no channel for a zero limit", func(ctx SpecContext) {
				Expect(retrieve(ctx, apichannel.RetrieveRequest{
					Where:      both,
					SearchTerm: term,
					Limit:      new(uint32(0)),
				})).To(BeEmpty())
			})

			It("Should require retrieve access on the aliases it fills", func(
				ctx SpecContext,
			) {
				reader := MustSucceed(userSvc.NewWriter(nil).Create(
					ctx,
					user.User{Name: "reader-" + uuid.New().String()},
				))
				grantOn(
					ctx,
					reader.OntologyID(),
					[]access.Action{access.ActionRetrieve},
					ontology.ID{Type: ontology.ResourceTypeChannel},
				)
				Expect(apiSvc.Retrieve(
					AuthedCtx(ctx, reader),
					apichannel.AliasedRetrieveRequest{
						Request:  apichannel.RetrieveRequest{Where: both},
						RangeKey: &rng.Key,
					},
				)).Error().To(MatchError(access.ErrDenied))
			})
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
				apichannel.AliasedRetrieveRequest{Request: apichannel.RetrieveRequest{
					Where: &channel.Where{
						Key: &filter.Set[channel.Key]{
							In: []channel.Key{ch.Key(), 999999},
						},
					},
				}},
			))
			Expect(res.Channels).To(HaveLen(1))
			Expect(res.Channels[0].Key).To(Equal(ch.Key()))
		})

		It("Should return a path error for an invalid regex", func(ctx SpecContext) {
			Expect(apiSvc.Retrieve(
				AuthedCtx(ctx, author),
				apichannel.AliasedRetrieveRequest{Request: apichannel.RetrieveRequest{
					Where: &channel.Where{Name: &filter.String{Regex: new("(a")}},
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

var _ = Describe("AliasedRetrieveRequest", func() {
	It("Should reject a bare channel request", func() {
		var req apichannel.AliasedRetrieveRequest
		Expect(json.Unmarshal([]byte(`{"where":{}}`), &req)).To(SatisfyAll(
			BeAssignableToTypeOf(&json.SemanticError{}),
			MatchError(ContainSubstring(`unknown object member name "where"`)),
		))
	})

	DescribeTable(
		"Should reject a MessagePack member that the request does not declare",
		func(body any) {
			data := MustSucceed(msgpack.Marshal(body))
			var req apichannel.AliasedRetrieveRequest
			Expect(msgpack.Unmarshal(data, &req)).
				To(MatchError(`msgpack: unknown field "keys"`))
		},
		Entry("in a bare channel request", map[string]any{"keys": []int{1048577}}),
		Entry("after the request", struct {
			Request map[string]any `msgpack:"request"`
			Keys    []int          `msgpack:"keys"`
		}{Request: map[string]any{}, Keys: []int{1048577}}),
	)

	It("Should decode a MessagePack request that it declares", func() {
		rng := uuid.New()
		data := MustSucceed(msgpack.Marshal(map[string]any{
			"request":   map[string]any{"search_term": "temp"},
			"range_key": rng,
		}))
		var req apichannel.AliasedRetrieveRequest
		Expect(msgpack.Unmarshal(data, &req)).To(Succeed())
		Expect(req.Request.SearchTerm).To(Equal("temp"))
		Expect(req.RangeKey).To(HaveValue(Equal(rng)))
	})
})

var _ = Describe("RetrieveRequest", func() {
	It("Should reject a request member that it does not declare", func() {
		var req apichannel.RetrieveRequest
		Expect(json.Unmarshal([]byte(`{"keys":[1]}`), &req)).To(SatisfyAll(
			BeAssignableToTypeOf(&json.SemanticError{}),
			MatchError(ContainSubstring(`unknown object member name "keys"`)),
		))
	})
})
