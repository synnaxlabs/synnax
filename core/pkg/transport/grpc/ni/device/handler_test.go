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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api/ni/device"
	"github.com/synnaxlabs/synnax/pkg/api/ni/device/pb"
	svcdevice "github.com/synnaxlabs/synnax/pkg/service/ni/device"
	devicepb "github.com/synnaxlabs/synnax/pkg/service/ni/device/pb"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	grpcdevice "github.com/synnaxlabs/synnax/pkg/transport/grpc/ni/device"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Handler", func() {
	It("Should create devices and return their keys", func(ctx SpecContext) {
		key := uuid.New()
		var received device.CreateRequest
		transport.NIDeviceCreate.BindHandler(func(
			_ context.Context,
			req device.CreateRequest,
		) (device.CreateResponse, error) {
			received = req
			req.Devices[0].Key = key
			return device.CreateResponse(req), nil
		})
		dev := svcdevice.Device{
			Rack:         rack.NewKey(1, 2),
			Name:         "Module",
			ResourceName: "{GUID}",
			Model:        "NI 9205",
		}
		pbDev := MustSucceed(devicepb.DeviceToPB(dev))
		res := MustSucceed(grpcdevice.NewNIDeviceCreateServiceClient(conn).Exec(
			ctx,
			&grpcdevice.CreateRequest{Devices: []*devicepb.Device{pbDev}},
		))
		Expect(received.Devices).To(HaveLen(1))
		Expect(received.Devices[0].ResourceName).To(Equal("{GUID}"))
		Expect(res.Devices[0].Key).To(Equal(key.String()))
	})

	It("Should retrieve devices by resource name", func(ctx SpecContext) {
		key := uuid.New()
		var received device.RetrieveRequest
		transport.NIDeviceRetrieve.BindHandler(func(
			_ context.Context,
			req device.RetrieveRequest,
		) (device.RetrieveResponse, error) {
			received = req
			return device.RetrieveResponse{
				Devices: []svcdevice.Device{{Key: key, Name: "Module"}},
			}, nil
		})
		where := MustSucceed(devicepb.WhereToPB(svcdevice.Where{
			ResourceName: &filter.String{Eq: new("{GUID}")},
		}))
		res := MustSucceed(grpcdevice.NewNIDeviceRetrieveServiceClient(conn).Exec(
			ctx,
			&pb.RetrieveRequest{Where: where, IncludeStatus: true},
		))
		Expect(*received.Where.ResourceName.Eq).To(Equal("{GUID}"))
		Expect(received.IncludeStatus).To(BeTrue())
		Expect(res.Devices[0].Key).To(Equal(key.String()))
	})

	It("Should update the identity group of a device", func(ctx SpecContext) {
		key := uuid.New()
		var received device.UpdateRequest
		transport.NIDeviceUpdate.BindHandler(func(
			_ context.Context,
			req device.UpdateRequest,
		) (struct{}, error) {
			received = req
			return struct{}{}, nil
		})
		update := svcdevice.Update{Variant: svcdevice.IdentityUpdate{
			Identity: svcdevice.Identity{
				Rack:         rack.NewKey(1, 2),
				Serial:       "01A",
				ResourceName: "{GUID}",
				Model:        "NI 9205",
			},
		}}
		pbUpdate := MustSucceed(devicepb.UpdateToPB(update))
		MustSucceed(grpcdevice.NewNIDeviceUpdateServiceClient(conn).Exec(
			ctx,
			&grpcdevice.UpdateRequest{Key: key.String(), Update: pbUpdate},
		))
		Expect(received.Key).To(Equal(key))
		Expect(received.Update).To(Equal(update))
	})

	It("Should delete devices by key", func(ctx SpecContext) {
		key := uuid.New()
		var received device.DeleteRequest
		transport.NIDeviceDelete.BindHandler(func(
			_ context.Context,
			req device.DeleteRequest,
		) (struct{}, error) {
			received = req
			return struct{}{}, nil
		})
		MustSucceed(grpcdevice.NewNIDeviceDeleteServiceClient(conn).Exec(
			ctx,
			&grpcdevice.DeleteRequest{Keys: []string{key.String()}},
		))
		Expect(received.Keys).To(Equal([]svcdevice.Key{key}))
	})

	It(
		"Should reject a port number that does not fit in 16 bits",
		func(ctx SpecContext) {
			transport.NIDeviceCreate.BindHandler(func(
				_ context.Context,
				req device.CreateRequest,
			) (device.CreateResponse, error) {
				return device.CreateResponse(req), nil
			})
			pbDev := MustSucceed(devicepb.DeviceToPB(svcdevice.Device{Key: uuid.New()}))
			pbDev.CounterInput = map[uint32]*devicepb.CounterInputChannels{65536: {}}
			Expect(grpcdevice.NewNIDeviceCreateServiceClient(conn).Exec(
				ctx,
				&grpcdevice.CreateRequest{Devices: []*devicepb.Device{pbDev}},
			)).Error().To(MatchError(ContainSubstring("out of range")))
		},
	)
})
