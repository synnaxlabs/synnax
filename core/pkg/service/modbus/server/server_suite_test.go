// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package server_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/modbus/server"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var db *gorp.DB

var _ = BeforeSuite(func() {
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	StartRun(db.Run)
})

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Modbus Server Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

// tcp returns a TCP connection to host on the Modbus port.
func tcp(host string) server.Connection {
	return server.Connection{Variant: server.TcpConnection{
		Timeout: telem.Second,
		Retries: 1,
		Host:    host,
		Port:    502,
		UnitID:  255,
	}}
}

// rtu returns an RTU connection to a unit on /dev/ttyUSB0 at the baud rate.
func rtu(baud server.BaudRate) server.Connection {
	return server.Connection{Variant: server.RtuConnection{
		Timeout:    telem.Second,
		Retries:    1,
		SerialPort: "/dev/ttyUSB0",
		BaudRate:   baud,
		Parity:     server.ParityEven,
		StopBits:   server.StopBitsOne,
		UnitID:     1,
	}}
}

// requestPolicy returns the schema defaults of the request policy of a server.
func requestPolicy() server.RequestPolicy {
	return server.RequestPolicy{
		MaxReadRegisters:  125,
		MaxWriteRegisters: 123,
		MaxReadBits:       2000,
		MaxWriteBits:      1968,
		MaxGap:            125,
		WriteFunction:     server.WriteFunctionMultiple,
	}
}

// read binds a float32 holding register at address to channel under index.
func read(address uint16, index, ch channel.Key) server.Read {
	return server.Read{
		Register: server.Register{Variant: server.HoldingRegisterRegister{
			Address: address,
			Value: server.Value{
				Variant: server.WholeValue{DataType: server.DataTypeFloat32},
			},
		}},
		Index:   index,
		Channel: ch,
	}
}

// write binds the coil at address to channel.
func write(address uint16, ch channel.Key) server.Write {
	return server.Write{
		Register: server.Register{Variant: server.CoilRegister{Address: address}},
		Channel:  ch,
	}
}
