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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api/channel"
	"github.com/synnaxlabs/synnax/pkg/api/channel/pb"
	svcchannel "github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	grpcchannel "github.com/synnaxlabs/synnax/pkg/transport/grpc/channel"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

var _ = Describe("Retrieve", func() {
	var (
		client   grpcchannel.ChannelRetrieveServiceClient
		received channel.RetrieveRequest
		response channel.RetrieveResponse
	)

	BeforeEach(func() {
		client = grpcchannel.NewChannelRetrieveServiceClient(conn)
		received = channel.RetrieveRequest{}
		response = channel.RetrieveResponse{}
		transport.ChannelRetrieve.BindHandler(func(
			_ context.Context,
			req channel.RetrieveRequest,
		) (channel.RetrieveResponse, error) {
			received = req
			return response, nil
		})
	})

	It("Should decode a filter tree", func(ctx SpecContext) {
		req := channel.RetrieveRequest{
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
		}
		MustSucceed(client.Exec(ctx, &grpcchannel.RetrieveRequest{
			Request: MustSucceed(pb.RetrieveRequestToPB(req)),
		}))
		Expect(received).To(Equal(req))
	})

	It("Should reject a request that sends the reserved keys field", func(
		ctx SpecContext,
	) {
		b := protowire.AppendTag(nil, 2, protowire.VarintType)
		b = protowire.AppendVarint(b, 1048577)
		var decoded grpcchannel.RetrieveRequest
		Expect(proto.Unmarshal(b, &decoded)).To(Succeed())
		Expect(client.Exec(ctx, &decoded)).Error().To(MatchError(ContainSubstring(
			"sy.validation---grpc.channel.RetrieveRequest has unknown fields",
		)))
	})

	It("Should reject a request that sends the reserved range key field", func(
		ctx SpecContext,
	) {
		b := protowire.AppendTag(nil, 6, protowire.BytesType)
		b = protowire.AppendString(b, "a3c0b7a6-0a52-4a8e-9f2f-6c1f0e4a9d10")
		var decoded grpcchannel.RetrieveRequest
		Expect(proto.Unmarshal(b, &decoded)).To(Succeed())
		Expect(client.Exec(ctx, &decoded)).Error().To(MatchError(ContainSubstring(
			"sy.validation---grpc.channel.RetrieveRequest has unknown fields",
		)))
	})

	It("Should reject an unknown field inside the request", func(ctx SpecContext) {
		inner := protowire.AppendTag(nil, 99, protowire.VarintType)
		inner = protowire.AppendVarint(inner, 1)
		b := protowire.AppendTag(nil, 5, protowire.BytesType)
		b = protowire.AppendBytes(b, inner)
		var decoded grpcchannel.RetrieveRequest
		Expect(proto.Unmarshal(b, &decoded)).To(Succeed())
		Expect(client.Exec(ctx, &decoded)).Error().To(MatchError(ContainSubstring(
			"sy.validation---api.channel.pb.RetrieveRequest has unknown fields",
		)))
	})

	It("Should encode a channel with a status", func(ctx SpecContext) {
		ch := channel.Channel{
			Key:      1048577,
			Name:     "pressure",
			DataType: telem.Float32T,
			Status: &channel.Status{
				Key:     "channel:1048577",
				Name:    "pressure",
				Variant: status.VariantSuccess,
				Message: "Calculation running",
				Time:    telem.TimeStamp(1),
				// The protobuf encoding has no nil list, so it decodes an empty one.
				Labels: []label.Key{},
			},
		}
		response = channel.RetrieveResponse{Channels: []channel.Channel{ch}}
		res := MustSucceed(pb.RetrieveResponseFromPB(
			MustSucceed(client.Exec(ctx, &grpcchannel.RetrieveRequest{
				Request: &pb.RetrieveRequest{},
			})),
		))
		Expect(res.Channels).To(HaveLen(1))
		Expect(res.Channels[0].Status).To(Equal(ch.Status))
	})
})
