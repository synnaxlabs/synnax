// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Channel, ChannelKey, ChannelKeys } from "@synnaxlabs/client";
import { Deep, unique } from "@synnaxlabs/x";

import {
  AxisKey,
  AXIS_KEYS,
  XAxisKey,
  X_AXIS_KEYS,
  YAxisKey,
  Y_AXIS_KEYS,
  XAxisRecord,
  YAxisRecord,
} from "@/core/vis/Axis";
import { VisBuilderContext } from "@/vis/context";

export type ChannelsState = XAxisRecord<ChannelKey> &
  YAxisRecord<readonly ChannelKey[]>;

export const ZERO_CHANNELS_STATE = {
  x1: 0,
  x2: 0,
  y1: [] as readonly ChannelKey[],
  y2: [] as readonly ChannelKey[],
  y3: [] as readonly ChannelKey[],
  y4: [] as readonly ChannelKey[],
};

export class Channels {
  private state: ChannelsState;
  channels: Channel[];

  constructor() {
    this.state = Deep.copy(ZERO_CHANNELS_STATE);
    this.channels = [];
  }

  static zeroState(): ChannelsState {
    return Deep.copy(ZERO_CHANNELS_STATE);
  }

  update(state: ChannelsState): void {
    this.state = state;
  }

  async build(ctx: VisBuilderContext): Promise<void> {
    const keysToFetch = this.uniqueKeys;
    const channels = await ctx.client.core.channels.retrieve(keysToFetch);
    if (channels.length !== keysToFetch.length)
      throw new Error(
        `Failed to fetch all channels. Expected ${keysToFetch.length}, got ${channels.length}`
      );
    this.channels = channels;
  }

  static isValid(core: ChannelsState): boolean {
    return (
      Y_AXIS_KEYS.some((axis) => core[axis].length > 0) &&
      X_AXIS_KEYS.some((axis) => {
        const v = core[axis];
        return v != null && v !== 0;
      })
    );
  }

  get(key: ChannelKey): Channel | undefined {
    return this.channels.find((c) => c.key === key);
  }

  getRequired(key: number): Channel {
    const channel = this.get(key);
    if (channel === undefined) throw new Error(`Channel ${key} not found`);
    return channel;
  }

  axis(key: AxisKey): Channel[] {
    return this.channels.filter((c) => {
      const v = this.state[key];
      if (Array.isArray(v)) return v.includes(c.key);
      return v === c.key;
    });
  }

  yAxisKeys(key: YAxisKey): readonly ChannelKey[] {
    return this.state[key];
  }

  xAxisKey(key: XAxisKey): ChannelKey {
    return this.state[key];
  }

  forEach(callback: (channel: Channel, axes: AxisKey[]) => void): void {
    this.channels.forEach((channel) => {
      const axes = AXIS_KEYS.filter((axis) => {
        const v = this.state[axis];
        if (Array.isArray(v)) return v.includes(channel.key);
        return v === channel.key;
      });
      callback(channel, axes);
    });
  }

  forEachAxis(callback: (channels: Channel[], axis: AxisKey) => void): void {
    AXIS_KEYS.forEach((axis) => callback(this.axis(axis), axis));
  }

  get uniqueKeys(): ChannelKeys {
    return unique(
      Object.values(this.state)
        .flat()
        .filter((k) => k !== 0)
    );
  }
}
