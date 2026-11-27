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
	"github.com/synnaxlabs/synnax/pkg/api/labjack/device"
	"github.com/synnaxlabs/synnax/pkg/api/labjack/device/pb"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	svcdevice "github.com/synnaxlabs/synnax/pkg/service/labjack/device"
	devicepb "github.com/synnaxlabs/synnax/pkg/service/labjack/device/pb"
	grpcdevice "github.com/synnaxlabs/synnax/pkg/transport/grpc/labjack/device"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Handler", func() {
	t7 := svcdevice.Device{
		Key:        uuid.New(),
		Rack:       1,
		Name:       "T7",
		Serial:     470010001,
		Connection: svcdevice.ConnectionUSB,
		Model:      svcdevice.ModelT7,
		Ports: svcdevice.Ports{Variant: svcdevice.T7Ports{
			Mux80Attached: true,
			AnalogInput: map[svcdevice.T7AnalogInputPort]channel.Key{
				svcdevice.T7AnalogInputPortAin48: 1,
			},
			AnalogOutput: map[svcdevice.AnalogOutputPort]svcdevice.CommandStatePair{
				svcdevice.AnalogOutputPortDac0: {
					Command: 2,
					State:   new(channel.Key(3)),
				},
			},
			DigitalInput: map[svcdevice.T7DigitalPort]channel.Key{
				svcdevice.T7DigitalPortDio0: 4,
			},
			DigitalOutput: map[svcdevice.T7DigitalPort]svcdevice.CommandStatePair{
				svcdevice.T7DigitalPortDio1: {Command: 5},
			},
		}},
	}

	BeforeEach(func() {
		transport.LabJackDeviceDispatch.BindHandler(func(
			context.Context,
			device.DispatchRequest,
		) (struct{}, error) {
			return struct{}{}, nil
		})
	})

	It("Should keep the port maps of a device through the translators", func() {
		pbDev := MustSucceed(devicepb.DeviceToPB(t7))
		Expect(devicepb.DeviceFromPB(pbDev)).To(Equal(t7))
	})

	It("Should create devices with their port maps", func(ctx SpecContext) {
		var received device.CreateRequest
		transport.LabJackDeviceCreate.BindHandler(func(
			_ context.Context,
			req device.CreateRequest,
		) (device.CreateResponse, error) {
			received = req
			return device.CreateResponse(req), nil
		})
		pbDev := MustSucceed(devicepb.DeviceToPB(t7))
		res := MustSucceed(grpcdevice.NewLabJackDeviceCreateServiceClient(conn).Exec(
			ctx,
			&grpcdevice.CreateRequest{Devices: []*devicepb.Device{pbDev}},
		))
		Expect(received.Devices).To(Equal([]svcdevice.Device{t7}))
		Expect(res.Devices[0].Key).To(Equal(t7.Key.String()))
	})

	It("Should refuse a port that the model does not have", func(ctx SpecContext) {
		transport.LabJackDeviceCreate.BindHandler(func(
			context.Context,
			device.CreateRequest,
		) (device.CreateResponse, error) {
			return device.CreateResponse{}, nil
		})
		pbDev := MustSucceed(devicepb.DeviceToPB(t7))
		pbDev.Ports.GetT7().DigitalInput = map[string]uint32{"DIO99": 4}
		Expect(grpcdevice.NewLabJackDeviceCreateServiceClient(conn).Exec(
			ctx,
			&grpcdevice.CreateRequest{Devices: []*devicepb.Device{pbDev}},
		)).Error().To(MatchError(ContainSubstring("DIO99")))
	})

	It("Should retrieve devices by serial", func(ctx SpecContext) {
		var received device.RetrieveRequest
		transport.LabJackDeviceRetrieve.BindHandler(func(
			_ context.Context,
			req device.RetrieveRequest,
		) (device.RetrieveResponse, error) {
			received = req
			return device.RetrieveResponse{Devices: []svcdevice.Device{t7}}, nil
		})
		where := MustSucceed(devicepb.WhereToPB(svcdevice.Where{
			Serial: &filter.Ordered[uint32]{Eq: new(uint32(470010001))},
		}))
		res := MustSucceed(grpcdevice.NewLabJackDeviceRetrieveServiceClient(conn).Exec(
			ctx,
			&pb.RetrieveRequest{Where: where},
		))
		Expect(*received.Where.Serial.Eq).To(BeEquivalentTo(470010001))
		Expect(res.Devices[0].Key).To(Equal(t7.Key.String()))
	})

	It("Should delete devices by key", func(ctx SpecContext) {
		var received device.DeleteRequest
		transport.LabJackDeviceDelete.BindHandler(func(
			_ context.Context,
			req device.DeleteRequest,
		) (struct{}, error) {
			received = req
			return struct{}{}, nil
		})
		MustSucceed(grpcdevice.NewLabJackDeviceDeleteServiceClient(conn).Exec(
			ctx,
			&grpcdevice.DeleteRequest{Keys: []string{t7.Key.String()}},
		))
		Expect(received.Keys).To(Equal([]svcdevice.Key{t7.Key}))
	})

	It("Should dispatch actions to a device", func(ctx SpecContext) {
		var received device.DispatchRequest
		transport.LabJackDeviceDispatch.BindHandler(func(
			_ context.Context,
			req device.DispatchRequest,
		) (struct{}, error) {
			received = req
			return struct{}{}, nil
		})
		act := svcdevice.NewSetAction(svcdevice.SetPayload{
			Path:  []string{"name"},
			Value: "Renamed",
		})
		MustSucceed(grpcdevice.NewLabJackDeviceDispatchServiceClient(conn).Exec(
			ctx,
			&grpcdevice.DispatchRequest{
				Key:         t7.Key.String(),
				DispatchKey: "dk",
				Actions:     [][]byte{MustSucceed(json.Codec.Encode(act))},
			},
		))
		Expect(received.Key).To(Equal(t7.Key))
		Expect(received.DispatchKey).To(Equal("dk"))
		Expect(received.Actions).To(HaveLen(1))
		Expect(received.Actions[0].Set.Path).To(Equal([]string{"name"}))
	})

	It("Should reject an action that does not decode", func(ctx SpecContext) {
		Expect(grpcdevice.NewLabJackDeviceDispatchServiceClient(conn).Exec(
			ctx,
			&grpcdevice.DispatchRequest{
				Key:     t7.Key.String(),
				Actions: [][]byte{[]byte("not json")},
			},
		)).Error().To(MatchError(ContainSubstring(`"path":["actions","0"]`)))
	})

	It("Should reject a dispatch with an invalid device key", func(ctx SpecContext) {
		Expect(grpcdevice.NewLabJackDeviceDispatchServiceClient(conn).Exec(
			ctx,
			&grpcdevice.DispatchRequest{Key: "not-a-uuid"},
		)).Error().To(MatchError(ContainSubstring("invalid uuid")))
	})
})
