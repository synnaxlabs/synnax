// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api/channel"
	"github.com/synnaxlabs/synnax/pkg/api/channel/pb"
	svcchannel "github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

var _ = Describe("gRPC Channel Translators", func() {
	Describe("RetrieveRequest", func() {
		t := retrieveRequestTranslator{}
		It("Should round-trip a filter tree and a range", func(ctx SpecContext) {
			rangeKey := uuid.New()
			msg := channel.AliasedRetrieveRequest{
				Request: channel.RetrieveRequest{
					Where: &svcchannel.Where{
						Or: []svcchannel.Where{
							{Key: &filter.Set[channel.Key]{In: []channel.Key{1048577}}},
							{Name: &filter.String{Regex: new("^pressure")}},
						},
						Not: &svcchannel.Where{
							DataType: &filter.Set[telem.DataType]{
								In: []telem.DataType{telem.StringT},
							},
						},
						Leaseholder: &filter.Set[node.Key]{Eq: new(node.Key(1))},
						IsIndex:     &filter.Bool{Eq: new(false)},
						Virtual:     new(false),
					},
					SearchTerm:    "pressure",
					Limit:         new(uint32(10)),
					IncludeStatus: true,
				},
				RangeKey: &rangeKey,
			}
			fwd := MustSucceed(t.Forward(ctx, msg))
			b := MustSucceed(proto.Marshal(fwd))
			var decoded RetrieveRequest
			Expect(proto.Unmarshal(b, &decoded)).To(Succeed())
			Expect(MustSucceed(t.Backward(ctx, &decoded))).To(Equal(msg))
		})

		It("Should reject a request that sends the reserved keys field", func(
			ctx SpecContext,
		) {
			b := protowire.AppendTag(nil, 2, protowire.VarintType)
			b = protowire.AppendVarint(b, 1048577)
			var decoded RetrieveRequest
			Expect(proto.Unmarshal(b, &decoded)).To(Succeed())
			Expect(t.Backward(ctx, &decoded)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(
					ContainSubstring("grpc.channel.RetrieveRequest has unknown fields"),
				),
			))
		})

		It("Should reject an unknown field inside the request", func(ctx SpecContext) {
			inner := protowire.AppendTag(nil, 99, protowire.VarintType)
			inner = protowire.AppendVarint(inner, 1)
			b := protowire.AppendTag(nil, 5, protowire.BytesType)
			b = protowire.AppendBytes(b, inner)
			var decoded RetrieveRequest
			Expect(proto.Unmarshal(b, &decoded)).To(Succeed())
			Expect(t.Backward(ctx, &decoded)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"api.channel.pb.RetrieveRequest has unknown fields",
				)),
			))
		})
	})

	Describe("RetrieveResponse", func() {
		t := retrieveResponseTranslator{}
		It("Should round-trip a channel with a status", func(ctx SpecContext) {
			msg := channel.RetrieveResponse{Channels: []channel.Channel{{
				Key:      1048577,
				Name:     "pressure",
				DataType: telem.Float32T,
				Status: &channel.Status{
					Key:     "channel:1048577",
					Name:    "pressure",
					Variant: status.VariantSuccess,
					Message: "Calculation running",
					Time:    telem.TimeStamp(1),
				},
			}}}
			fwd := MustSucceed(t.Forward(ctx, msg))
			b := MustSucceed(proto.Marshal(fwd))
			var decoded pb.RetrieveResponse
			Expect(proto.Unmarshal(b, &decoded)).To(Succeed())
			res := MustSucceed(t.Backward(ctx, &decoded))
			Expect(res.Channels).To(HaveLen(1))
			Expect(res.Channels[0].Status).To(Equal(msg.Channels[0].Status))
		})
	})
})
