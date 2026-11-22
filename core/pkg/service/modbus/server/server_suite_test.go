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
func rtu(baud uint32) server.Connection {
	return server.Connection{Variant: server.RtuConnection{
		TimeoutMs:  1000,
		Retries:    1,
		SerialPort: "/dev/ttyUSB0",
		BaudRate:   baud,
		Parity:     server.ParityEven,
		StopBits:   1,
		UnitID:     1,
	}}
}

// requests returns the schema defaults of the requests of a server.
func requests() server.Requests {
	return server.Requests{
		MaxRegisters: 125,
		MaxGap:       125,
		WriteCodes:   server.WriteCodesMultiple,
	}
}
