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
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channeltestutil "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/modbus/server"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	node testutil.Node
	db   *gorp.DB
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node = testutil.NewNode(ctx)
	db = node.DB
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
		Location: server.Location{Variant: server.HoldingRegisterLocation{
			Address: address,
			Value: server.Value{
				Variant: server.WholeValue{DataType: telem.NumericDataTypeFloat32},
			},
		}},
		Index:   index,
		Channel: ch,
	}
}

// write binds the coil at address to channel.
func write(address uint16, ch channel.Key) server.Write {
	return server.Write{
		Location: server.WriteLocation{
			Variant: server.CoilWriteLocation{Address: address},
		},
		Channel: ch,
	}
}

// holdingWrite binds the value of the holding register at address to channel.
func holdingWrite(
	address uint16,
	value server.WriteValueVariant,
	ch channel.Key,
) server.Write {
	return server.Write{
		Location: server.WriteLocation{Variant: server.HoldingRegisterWriteLocation{
			Address: address,
			Value:   server.WriteValue{Variant: value},
		}},
		Channel: ch,
	}
}

// whole returns a write of a number of the data type.
func whole(dt telem.NumericDataType) server.WholeWriteValue {
	return server.WholeWriteValue{DataType: dt}
}

// bit returns a write of one bit of a register.
func bit(b uint8) server.BitWriteValue {
	return server.BitWriteValue{Bit: b}
}

// createChannels creates count virtual float64 channels in tx and returns their keys.
func createChannels(
	ctx context.Context,
	svc *channel.Service,
	tx gorp.Tx,
	count int,
) channel.Keys {
	GinkgoHelper()
	channels := make([]channel.Channel, count)
	for i := range channels {
		channels[i] = channel.Channel{
			Name:     channeltestutil.UniqueChannelName(),
			DataType: telem.Float64T,
			Virtual:  true,
		}
	}
	Expect(svc.NewWriter(tx).CreateMany(ctx, &channels)).To(Succeed())
	return channel.KeysFromChannels(channels)
}
