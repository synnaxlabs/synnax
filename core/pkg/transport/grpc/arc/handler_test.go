// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api/arc"
	"github.com/synnaxlabs/synnax/pkg/api/arc/pb"
	svcarc "github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	grpcarc "github.com/synnaxlabs/synnax/pkg/transport/grpc/arc"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

var _ = Describe("Retrieve", func() {
	var (
		client   grpcarc.ArcRetrieveServiceClient
		received arc.RetrieveRequest
	)

	BeforeEach(func() {
		client = grpcarc.NewArcRetrieveServiceClient(conn)
		received = arc.RetrieveRequest{}
		transport.ArcRetrieve.BindHandler(func(
			_ context.Context,
			req arc.RetrieveRequest,
		) (arc.RetrieveResponse, error) {
			received = req
			return arc.RetrieveResponse{}, nil
		})
	})

	It("Should decode a filter tree", func(ctx SpecContext) {
		msg := arc.RetrieveRequest{
			Where: &svcarc.Where{
				Or: []svcarc.Where{
					{Key: &filter.Set[arc.Key]{In: []arc.Key{uuid.New()}}},
					{Name: &filter.String{Regex: new("^pump")}},
				},
			},
			SearchTerm:     "pump",
			Limit:          new(uint32(10)),
			IncludeProgram: true,
		}
		MustSucceed(client.Exec(ctx, &grpcarc.RetrieveRequest{
			Request: MustSucceed(pb.RetrieveRequestToPB(msg)),
		}))
		Expect(received).To(Equal(msg))
	})

	It("Should reject a request that sends the reserved keys field", func(
		ctx SpecContext,
	) {
		b := protowire.AppendTag(nil, 1, protowire.BytesType)
		b = protowire.AppendString(b, uuid.New().String())
		var decoded grpcarc.RetrieveRequest
		Expect(proto.Unmarshal(b, &decoded)).To(Succeed())
		Expect(client.Exec(ctx, &decoded)).Error().To(MatchError(ContainSubstring(
			"sy.validation---grpc.arc.RetrieveRequest has unknown fields",
		)))
	})
})

var _ = Describe("Calculations", func() {
	It("Should carry the requested channels and the response", func(ctx SpecContext) {
		var received arc.CalculationsRequest
		transport.ArcCalculations.BindHandler(func(
			_ context.Context,
			req arc.CalculationsRequest,
		) (arc.CalculationsResponse, error) {
			received = req
			return arc.CalculationsResponse{
				Calculations: []svcarc.Calculation{{Channel: 3}},
				Channels:     channel.Keys{1, 2, 3},
				Omitted:      channel.Keys{2},
			}, nil
		})
		client := grpcarc.NewArcCalculationsServiceClient(conn)
		res := MustSucceed(client.Exec(ctx, &grpcarc.CalculationsRequest{
			Channels: []uint32{1, 2},
		}))
		Expect(received.Channels).To(Equal(channel.Keys{1, 2}))
		Expect(res.Calculations).To(HaveLen(1))
		Expect(res.Calculations[0].Channel).To(Equal(uint32(3)))
		Expect(res.Channels).To(Equal([]uint32{1, 2, 3}))
		Expect(res.Omitted).To(Equal([]uint32{2}))
	})
})
