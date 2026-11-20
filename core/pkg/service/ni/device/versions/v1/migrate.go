// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"context"
	"strconv"
	"strings"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp/migrate"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v0 "github.com/synnaxlabs/synnax/pkg/service/ni/device/versions/v0"
	"go.uber.org/zap"
)

// Migration lifts stored NI devices from v0 to v1 and drops their DAQmx alias. A
// simulated device moves its serial, which held its NI-MAX resource name, to its
// resource name and gets an empty serial. A physical device keeps its serial and gets
// an empty resource name until the scanner writes its identity. Until then, the device
// fails validation, so a configuration update to it fails. Each port keeps its channels
// and takes the index of its old port group. The "boardTempSensor" analog input becomes
// the temperature channel. A port key that names no port is dropped.
var Migration = migrate.NewInstrumentedEntry("v62_ni_device_identity", migrateDevice)

func migrateDevice(
	ctx context.Context,
	old v0.Device,
	ins alamos.Instrumentation,
) (Device, error) {
	d, err := autoMigrateDevice(ctx, old)
	if err != nil {
		return d, err
	}
	if old.Simulated {
		d.ResourceName = old.Serial
		d.Serial = ""
	}
	drop := func(group, key string) {
		ins.L.Warn(
			"dropping NI device port that v1 cannot hold",
			zap.Stringer("key", old.Key),
			zap.String("group", group),
			zap.String("port", key),
		)
	}
	d.AnalogInput = migrateAnalogInput(old.AnalogInput, drop)
	d.AnalogOutput = migrateAnalogOutput(old.AnalogOutput, drop)
	d.CounterInput = migrateCounterInput(old.CounterInput, drop)
	d.DigitalInput = migrateDigitalInput(old.DigitalInput, drop)
	d.DigitalOutput = migrateDigitalOutput(old.DigitalOutput, drop)
	return d, nil
}

// dropFunc reports a v0 port key in a port group that v1 cannot hold.
type dropFunc func(group, key string)

func migrateAnalogInput(old v0.InputPorts, drop dropFunc) AnalogInputPorts {
	ports := AnalogInputPorts{Index: old.Index, Channels: map[uint16]channel.Key{}}
	for k, ch := range old.Channels {
		if k == "boardTempSensor" {
			ports.Temperature = ch
		} else if port, ok := parsePort(k); ok {
			ports.Channels[port] = ch
		} else {
			drop("analog_input", k)
		}
	}
	return ports
}

func migrateAnalogOutput(
	old v0.OutputPorts,
	drop dropFunc,
) map[uint16]AnalogOutputChannels {
	ports := map[uint16]AnalogOutputChannels{}
	for k, pair := range old.Channels {
		port, ok := parsePort(k)
		if !ok {
			drop("analog_output", k)
			continue
		}
		ports[port] = AnalogOutputChannels{
			StateIndex: old.StateIndex,
			Command:    pair.Command,
			State:      pair.State,
		}
	}
	return ports
}

func migrateCounterInput(
	old v0.InputPorts,
	drop dropFunc,
) map[uint16]CounterInputChannels {
	ports := map[uint16]CounterInputChannels{}
	for k, ch := range old.Channels {
		port, ok := parsePort(k)
		if !ok {
			drop("counter_input", k)
			continue
		}
		ports[port] = CounterInputChannels{Index: old.Index, Channel: ch}
	}
	return ports
}

func migrateDigitalInput(
	old v0.InputPorts,
	drop dropFunc,
) map[uint16]DigitalInputChannels {
	ports := map[uint16]DigitalInputChannels{}
	for k, ch := range old.Channels {
		port, line, ok := parseLine(k)
		if !ok {
			drop("digital_input", k)
			continue
		}
		p, exists := ports[port]
		if !exists {
			p = DigitalInputChannels{Index: old.Index, Lines: map[uint8]channel.Key{}}
		}
		p.Lines[line] = ch
		ports[port] = p
	}
	return ports
}

func migrateDigitalOutput(
	old v0.OutputPorts,
	drop dropFunc,
) map[uint16]DigitalOutputChannels {
	ports := map[uint16]DigitalOutputChannels{}
	for k, pair := range old.Channels {
		port, line, ok := parseLine(k)
		if !ok {
			drop("digital_output", k)
			continue
		}
		p, exists := ports[port]
		if !exists {
			p = DigitalOutputChannels{
				StateIndex: old.StateIndex,
				Lines:      map[uint8]CommandStatePair{},
			}
		}
		p.Lines[line] = pair
		ports[port] = p
	}
	return ports
}

func parsePort(key string) (uint16, bool) {
	port, err := strconv.ParseUint(key, 10, 16)
	return uint16(port), err == nil
}

// parseLine parses a v0 digital line key such as "0l7", which names port 0, line 7.
func parseLine(key string) (uint16, uint8, bool) {
	portKey, lineKey, found := strings.Cut(key, "l")
	if !found {
		return 0, 0, false
	}
	port, portOK := parsePort(portKey)
	line, err := strconv.ParseUint(lineKey, 10, 8)
	return port, uint8(line), portOK && err == nil
}
