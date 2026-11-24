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
	v0 "github.com/synnaxlabs/synnax/pkg/service/modbus/server/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("Server", func() {
	tcp := func() v0.TcpConnection {
		return v0.TcpConnection{
			Timeout: telem.Second,
			Retries: 1,
			Host:    "plc",
			Port:    502,
			UnitID:  255,
		}
	}
	rtu := func() v0.RtuConnection {
		return v0.RtuConnection{
			Timeout:    telem.Second,
			Retries:    1,
			SerialPort: "/dev/ttyUSB0",
			BaudRate:   19200,
			Parity:     v0.ParityEven,
			StopBits:   v0.StopBitsOne,
			UnitID:     1,
		}
	}
	coil := func(address uint16) v0.Location {
		return v0.Location{Variant: v0.CoilLocation{
			BaseLocation: v0.BaseLocation{Address: address},
		}}
	}
	holding := func(address uint16, value v0.ValueVariant) v0.Location {
		return v0.Location{Variant: v0.HoldingRegisterLocation{
			BaseLocation: v0.BaseLocation{Address: address},
			Value:        v0.Value{Variant: value},
		}}
	}
	writeCoil := func(address uint16) v0.WriteLocation {
		return v0.WriteLocation{Variant: v0.CoilWriteLocation{Address: address}}
	}
	server := func() v0.Server {
		return v0.Server{
			Key:        uuid.MustParse("22222222-2222-4222-8222-222222222222"),
			Rack:       1,
			Name:       "PLC",
			Connection: v0.Connection{Variant: tcp()},
			RequestPolicy: v0.RequestPolicy{
				MaxReadRegisters:  125,
				MaxWriteRegisters: 123,
				MaxReadBits:       2000,
				MaxWriteBits:      1968,
				MaxGap:            125,
				WriteFunction:     v0.WriteFunctionMultiple,
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
			Expect(s.RequestPolicy).To(Equal(server().RequestPolicy))
			Expect(s.Validate()).To(Succeed())
		},
		Entry("TCP", `{"transport": "tcp", "host": "plc"}`, tcp()),
		Entry(
			"RTU",
			`{"transport": "rtu", "serial_port": "/dev/ttyUSB0"}`,
			rtu(),
		),
	)

	It("Should round trip a server through JSON", func() {
		s := server()
		s.Connection = v0.Connection{Variant: rtu()}
		s.Reads = []v0.Read{{
			Location: holding(5, v0.WholeValue{
				DataType:     telem.NumericDataTypeFloat32,
				WordsSwapped: new(true),
			}),
			Index:   3,
			Channel: 4,
		}, {
			Location: v0.Location{Variant: v0.InputRegisterLocation{
				BaseLocation: v0.BaseLocation{Address: 6},
				Value:        v0.Value{Variant: v0.BitValue{Bit: 3}},
			}},
			Index:   3,
			Channel: 5,
		}, {
			Location: holding(7, v0.StringValue{Length: 8}),
			Index:    3,
			Channel:  6,
		}}
		s.Writes = []v0.Write{{Location: writeCoil(1), Channel: 7}}
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
		Entry("port 0", func(s *v0.Server) {
			c := tcp()
			c.Port = 0
			s.Connection = v0.Connection{Variant: c}
		}, "port"),
		Entry("a timeout under a millisecond", func(s *v0.Server) {
			c := tcp()
			c.Timeout = telem.Microsecond
			s.Connection = v0.Connection{Variant: c}
		}, "timeout"),
		Entry("a timeout over a minute", func(s *v0.Server) {
			c := tcp()
			c.Timeout = 61 * telem.Second
			s.Connection = v0.Connection{Variant: c}
		}, "timeout"),
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
		Entry("a baud rate the Driver cannot set", func(s *v0.Server) {
			c := rtu()
			c.BaudRate = 14400
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
			s.RequestPolicy.MaxReadRegisters = 126
		}, "max_read_registers"),
		Entry("a write of 124 registers", func(s *v0.Server) {
			s.RequestPolicy.MaxWriteRegisters = 124
		}, "max_write_registers"),
		Entry("a read of 2001 bits", func(s *v0.Server) {
			s.RequestPolicy.MaxReadBits = 2001
		}, "max_read_bits"),
		Entry("a write of 1969 bits", func(s *v0.Server) {
			s.RequestPolicy.MaxWriteBits = 1969
		}, "max_write_bits"),
		Entry("a read of no registers", func(s *v0.Server) {
			s.RequestPolicy.MaxReadRegisters = 0
		}, "max_read_registers"),
		Entry("an unknown write function", func(s *v0.Server) {
			s.RequestPolicy.WriteFunction = "masked"
		}, "write_function"),
		Entry("a delay above five seconds", func(s *v0.Server) {
			s.RequestPolicy.Delay = 5*telem.Second + 1
		}, "delay"),
		Entry("a negative delay", func(s *v0.Server) {
			s.RequestPolicy.Delay = -1
		}, "delay"),
		Entry("a read with no index", func(s *v0.Server) {
			s.Reads = []v0.Read{{
				Location: coil(0),
				Channel:  1,
			}}
		}, "reads.0.index"),
		Entry("a read with no channel", func(s *v0.Server) {
			s.Reads = []v0.Read{
				{Location: coil(0), Index: 1},
			}
		}, "reads.0.channel"),
		Entry("a read with no table", func(s *v0.Server) {
			s.Reads = []v0.Read{{Index: 1, Channel: 2}}
		}, "reads.0.location.table"),
		Entry("a read with no value type", func(s *v0.Server) {
			s.Reads = []v0.Read{{
				Location: holding(0, nil),
				Index:    1,
				Channel:  2,
			}}
		}, "reads.0.location.value.type"),
		Entry("a read of bit 16", func(s *v0.Server) {
			s.Reads = []v0.Read{{
				Location: holding(0, v0.BitValue{Bit: 16}),
				Index:    1,
				Channel:  2,
			}}
		}, "reads.0.location.value.bit"),
		Entry("a read of an empty string", func(s *v0.Server) {
			s.Reads = []v0.Read{{
				Location: holding(0, v0.StringValue{}),
				Index:    1,
				Channel:  2,
			}}
		}, "reads.0.location.value.length"),
		Entry("a read of a string of 251 characters", func(s *v0.Server) {
			s.Reads = []v0.Read{{
				Location: holding(0, v0.StringValue{Length: 251}),
				Index:    1,
				Channel:  2,
			}}
		}, "reads.0.location.value.length"),
		Entry("a write of an unknown data type", func(s *v0.Server) {
			s.Writes = []v0.Write{{
				Location: v0.WriteLocation{
					Variant: v0.HoldingRegisterWriteLocation{
						Value: v0.WriteValue{
							Variant: v0.WholeWriteValue{
								Number: v0.Number{DataType: "bfloat16"},
							},
						},
					},
				},
				Channel: 1,
			}}
		}, "writes.0.location.value.data_type"),
		Entry("a write with no channel", func(s *v0.Server) {
			s.Writes = []v0.Write{{Location: writeCoil(0)}}
		}, "writes.0.channel"),
	)
})

