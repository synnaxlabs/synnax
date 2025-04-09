// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep, id } from "@synnaxlabs/x";

import { Common } from "@/hardware/common";
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

const createDigitalChannel = <C extends DigitalChannel>(
  channels: C[],
  zeroChannel: C,
): C => {
  const line = channels.length ? Math.max(...channels.map(({ line }) => line)) + 1 : 0;
  return { ...zeroChannel, key: id.create(), line };
};

export const createDIChannel = (channels: DIChannel[]): DIChannel =>
  createDigitalChannel<DIChannel>(channels, ZERO_DI_CHANNEL);

export const createDOChannel = (channels: DOChannel[]): DOChannel =>
  createDigitalChannel<DOChannel>(channels, ZERO_DO_CHANNEL);

const createAnalogChannel = <C extends AnalogChannel>(
  channels: C[],
  index: number,
  zeroChannel: C,
  override: Partial<C>,
): C => {
  const key = id.create();
  let template: C;
  if (channels.length === 0) template = deep.copy(zeroChannel);
  else if (index === -1) template = deep.copy(channels[0]);
  else template = deep.copy(channels[index]);
  const existingPorts = new Set(channels.map(({ port }) => port));
  let port = 0;
  while (existingPorts.has(port)) port++;
  return { ...template, key, port, ...override };
};

export const createAIChannel = (channels: AIChannel[], index: number): AIChannel =>
  createAnalogChannel(
    channels,
    index,
    ZERO_AI_CHANNEL,
    Common.Task.READ_CHANNEL_OVERRIDE,
  );

export const createAOChannel = (channels: AOChannel[], index: number): AOChannel =>
  createAnalogChannel(
    channels,
    index,
    ZERO_AO_CHANNEL,
    Common.Task.WRITE_CHANNEL_OVERRIDE,
  );
