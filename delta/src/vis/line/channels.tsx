import { useState } from "react";

import { Channel } from "@synnaxlabs/client";
import { useAsyncEffect } from "@synnaxlabs/pluto";
import { Deep } from "@synnaxlabs/x";

import { useClusterClient } from "@/cluster";
import { useMemoSelect } from "@/hooks";
import { LayoutStoreState } from "@/layout";
import { AxisKey, AXIS_KEYS, XAxisKey, YAxisKey, Y_AXIS_KEYS } from "@/vis/axis";
import {
  ChannelsState,
  GOOD_STATUS,
  LineVis,
  Status,
  StatusProvider,
  ZERO_CHANNELS_STATE,
} from "@/vis/line/core";
import { selectRequiredVis, VisStoreState } from "@/vis/store";

export class Channels implements StatusProvider {
  readonly core: ChannelsState;
  readonly channels: Channel[];
  readonly status: Status;

  constructor(core: ChannelsState, channels: Channel[], status: Status) {
    this.core = core;
    this.channels = channels;
    this.status = status;
  }

  static zero(): Channels {
    return new Channels(Deep.copy(ZERO_CHANNELS_STATE), [], GOOD_STATUS);
  }

  static use(key: string): Channels {
    const [channels, setChannels] = useState<Channels>(Channels.zero());
    const client = useClusterClient();

    const core = useMemoSelect(
      (state: VisStoreState & LayoutStoreState) =>
        selectRequiredVis<LineVis>(state, key, "line").channels,
      [key]
    );

    const valid =
      Y_AXIS_KEYS.some((axis) => core[axis].length > 0) &&
      Y_AXIS_KEYS.some((axis) => {
        const v = core[axis];
        return v != null && v.length > 0;
      });

    useAsyncEffect(async () => {
      if (client === null) return;
      if (!valid)
        return setChannels(
          new Channels(core, [], {
            display: true,
            children: "Invalid Visualization",
            variant: "info",
          })
        );
      const keys = Channels.toKeys(core);
      const channels = await client.channels.retrieve(keys);
      setChannels(new Channels(core, channels, GOOD_STATUS));
    }, [client, core]);

    return channels;
  }

  private static toKeys(core: ChannelsState): string[] {
    return Object.values(core)
      .flat()
      .filter((key) => key != null && key !== "");
  }

  get keys(): string[] {
    return Channels.toKeys(this.core);
  }

  get(key: string): Channel | undefined {
    return this.channels.find((c) => c.key === key);
  }

  getRequired(key: string): Channel {
    const channel = this.get(key);
    if (channel === undefined) throw new Error(`Channel ${key} not found`);
    return channel;
  }

  axis(key: AxisKey): Channel[] {
    return this.channels.filter((c) => this.core[key].includes(c.key));
  }

  yAxisKeys(key: YAxisKey): readonly string[] {
    return this.core[key];
  }

  xAxisKey(key: XAxisKey): string {
    return this.core[key];
  }

  forEach(callback: (channel: Channel, axes: AxisKey[]) => void): void {
    this.channels.forEach((channel) => {
      const axes = AXIS_KEYS.filter((axis) => this.core[axis].includes(channel.key));
      callback(channel, axes);
    });
  }

  forEachAxis(callback: (channels: Channel[], axis: AxisKey) => void): void {
    AXIS_KEYS.forEach((axis) => callback(this.axis(axis), axis));
  }
}
