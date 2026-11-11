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
	d.AnalogInput = AnalogInputPorts{
		Index:    old.AnalogInput.Index,
		Channels: map[uint16]channel.Key{},
	}
	for k, ch := range old.AnalogInput.Channels {
		if k == "boardTempSensor" {
			d.AnalogInput.Temperature = ch
		} else if port, ok := parsePort(k); ok {
			d.AnalogInput.Channels[port] = ch
		} else {
			drop("analog_input", k)
		}
	}
	d.AnalogOutput = map[uint16]AnalogOutputChannels{}
	for k, pair := range old.AnalogOutput.Channels {
		port, ok := parsePort(k)
		if !ok {
			drop("analog_output", k)
			continue
		}
		d.AnalogOutput[port] = AnalogOutputChannels{
			StateIndex: old.AnalogOutput.StateIndex,
			Command:    pair.Command,
			State:      pair.State,
		}
	}
	d.CounterInput = map[uint16]CounterInputChannels{}
	for k, ch := range old.CounterInput.Channels {
		port, ok := parsePort(k)
		if !ok {
			drop("counter_input", k)
			continue
		}
		d.CounterInput[port] = CounterInputChannels{
			Index:   old.CounterInput.Index,
			Channel: ch,
		}
	}
	d.DigitalInput = map[uint16]DigitalInputChannels{}
	for k, ch := range old.DigitalInput.Channels {
		port, line, ok := parseLine(k)
		if !ok {
			drop("digital_input", k)
			continue
		}
		p, exists := d.DigitalInput[port]
		if !exists {
			p = DigitalInputChannels{
				Index: old.DigitalInput.Index,
				Lines: map[uint8]channel.Key{},
			}
		}
		p.Lines[line] = ch
		d.DigitalInput[port] = p
	}
	d.DigitalOutput = map[uint16]DigitalOutputChannels{}
	for k, pair := range old.DigitalOutput.Channels {
		port, line, ok := parseLine(k)
		if !ok {
			drop("digital_output", k)
			continue
		}
		p, exists := d.DigitalOutput[port]
		if !exists {
			p = DigitalOutputChannels{
				StateIndex: old.DigitalOutput.StateIndex,
				Lines:      map[uint8]CommandStatePair{},
			}
		}
		p.Lines[line] = pair
		d.DigitalOutput[port] = p
	}
	return d, nil
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
