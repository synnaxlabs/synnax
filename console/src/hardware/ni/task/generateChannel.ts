// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep, id } from "@synnaxlabs/x";

import {
  type AnalogChannel,
  type AnalogInputChannel,
  type AnalogOutputChannel,
  type DigitalChannel,
  type DigitalInputChannel,
  type DigitalOutputChannel,
  ZERO_ANALOG_INPUT_CHANNEL,
  ZERO_ANALOG_OUTPUT_CHANNEL,
  ZERO_DIGITAL_INPUT_CHANNEL,
  ZERO_DIGITAL_OUTPUT_CHANNEL,
} from "@/hardware/ni/task/types";

const generateDigitalChannel = <C extends DigitalChannel>(
  channels: C[],
  zeroChannel: C,
): C => {
  const line = Math.max(0, ...channels.map(({ line }) => line)) + 1;
  return { ...zeroChannel, key: id.id(), line };
};

export const generateDigitalInputChannel = (
  channels: DigitalInputChannel[],
): DigitalInputChannel => generateDigitalChannel(channels, ZERO_DIGITAL_INPUT_CHANNEL);

export const generateDigitalOutputChannel = (
  channels: DigitalOutputChannel[],
): DigitalOutputChannel =>
  generateDigitalChannel(channels, ZERO_DIGITAL_OUTPUT_CHANNEL);

const generateAnalogChannel = <C extends AnalogChannel>(
  channels: C[],
  index: number,
  zeroChannel: C,
): C => {
  const key = id.id();
  if (index === -1) return { ...deep.copy(zeroChannel), key };
  const existingPorts = new Set(channels.map(({ port }) => port));
  let port = 0;
  while (existingPorts.has(port)) port++;
  return { ...deep.copy(channels[index]), key, port };
};

export const generateAnalogInputChannel = (
  channels: AnalogInputChannel[],
  index: number,
): AnalogInputChannel =>
  generateAnalogChannel(channels, index, ZERO_ANALOG_INPUT_CHANNEL);

export const generateAnalogOutputChannel = (
  channels: AnalogOutputChannel[],
  index: number,
): AnalogOutputChannel =>
  generateAnalogChannel(channels, index, ZERO_ANALOG_OUTPUT_CHANNEL);
