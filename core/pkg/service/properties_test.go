// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package service_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/serial"
	"github.com/synnaxlabs/synnax/pkg/service/tcp"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/udp"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Device properties", Ordered, func() {
	var (
		layer *service.Layer
		db    *gorp.DB
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		layer = MustOpen(svctestutil.OpenLayer(ctx, node))
	})

	newDevice := func(deviceMake string, props msgpack.EncodedJSON) device.Device {
		return device.Device{
			Key:        UniqueChannelName(),
			Rack:       layer.Rack.EmbeddedKey,
			Name:       deviceMake,
			Make:       deviceMake,
			Model:      deviceMake,
			Location:   deviceMake,
			Properties: props,
		}
	}

	create := func(ctx SpecContext, dev device.Device) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Device.NewWriter(tx).Create(ctx, &dev)
		})
	}

	DescribeTable("Should refuse a bus device whose properties fail to validate",
		func(
			ctx SpecContext,
			deviceMake string,
			props msgpack.EncodedJSON,
			path string,
			message string,
		) {
			err := create(ctx, newDevice(deviceMake, props))
			Expect(err).To(MatchError(validate.ErrValidation))
			Expect(err).To(MatchError(ContainSubstring(path + ": " + message)))
		},
		Entry("serial with no port", serial.Make, msgpack.EncodedJSON{
			"baud_rate": 9600,
			"data_bits": 8,
		}, "properties.port", "required"),
		Entry("serial with too few data bits", serial.Make, msgpack.EncodedJSON{
			"port":      "/dev/ttyUSB0",
			"baud_rate": 9600,
			"data_bits": 4,
		}, "properties.data_bits", "must be greater than or equal to 5"),
		Entry("TCP with no host", tcp.Make, msgpack.EncodedJSON{
			"port": 502,
		}, "properties.host", "required"),
		Entry("UDP with a remote with no port", udp.Make, msgpack.EncodedJSON{
			"remote": map[string]any{"host": "127.0.0.1"},
		}, "properties.remote.port", "must be greater than or equal to 1"),
	)

	It("Should refuse a bus device whose properties do not decode", func(
		ctx SpecContext,
	) {
		Expect(create(ctx, newDevice(tcp.Make, msgpack.EncodedJSON{
			"host": "127.0.0.1",
			"port": "modbus",
		}))).To(MatchError(validate.ErrValidation))
	})

	It("Should create bus devices whose properties validate", func(ctx SpecContext) {
		Expect(create(ctx, newDevice(serial.Make, msgpack.EncodedJSON{
			"port":      "/dev/ttyUSB0",
			"baud_rate": 9600,
			"data_bits": 8,
		}))).To(Succeed())
		Expect(create(ctx, newDevice(tcp.Make, msgpack.EncodedJSON{
			"host": "127.0.0.1",
			"port": 502,
		}))).To(Succeed())
		Expect(create(ctx, newDevice(udp.Make, msgpack.EncodedJSON{}))).To(Succeed())
	})

	It("Should refuse an edit that breaks the properties of a bus device", func(
		ctx SpecContext,
	) {
		dev := newDevice(
			tcp.Make,
			msgpack.EncodedJSON{"host": "127.0.0.1", "port": 502},
		)
		Expect(create(ctx, dev)).To(Succeed())
		err := db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Device.NewWriter(tx).Apply(
				ctx,
				dev.Key,
				device.NewSetAction(device.SetPayload{
					Path:  []string{"properties", "host"},
					Value: "",
				}),
			)
		})
		Expect(err).To(MatchError(validate.ErrValidation))
		Expect(err).To(MatchError(ContainSubstring("properties.host: required")))
	})

	It("Should store any properties on a device of a make with no check", func(
		ctx SpecContext,
	) {
		Expect(create(ctx, newDevice("modbus", msgpack.EncodedJSON{
			"port": "modbus",
		}))).To(Succeed())
	})
})
