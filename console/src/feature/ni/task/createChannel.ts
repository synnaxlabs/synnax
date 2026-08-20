// Copyright 2026 Synnax Labs, Inc.
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
  type CIChannel,
  createAIChannel,
  createAOChannel,
  createCIChannel,
  createDIChannel,
  createDOChannel,
  type DIChannel,
  type DigitalChannel,
  type DOChannel,
} from "@/feature/ni/task/types";
import { Task } from "@/platform/task";

const createDigitalChannel = <C extends DigitalChannel>(
  channels: C[],
  zeroChannel: C,
): C => {
  const line = channels.length ? Math.max(...channels.map(({ line }) => line)) + 1 : 0;
  return { ...zeroChannel, key: id.create(), line };
};

export const createNextDIChannel = (channels: DIChannel[]): DIChannel =>
  createDigitalChannel<DIChannel>(channels, createDIChannel());

export const createNextDOChannel = (channels: DOChannel[]): DOChannel =>
  createDigitalChannel<DOChannel>(channels, createDOChannel());

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
  return { ...template, key, ...nextPort(channels, template), ...override };
};

// nextPort assigns the lowest port no channel is using. A channel type with no port,
// such as a built-in sensor, gets none.
const nextPort = <C extends AnalogChannel>(
  channels: C[],
  template: C,
): { port?: number } => {
  if (!("port" in template)) return {};
  const existing = new Set(channels.flatMap((c) => ("port" in c ? [c.port] : [])));
  let port = 0;
  while (existing.has(port)) port++;
  return { port };
};

export const createNextAIChannel = (channels: AIChannel[], key?: string): AIChannel =>
  createAnalogChannel(channels, createAIChannel(), Task.READ_CHANNEL_OVERRIDE, key);

export const createNextAOChannel = (channels: AOChannel[], key?: string): AOChannel =>
  createAnalogChannel(channels, createAOChannel(), Task.WRITE_CHANNEL_OVERRIDE, key);

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

export const createNextCIChannel = (channels: CIChannel[], key?: string): CIChannel =>
  createCounterChannel(channels, createCIChannel(), Task.READ_CHANNEL_OVERRIDE, key);