var _ = Describe("Connection", func() {
	DescribeTable("Should round trip through msgpack",
		func(c v0.Connection) {
			var decoded v0.Connection
			Expect(msgpack.Unmarshal(MustSucceed(msgpack.Marshal(c)), &decoded)).
				To(Succeed())
			Expect(decoded).To(Equal(c))
		},
		Entry("TCP", v0.Connection{Variant: v0.TcpConnection{
			BytesSwapped: true,
			Timeout:      500 * telem.Millisecond,
			Host:         "plc",
			Port:         502,
			UnitID:       3,
		}}),
		Entry("RTU", v0.Connection{Variant: v0.RtuConnection{
			Timeout:    telem.Second,
			Retries:    2,
			SerialPort: "COM3",
			BaudRate:   9600,
			Parity:     v0.ParityNone,
			StopBits:   v0.StopBitsTwo,
			UnitID:     7,
		}}),
		Entry("no variant", v0.Connection{}),
	)

	It("Should apply the defaults of an omitted field", func() {
		raw := MustSucceed(msgpack.Marshal(map[string]any{
			"transport":   "rtu",
			"serial_port": "COM3",
		}))
		var decoded v0.Connection
		Expect(msgpack.Unmarshal(raw, &decoded)).To(Succeed())
		Expect(decoded.Variant).To(Equal(v0.RtuConnection{
			Timeout:    telem.Second,
			Retries:    1,
			SerialPort: "COM3",
			BaudRate:   19200,
			Parity:     v0.ParityEven,
			StopBits:   v0.StopBitsOne,
			UnitID:     1,
		}))
	})

	It("Should reject an unknown transport", func() {
		raw := MustSucceed(msgpack.Marshal(map[string]any{"transport": "udp"}))
		var decoded v0.Connection
		Expect(msgpack.Unmarshal(raw, &decoded)).
			To(MatchError(validate.ErrValidation))
	})
})

