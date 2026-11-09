// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package device_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api/device"
	"github.com/synnaxlabs/synnax/pkg/api/device/pb"
	svcdevice "github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	grpcdevice "github.com/synnaxlabs/synnax/pkg/transport/grpc/device"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

var _ = Describe("Retrieve", func() {
	var (
		client   grpcdevice.DeviceRetrieveServiceClient
		received device.RetrieveRequest
	)

	BeforeEach(func() {
		client = grpcdevice.NewDeviceRetrieveServiceClient(conn)
		received = device.RetrieveRequest{}
		transport.DeviceRetrieve.BindHandler(func(
			_ context.Context,
			req device.RetrieveRequest,
		) (device.RetrieveResponse, error) {
			received = req
			return device.RetrieveResponse{}, nil
		})
	})

	It("Should decode a filter tree", func(ctx SpecContext) {
		msg := device.RetrieveRequest{
			Where: &svcdevice.Where{
				Or: []svcdevice.Where{
					{Key: &filter.Set[string]{In: []string{"dev1"}}},
					{Make: &filter.String{Eq: new("NI")}},
				},
				Not: &svcdevice.Where{
					Rack: &filter.Set[rack.Key]{In: []rack.Key{1}},
				},
				Name: &filter.String{Regex: new("^PT")},
			},
			SearchTerm:    "cDAQ",
			IncludeStatus: true,
		}
		MustSucceed(client.Exec(ctx, &grpcdevice.RetrieveRequest{
			Request: MustSucceed(pb.RetrieveRequestToPB(msg)),
		}))
		Expect(received).To(Equal(msg))
	})

	It("Should reject a request that sends the reserved keys field", func(
		ctx SpecContext,
	) {
		b := protowire.AppendTag(nil, 1, protowire.BytesType)
		b = protowire.AppendString(b, "dev1")
		var decoded grpcdevice.RetrieveRequest
		Expect(proto.Unmarshal(b, &decoded)).To(Succeed())
		Expect(client.Exec(ctx, &decoded)).Error().To(MatchError(ContainSubstring(
			"sy.validation---grpc.device.RetrieveRequest has unknown fields",
		)))
	})

	It("Should reject an unknown field inside the request", func(ctx SpecContext) {
		inner := protowire.AppendTag(nil, 99, protowire.VarintType)
		inner = protowire.AppendVarint(inner, 1)
		b := protowire.AppendTag(nil, 13, protowire.BytesType)
		b = protowire.AppendBytes(b, inner)
		var decoded grpcdevice.RetrieveRequest
		Expect(proto.Unmarshal(b, &decoded)).To(Succeed())
		Expect(client.Exec(ctx, &decoded)).Error().To(MatchError(ContainSubstring(
			"sy.validation---api.device.pb.RetrieveRequest has unknown fields",
		)))
	})
})

var _ = Describe("Dispatch", func() {
	var (
		client   grpcdevice.DeviceDispatchServiceClient
		received device.DispatchRequest
	)

	BeforeEach(func() {
		client = grpcdevice.NewDeviceDispatchServiceClient(conn)
		received = device.DispatchRequest{}
		transport.DeviceDispatch.BindHandler(func(
			_ context.Context,
			req device.DispatchRequest,
		) (struct{}, error) {
			received = req
			return struct{}{}, nil
		})
	})

	It("Should decode each action from the JSON form that HTTP takes", func(
		ctx SpecContext,
	) {
		set := MustSucceed(structpb.NewStruct(map[string]any{
			"type": "set",
			"set": map[string]any{
				"path":  []any{"properties"},
				"value": map[string]any{"port": 2.0},
			},
		}))
		MustSucceed(client.Exec(ctx, &grpcdevice.DispatchRequest{
			Key:         "dev1",
			DispatchKey: "dk",
			Actions:     []*structpb.Struct{set},
		}))
		Expect(received).To(Equal(device.DispatchRequest{
			Key:         "dev1",
			DispatchKey: "dk",
			Actions: []svcdevice.Action{svcdevice.NewSetAction(svcdevice.SetPayload{
				Path:  []string{"properties"},
				Value: map[string]any{"port": 2.0},
			})},
		}))
	})

	It("Should reject an action that does not decode", func(ctx SpecContext) {
		bad := MustSucceed(structpb.NewStruct(map[string]any{"type": 1.0}))
		Expect(client.Exec(ctx, &grpcdevice.DispatchRequest{
			Key:     "dev1",
			Actions: []*structpb.Struct{bad},
		})).Error().To(MatchError(ContainSubstring(`"path":["actions","0"]`)))
	})
})
