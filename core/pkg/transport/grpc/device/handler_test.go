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
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api/device"
	"github.com/synnaxlabs/synnax/pkg/api/device/pb"
	svcdevice "github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	grpcdevice "github.com/synnaxlabs/synnax/pkg/transport/grpc/device"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/patch"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
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
		MustSucceed(client.Exec(ctx, &grpcdevice.DispatchRequest{
			Key:         "dev1",
			DispatchKey: "dk",
			Actions: [][]byte{[]byte(`{"type": "set", "set": {
				"path": ["properties"],
				"value": {"port": 2}
			}}`)},
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
		Expect(client.Exec(ctx, &grpcdevice.DispatchRequest{
			Key:     "dev1",
			Actions: [][]byte{[]byte(`{"type": 1}`)},
		})).Error().To(MatchError(ContainSubstring(`"path":["actions","0"]`)))
	})

	It("Should encode each action as JSON that decodes to the same action", func(
		ctx SpecContext,
	) {
		translator := transport.DeviceDispatch.(*fgrpc.UnaryServer[
			device.DispatchRequest,
			*grpcdevice.DispatchRequest,
			struct{},
			*emptypb.Empty,
		]).RequestTranslator.(fgrpc.Translator[
			device.DispatchRequest,
			*grpcdevice.DispatchRequest,
		])
		req := device.DispatchRequest{
			Key:         "dev1",
			DispatchKey: "dk",
			Actions: []svcdevice.Action{svcdevice.NewSetAction(svcdevice.SetPayload{
				Path:   []string{"model"},
				Value:  "USB-6000",
				Guards: []patch.Guard{{Path: []string{"model"}, Value: "USB-6008"}},
			})},
		}
		Expect(translator.Backward(ctx, MustSucceed(translator.Forward(ctx, req)))).
			To(Equal(req))
	})
})
