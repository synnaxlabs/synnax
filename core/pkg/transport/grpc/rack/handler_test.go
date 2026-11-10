// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rack_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api/rack"
	"github.com/synnaxlabs/synnax/pkg/api/rack/pb"
	svcrack "github.com/synnaxlabs/synnax/pkg/service/rack"
	rackpb "github.com/synnaxlabs/synnax/pkg/service/rack/pb"
	grpcrack "github.com/synnaxlabs/synnax/pkg/transport/grpc/rack"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/patch"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ = Describe("Create", func() {
	It("Should accept a rack with no role, as the C++ client sends", func(
		ctx SpecContext,
	) {
		var received rack.CreateRequest
		transport.RackCreate.BindHandler(func(
			_ context.Context,
			req rack.CreateRequest,
		) (rack.CreateResponse, error) {
			received = req
			return rack.CreateResponse{}, nil
		})
		MustSucceed(grpcrack.NewRackCreateServiceClient(conn).Exec(
			ctx,
			&grpcrack.CreateRequest{
				Racks: []*rackpb.Rack{{Key: 65537, Name: "Driver rack"}},
			},
		))
		Expect(received.Racks).To(Equal([]rack.Rack{{Key: 65537, Name: "Driver rack"}}))
	})
})

var _ = Describe("Retrieve", func() {
	var (
		client   grpcrack.RackRetrieveServiceClient
		received rack.RetrieveRequest
	)

	BeforeEach(func() {
		client = grpcrack.NewRackRetrieveServiceClient(conn)
		received = rack.RetrieveRequest{}
		transport.RackRetrieve.BindHandler(func(
			_ context.Context,
			req rack.RetrieveRequest,
		) (rack.RetrieveResponse, error) {
			received = req
			return rack.RetrieveResponse{}, nil
		})
	})

	It("Should decode a filter tree", func(ctx SpecContext) {
		msg := rack.RetrieveRequest{
			Where: &svcrack.Where{
				Or: []svcrack.Where{
					{Key: &filter.Set[rack.Key]{In: []rack.Key{65537}}},
					{Integrations: &filter.Contains[string]{Any: []string{"ni"}}},
				},
				Not: &svcrack.Where{Embedded: &filter.Bool{Eq: new(true)}},
			},
			Limit:         new(uint32(10)),
			IncludeStatus: true,
		}
		MustSucceed(client.Exec(ctx, &grpcrack.RetrieveRequest{
			Request: MustSucceed(pb.RetrieveRequestToPB(msg)),
		}))
		Expect(received).To(Equal(msg))
	})

	It("Should reject a request that sends the reserved keys field", func(
		ctx SpecContext,
	) {
		b := protowire.AppendTag(nil, 1, protowire.VarintType)
		b = protowire.AppendVarint(b, 65537)
		var decoded grpcrack.RetrieveRequest
		Expect(proto.Unmarshal(b, &decoded)).To(Succeed())
		Expect(client.Exec(ctx, &decoded)).Error().To(MatchError(ContainSubstring(
			"sy.validation---grpc.rack.RetrieveRequest has unknown fields",
		)))
	})

	It("Should reject an unknown field inside the request", func(ctx SpecContext) {
		inner := protowire.AppendTag(nil, 99, protowire.VarintType)
		inner = protowire.AppendVarint(inner, 1)
		b := protowire.AppendTag(nil, 5, protowire.BytesType)
		b = protowire.AppendBytes(b, inner)
		var decoded grpcrack.RetrieveRequest
		Expect(proto.Unmarshal(b, &decoded)).To(Succeed())
		Expect(client.Exec(ctx, &decoded)).Error().To(MatchError(ContainSubstring(
			"sy.validation---api.rack.pb.RetrieveRequest has unknown fields",
		)))
	})
})

var _ = Describe("Dispatch", func() {
	var (
		client   grpcrack.RackDispatchServiceClient
		received rack.DispatchRequest
	)

	BeforeEach(func() {
		client = grpcrack.NewRackDispatchServiceClient(conn)
		received = rack.DispatchRequest{}
		transport.RackDispatch.BindHandler(func(
			_ context.Context,
			req rack.DispatchRequest,
		) (struct{}, error) {
			received = req
			return struct{}{}, nil
		})
	})

	It("Should decode each action from the JSON form that HTTP takes", func(
		ctx SpecContext,
	) {
		MustSucceed(client.Exec(ctx, &grpcrack.DispatchRequest{
			Key:         65537,
			DispatchKey: "dk",
			Actions: [][]byte{
				[]byte(
					`{"type": "set", "set": {"path": ["version"], "value": "0.50.0"}}`,
				),
			},
		}))
		Expect(received).To(Equal(rack.DispatchRequest{
			Key:         65537,
			DispatchKey: "dk",
			Actions: []svcrack.Action{svcrack.NewSetAction(svcrack.SetPayload{
				Path:  []string{"version"},
				Value: "0.50.0",
			})},
		}))
	})

	It("Should reject an action that does not decode", func(ctx SpecContext) {
		Expect(client.Exec(ctx, &grpcrack.DispatchRequest{
			Key:     65537,
			Actions: [][]byte{[]byte(`{"type": 1}`)},
		})).Error().To(MatchError(ContainSubstring(`"path":["actions","0"]`)))
	})

	It("Should encode each action as JSON that decodes to the same action", func(
		ctx SpecContext,
	) {
		translator := transport.RackDispatch.(*fgrpc.UnaryServer[
			rack.DispatchRequest,
			*grpcrack.DispatchRequest,
			struct{},
			*emptypb.Empty,
		]).RequestTranslator.(fgrpc.Translator[
			rack.DispatchRequest,
			*grpcrack.DispatchRequest,
		])
		req := rack.DispatchRequest{
			Key:         65537,
			DispatchKey: "dk",
			Actions: []svcrack.Action{svcrack.NewSetAction(svcrack.SetPayload{
				Path:   []string{"version"},
				Value:  "0.2.0",
				Guards: []patch.Guard{{Path: []string{"version"}, Value: "0.1.0"}},
			})},
		}
		Expect(translator.Backward(ctx, MustSucceed(translator.Forward(ctx, req)))).
			To(Equal(req))
	})
})