var _ = Describe("Location", func() {
	location := func(variant v0.LocationVariant) v0.Location {
		return v0.Location{Variant: variant}
	}
	DescribeTable("Should round trip through msgpack",
		func(r v0.Location) {
			var decoded v0.Location
			Expect(msgpack.Unmarshal(MustSucceed(msgpack.Marshal(r)), &decoded)).
				To(Succeed())
			Expect(decoded).To(Equal(r))
		},
		Entry("a coil", location(v0.CoilLocation{
			Address: 3,
		})),
		Entry("a discrete input", location(v0.DiscreteInputLocation{
			Address: 4,
		})),
		Entry("a whole value", location(v0.HoldingRegisterLocation{
			Address: 5,
			Value: v0.Value{Variant: v0.WholeValue{
				DataType:     v0.DataTypeFloat32,
				WordsSwapped: new(true),
			}},
		})),
		Entry("a bit", location(v0.InputRegisterLocation{
			Address: 6,
			Value:   v0.Value{Variant: v0.BitValue{Bit: 7}},
		})),
		Entry("a string", location(v0.InputRegisterLocation{
			Address: 7,
			Value:   v0.Value{Variant: v0.StringValue{Length: 8}},
		})),
		Entry("no variant", v0.Location{}),
	)

	It("Should apply the default data type of a whole value", func() {
		raw := MustSucceed(msgpack.Marshal(map[string]any{
			"table":   "holding_register",
			"address": 2,
			"value":   map[string]any{"type": "whole"},
		}))
		var decoded v0.Location
		Expect(msgpack.Unmarshal(raw, &decoded)).To(Succeed())
		Expect(decoded.Variant).To(Equal(v0.HoldingRegisterLocation{
			Address: 2,
			Value:   v0.Value{Variant: v0.WholeValue{DataType: v0.DataTypeUint16}},
		}))
	})

	DescribeTable("Should reject an unknown type",
		func(raw map[string]any) {
			var decoded v0.Location
			Expect(msgpack.Unmarshal(MustSucceed(msgpack.Marshal(raw)), &decoded)).
				To(MatchError(validate.ErrValidation))
		},
		Entry("location", map[string]any{"table": "file_record"}),
		Entry("value", map[string]any{
			"table": "input_register",
			"value": map[string]any{"type": "half"},
		}),
	)
})

var _ = Describe("WriteLocation", func() {
	DescribeTable("Should round trip through msgpack",
		func(l v0.WriteLocation) {
			var decoded v0.WriteLocation
			Expect(msgpack.Unmarshal(MustSucceed(msgpack.Marshal(l)), &decoded)).
				To(Succeed())
			Expect(decoded).To(Equal(l))
		},
		Entry("a coil", v0.WriteLocation{Variant: v0.CoilWriteLocation{Address: 3}}),
		Entry("a whole value", v0.WriteLocation{
			Variant: v0.HoldingRegisterWriteLocation{
				Address: 5,
				Value: v0.WriteValue{Variant: v0.WholeWriteValue{
					DataType:     v0.DataTypeFloat32,
					BytesSwapped: new(true),
				}},
			},
		}),
		Entry("a bit", v0.WriteLocation{Variant: v0.HoldingRegisterWriteLocation{
			Address: 6,
			Value:   v0.WriteValue{Variant: v0.BitWriteValue{Bit: 2}},
		}}),
		Entry("no variant", v0.WriteLocation{}),
	)

	DescribeTable("Should reject a location that write tasks cannot write",
		func(raw map[string]any) {
			var decoded v0.WriteLocation
			Expect(msgpack.Unmarshal(MustSucceed(msgpack.Marshal(raw)), &decoded)).
				To(MatchError(validate.ErrValidation))
		},
		Entry("a discrete input", map[string]any{"table": "discrete_input"}),
		Entry("a string", map[string]any{
			"table": "holding_register",
			"value": map[string]any{"type": "string", "length": 4},
		}),
	)
})
