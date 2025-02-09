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
  type AIChannel,
  type AnalogChannel,
  type AOChannel,
  type DIChannel,
  type DigitalChannel,
  type DOChannel,
  ZERO_AI_CHANNEL,
  ZERO_AO_CHANNEL,
  ZERO_DI_CHANNEL,
  ZERO_DO_CHANNEL,
} from "@/hardware/ni/task/types";

const generateDigitalChannel = <C extends DigitalChannel>(
  channels: C[],
  zeroChannel: C,
): C => {
  const line = Math.max(0, ...channels.map(({ line }) => line)) + 1;
  return { ...zeroChannel, key: id.id(), line };
};

export const generateDIChannel = (channels: DIChannel[]): DIChannel =>
  generateDigitalChannel(channels, ZERO_DI_CHANNEL);

export const generateDOChannel = (channels: DOChannel[]): DOChannel =>
  generateDigitalChannel(channels, ZERO_DO_CHANNEL);

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

export const generateAIChannel = (channels: AIChannel[], index: number): AIChannel =>
  generateAnalogChannel(channels, index, ZERO_AI_CHANNEL);

export const generateAOChannel = (channels: AOChannel[], index: number): AOChannel =>
  generateAnalogChannel(channels, index, ZERO_AO_CHANNEL);
