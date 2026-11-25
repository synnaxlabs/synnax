// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"context"

	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/ni/device/versions/v1"
)

// presentKey returns nil for the 0 that v1 stored for an unset channel.
func presentKey(key channel.Key) *channel.Key {
	if key == 0 {
		return nil
	}
	return &key
}

// MigrateAnalogInputPorts migrates v1 analog input ports, leaving unset channels
// absent.
func MigrateAnalogInputPorts(
	ctx context.Context,
	old v1.AnalogInputPorts,
) (AnalogInputPorts, error) {
	ports, err := autoMigrateAnalogInputPorts(ctx, old)
	if err != nil {
		return AnalogInputPorts{}, err
	}
	ports.Index = presentKey(old.Index)
	ports.Temperature = presentKey(old.Temperature)
	return ports, nil
}

// MigrateAnalogOutputChannels migrates the channels of a v1 analog output port,
// leaving unset channels absent.
func MigrateAnalogOutputChannels(
	ctx context.Context,
	old v1.AnalogOutputChannels,
) (AnalogOutputChannels, error) {
	channels, err := autoMigrateAnalogOutputChannels(ctx, old)
	if err != nil {
		return AnalogOutputChannels{}, err
	}
	channels.StateIndex = presentKey(old.StateIndex)
	channels.Command = presentKey(old.Command)
	channels.State = presentKey(old.State)
	return channels, nil
}

// MigrateCounterInputChannels migrates the channels of a v1 counter input, leaving
// unset channels absent.
func MigrateCounterInputChannels(
	ctx context.Context,
	old v1.CounterInputChannels,
) (CounterInputChannels, error) {
	channels, err := autoMigrateCounterInputChannels(ctx, old)
	if err != nil {
		return CounterInputChannels{}, err
	}
	channels.Index = presentKey(old.Index)
	channels.Channel = presentKey(old.Channel)
	return channels, nil
}

// MigrateDigitalInputChannels migrates the channels of a v1 digital input port,
// leaving an unset index channel absent.
func MigrateDigitalInputChannels(
	ctx context.Context,
	old v1.DigitalInputChannels,
) (DigitalInputChannels, error) {
	channels, err := autoMigrateDigitalInputChannels(ctx, old)
	if err != nil {
		return DigitalInputChannels{}, err
	}
	channels.Index = presentKey(old.Index)
	return channels, nil
}

// MigrateDigitalOutputChannels migrates the channels of a v1 digital output port,
// leaving an unset state index channel absent.
func MigrateDigitalOutputChannels(
	ctx context.Context,
	old v1.DigitalOutputChannels,
) (DigitalOutputChannels, error) {
	channels, err := autoMigrateDigitalOutputChannels(ctx, old)
	if err != nil {
		return DigitalOutputChannels{}, err
	}
	channels.StateIndex = presentKey(old.StateIndex)
	return channels, nil
}

// MigrateCommandStatePair migrates a v1 command and state pair, leaving unset channels
// absent.
func MigrateCommandStatePair(
	ctx context.Context,
	old v1.CommandStatePair,
) (CommandStatePair, error) {
	pair, err := autoMigrateCommandStatePair(ctx, old)
	if err != nil {
		return CommandStatePair{}, err
	}
	pair.Command = presentKey(old.Command)
	pair.State = presentKey(old.State)
	return pair, nil
}
