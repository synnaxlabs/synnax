// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api/ranger"
	"github.com/synnaxlabs/synnax/pkg/api/ranger/pb"
	svcranger "github.com/synnaxlabs/synnax/pkg/service/ranger"
	grpcranger "github.com/synnaxlabs/synnax/pkg/transport/grpc/ranger"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

var _ = Describe("Retrieve", func() {
	var (
		client   grpcranger.RangeRetrieveServiceClient
		received ranger.RetrieveRequest
	)

	BeforeEach(func() {
		client = grpcranger.NewRangeRetrieveServiceClient(conn)
		received = ranger.RetrieveRequest{}
		transport.RangeRetrieve.BindHandler(func(
			_ context.Context,
			req ranger.RetrieveRequest,
		) (ranger.RetrieveResponse, error) {
			received = req
			return ranger.RetrieveResponse{}, nil
		})
	})

	It("Should decode a filter tree", func(ctx SpecContext) {
		key := uuid.New()
		msg := ranger.RetrieveRequest{
			Where: &svcranger.Where{
				Or: []svcranger.Where{
					{Key: &filter.Set[ranger.Key]{In: []ranger.Key{key}}},
					{
						Not: &svcranger.Where{
							Name: &filter.String{Eq: new("Cold flow")},
						},
					},
				},
				Overlaps: &telem.TimeRange{Start: 1, End: 2},
				Parents:  []ranger.Key{key},
			},
			Limit: new(uint32(10)),
		}
		MustSucceed(client.Exec(ctx, &grpcranger.RetrieveRequest{
			Request: MustSucceed(pb.RetrieveRequestToPB(msg)),
		}))
		Expect(received).To(Equal(msg))
	})

	It("Should reject a request that sends the reserved keys field", func(
		ctx SpecContext,
	) {
		b := protowire.AppendTag(nil, 1, protowire.BytesType)
		b = protowire.AppendString(b, uuid.New().String())
		var decoded grpcranger.RetrieveRequest
		Expect(proto.Unmarshal(b, &decoded)).To(Succeed())
		Expect(client.Exec(ctx, &decoded)).Error().To(MatchError(ContainSubstring(
			"sy.validation---grpc.ranger.RetrieveRequest has unknown fields",
		)))
	})

	It("Should reject an unknown field inside the request", func(ctx SpecContext) {
		inner := protowire.AppendTag(nil, 99, protowire.VarintType)
		inner = protowire.AppendVarint(inner, 1)
		b := protowire.AppendTag(nil, 3, protowire.BytesType)
		b = protowire.AppendBytes(b, inner)
		var decoded grpcranger.RetrieveRequest
		Expect(proto.Unmarshal(b, &decoded)).To(Succeed())
		Expect(client.Exec(ctx, &decoded)).Error().To(MatchError(ContainSubstring(
			"sy.validation---api.ranger.pb.RetrieveRequest has unknown fields",
		)))
	})
})
