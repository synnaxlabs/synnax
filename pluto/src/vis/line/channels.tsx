// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Channel, ChannelKey, Synnax } from "@synnaxlabs/client";
import { Deep } from "@synnaxlabs/x";

import {
  AxisKey,
  AXIS_KEYS,
  XAxisKey,
  X_AXIS_KEYS,
  YAxisKey,
  Y_AXIS_KEYS,
} from "@/vis/axis";
import {
  ChannelsState,
  GOOD_STATUS,
  Status,
  StatusProvider,
  ZERO_CHANNELS_STATE,
} from "@/vis/line/core";

export class Channels implements StatusProvider {
  readonly core: ChannelsState;
  readonly channels: Channel[];
  readonly status: Status;

  constructor(core: ChannelsState, channels: Channel[], status: Status) {
    this.core = core;
    this.channels = channels;
    this.status = status;
  }

  static async use(client: Synnax, state: ChannelsState): Promise<Channels> {
    const channels = await client.channels.retrieve(...Channels.toKeys(state));
    return new Channels(state, channels, GOOD_STATUS);
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

  get keys(): readonly ChannelKey[] {
    return Channels.toKeys(this.core);
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
      const v = this.core[key];
      if (Array.isArray(v)) return v.includes(c.key);
      return v === c.key;
    });
  }

  yAxisKeys(key: YAxisKey): readonly ChannelKey[] {
    return this.core[key];
  }

  xAxisKey(key: XAxisKey): ChannelKey {
    return this.core[key];
  }

  forEach(callback: (channel: Channel, axes: AxisKey[]) => void): void {
    this.channels.forEach((channel) => {
      const axes = AXIS_KEYS.filter((axis) => {
        const v = this.core[axis];
        if (Array.isArray(v)) return v.includes(channel.key);
        return v === channel.key;
      });
      callback(channel, axes);
    });
  }

  forEachAxis(callback: (channels: Channel[], axis: AxisKey) => void): void {
    AXIS_KEYS.forEach((axis) => callback(this.axis(axis), axis));
  }

  static zero(): Channels {
    return new Channels(Deep.copy(ZERO_CHANNELS_STATE), [], GOOD_STATUS);
  }

  private static toKeys(core: ChannelsState): readonly ChannelKey[] {
    return Object.values(core)
      .flat()
      .filter((key) => key != null && key !== 0);
  }
}
