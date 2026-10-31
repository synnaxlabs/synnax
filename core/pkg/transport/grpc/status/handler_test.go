// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api/status"
	"github.com/synnaxlabs/synnax/pkg/api/status/pb"
	svcstatus "github.com/synnaxlabs/synnax/pkg/service/status"
	grpcstatus "github.com/synnaxlabs/synnax/pkg/transport/grpc/status"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

var _ = Describe("Retrieve", func() {
	var (
		client   grpcstatus.StatusRetrieveServiceClient
		received status.RetrieveRequest
	)

	BeforeEach(func() {
		client = grpcstatus.NewStatusRetrieveServiceClient(conn)
		received = status.RetrieveRequest{}
		transport.StatusRetrieve.BindHandler(func(
			_ context.Context,
			req status.RetrieveRequest,
		) (status.RetrieveResponse, error) {
			received = req
			return status.RetrieveResponse{}, nil
		})
	})

	It("Should decode a filter tree", func(ctx SpecContext) {
		msg := status.RetrieveRequest{
			Where: &svcstatus.Where{
				Or: []svcstatus.Where{
					{
						Key: &filter.Set[svcstatus.Key]{
							In: []svcstatus.Key{"a", "b"},
						},
					},
					{Not: &svcstatus.Where{Variant: &filter.Set[svcstatus.Variant]{
						Eq: new(svcstatus.VariantError),
					}}},
				},
			},
			SearchTerm:    "cryostat",
			Limit:         new(uint32(10)),
			IncludeLabels: true,
		}
		MustSucceed(client.Exec(ctx, &grpcstatus.RetrieveRequest{
			Request: MustSucceed(pb.RetrieveRequestToPB(msg)),
		}))
		Expect(received).To(Equal(msg))
	})

	It("Should reject a request that sends the reserved keys field", func(
		ctx SpecContext,
	) {
		b := protowire.AppendTag(nil, 1, protowire.BytesType)
		b = protowire.AppendString(b, "a")
		var decoded grpcstatus.RetrieveRequest
		Expect(proto.Unmarshal(b, &decoded)).To(Succeed())
		Expect(client.Exec(ctx, &decoded)).Error().To(MatchError(ContainSubstring(
			"sy.validation---grpc.status.RetrieveRequest has unknown fields",
		)))
	})

	It("Should reject an unknown field inside the request", func(ctx SpecContext) {
		inner := protowire.AppendTag(nil, 99, protowire.VarintType)
		inner = protowire.AppendVarint(inner, 1)
		b := protowire.AppendTag(nil, 9, protowire.BytesType)
		b = protowire.AppendBytes(b, inner)
		var decoded grpcstatus.RetrieveRequest
		Expect(proto.Unmarshal(b, &decoded)).To(Succeed())
		Expect(client.Exec(ctx, &decoded)).Error().To(MatchError(ContainSubstring(
			"sy.validation---api.status.pb.RetrieveRequest has unknown fields",
		)))
	})
})
