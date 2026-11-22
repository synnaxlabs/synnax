// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	v0 "github.com/synnaxlabs/synnax/pkg/service/modbus/server/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Server", func() {
	tcp := func() v0.TcpConnection {
		return v0.TcpConnection{
			TimeoutMs: 1000,
			Retries:   1,
			Host:      "plc",
			Port:      502,
			UnitID:    255,
		}
	}
	rtu := func() v0.RtuConnection {
		return v0.RtuConnection{
			TimeoutMs:  1000,
			Retries:    1,
			SerialPort: "/dev/ttyUSB0",
			BaudRate:   19200,
			Parity:     v0.ParityEven,
			StopBits:   1,
			UnitID:     1,
		}
	}
	server := func() v0.Server {
		return v0.Server{
			Key:        uuid.MustParse("22222222-2222-4222-8222-222222222222"),
			Rack:       1,
			Name:       "PLC",
			Connection: v0.Connection{Variant: tcp()},
			Requests: v0.Requests{
				MaxRegisters: 125,
				MaxGap:       125,
				WriteCodes:   v0.WriteCodesMultiple,
			},
		}
	}

	It("Should have a table prefix of its own", func() {
		Expect(types.Name[v0.Server]()).To(Equal("modbus_server"))
	})

	It("Should identify itself by its key", func() {
		s := server()
		Expect(s.GorpKey()).To(Equal(s.Key))
		Expect(s.ResourceID()).To(Equal(resource.ID{
			Type: resource.TypeModbusServer,
			Key:  s.Key.String(),
		}))
	})

	DescribeTable("Should apply the defaults of each transport",
		func(connection string, expected v0.ConnectionVariant) {
			var s v0.Server
			Expect(json.Unmarshal([]byte(`{
				"key": "22222222-2222-4222-8222-222222222222",
				"rack": 1,
				"name": "PLC",
				"connection": `+connection+`
			}`), &s)).To(Succeed())
			Expect(s.Connection.Variant).To(Equal(expected))
			Expect(s.Requests).To(Equal(server().Requests))
			Expect(s.Validate()).To(Succeed())
		},
		Entry("TCP", `{"transport": "tcp", "host": "plc", "port": 502}`, tcp()),
		Entry(
			"RTU",
			`{"transport": "rtu", "serial_port": "/dev/ttyUSB0"}`,
			rtu(),
		),
	)

	It("Should round trip a server through JSON", func() {
		s := server()
		s.Connection = v0.Connection{Variant: rtu()}
		s.ReadIndex = 3
		s.ReadChannels = map[string]channel.Key{"holding-register-5-float32": 4}
		s.WriteChannels = map[string]channel.Key{"coil-1": 5}
		var decoded v0.Server
		Expect(json.Unmarshal(MustSucceed(json.Marshal(s)), &decoded)).To(Succeed())
		Expect(decoded).To(Equal(s))
	})

	DescribeTable("Should accept a valid server",
		func(connection v0.ConnectionVariant) {
			s := server()
			s.Connection = v0.Connection{Variant: connection}
			Expect(s.Validate()).To(Succeed())
		},
		Entry("TCP", tcp()),
		Entry("RTU", rtu()),
	)

	DescribeTable("Should reject an invalid server",
		func(change func(*v0.Server), path string) {
			s := server()
			change(&s)
			Expect(s.Validate()).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(path)),
			))
		},
		Entry("no name", func(s *v0.Server) { s.Name = "" }, "name"),
		Entry("no transport", func(s *v0.Server) {
			s.Connection = v0.Connection{}
		}, "transport"),
		Entry("no host", func(s *v0.Server) {
			c := tcp()
			c.Host = ""
			s.Connection = v0.Connection{Variant: c}
		}, "host"),
		Entry("no port", func(s *v0.Server) {
			c := tcp()
			c.Port = 0
			s.Connection = v0.Connection{Variant: c}
		}, "port"),
		Entry("a zero timeout", func(s *v0.Server) {
			c := tcp()
			c.TimeoutMs = 0
			s.Connection = v0.Connection{Variant: c}
		}, "timeout_ms"),
		Entry("too many retries", func(s *v0.Server) {
			c := tcp()
			c.Retries = 11
			s.Connection = v0.Connection{Variant: c}
		}, "retries"),
		Entry("no serial port", func(s *v0.Server) {
			c := rtu()
			c.SerialPort = ""
			s.Connection = v0.Connection{Variant: c}
		}, "serial_port"),
		Entry("a baud rate above the limit", func(s *v0.Server) {
			c := rtu()
			c.BaudRate = 460800
			s.Connection = v0.Connection{Variant: c}
		}, "baud_rate"),
		Entry("an unknown parity", func(s *v0.Server) {
			c := rtu()
			c.Parity = "mark"
			s.Connection = v0.Connection{Variant: c}
		}, "parity"),
		Entry("three stop bits", func(s *v0.Server) {
			c := rtu()
			c.StopBits = 3
			s.Connection = v0.Connection{Variant: c}
		}, "stop_bits"),
		Entry("an RTU unit of 248", func(s *v0.Server) {
			c := rtu()
			c.UnitID = 248
			s.Connection = v0.Connection{Variant: c}
		}, "unit_id"),
		Entry("a read of 126 registers", func(s *v0.Server) {
			s.Requests.MaxRegisters = 126
		}, "max_registers"),
		Entry("an unknown write code", func(s *v0.Server) {
			s.Requests.WriteCodes = "masked"
		}, "write_codes"),
		Entry("a delay above five seconds", func(s *v0.Server) {
			s.Requests.DelayMs = 5001
		}, "delay_ms"),
	)
})
