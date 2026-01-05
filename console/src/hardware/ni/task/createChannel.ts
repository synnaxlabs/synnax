// Copyright 2026 Synnax Labs, Inc.
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
  type CIChannel,
  type DIChannel,
  type DigitalChannel,
  type DOChannel,
  ZERO_AI_CHANNEL,
  ZERO_AO_CHANNEL,
  ZERO_CI_CHANNEL,
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
  zeroChannel: C,
  override: Partial<C>,
  keyToCopy?: string,
): C => {
  const key = id.create();
  let template: C;
  if (channels.length === 0) template = deep.copy(zeroChannel);
  else if (keyToCopy == null) template = deep.copy(channels[0]);
  else {
    const channel = channels.find(({ key }) => key === keyToCopy);
    if (channel == null) return { ...deep.copy(zeroChannel), key };
    template = deep.copy(channel);
  }
  const existingPorts = new Set(channels.map(({ port }) => port));
  let port = 0;
  while (existingPorts.has(port)) port++;
  return { ...template, key, port, ...override };
};

export const createAIChannel = (channels: AIChannel[], key?: string): AIChannel =>
  createAnalogChannel(
    channels,
    ZERO_AI_CHANNEL,
    Common.Task.READ_CHANNEL_OVERRIDE,
    key,
  );

export const createAOChannel = (channels: AOChannel[], key?: string): AOChannel =>
  createAnalogChannel(
    channels,
    ZERO_AO_CHANNEL,
    Common.Task.WRITE_CHANNEL_OVERRIDE,
    key,
  );

const createCounterChannel = <C extends CIChannel>(
  channels: C[],
  zeroChannel: C,
  override: Partial<C>,
  keyToCopy?: string,
): C => {
  const key = id.create();
  let template: C;
  if (channels.length === 0) template = deep.copy(zeroChannel);
  else if (keyToCopy == null) template = deep.copy(channels[0]);
  else {
    const channel = channels.find(({ key }) => key === keyToCopy);
    if (channel == null) return { ...deep.copy(zeroChannel), key };
    template = deep.copy(channel);
  }
  const existingPorts = new Set(channels.map(({ port }) => port));
  let port = 0;
  while (existingPorts.has(port)) port++;
  return { ...template, key, port, ...override };
};

export const createCIChannel = (channels: CIChannel[], key?: string): CIChannel =>
  createCounterChannel(
    channels,
    ZERO_CI_CHANNEL,
    Common.Task.READ_CHANNEL_OVERRIDE,
    key,
  );
