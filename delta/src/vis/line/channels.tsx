import { useState } from "react";

import { Channel } from "@synnaxlabs/client";
import { useAsyncEffect } from "@synnaxlabs/pluto";

import { useClusterClient } from "@/cluster";
import { useMemoSelect } from "@/hooks";
import { LayoutStoreState } from "@/layout";
import { AxisKey, AXIS_KEYS, XAxisKey, YAxisKey, Y_AXIS_KEYS } from "@/vis/axis";
import { ChannelsState, LineVis, ZERO_CHANNELS_STATE } from "@/vis/line/core";
import { selectRequiredVis, VisualizationStoreState } from "@/vis/store";

// This is what lives in redux
export class Channels {
  readonly core: ChannelsState;
  readonly channels: Channel[];

  constructor(core: ChannelsState, channels: Channel[]) {
    this.core = core;
    this.channels = channels;
  }

  static zero(): Channels {
    return new Channels({ ...ZERO_CHANNELS_STATE }, []);
  }

  static use(key: string): Channels {
    const [channels, setChannels] = useState<Channels>(Channels.zero());
    const client = useClusterClient();

    const core = useMemoSelect(
      (state: VisualizationStoreState & LayoutStoreState) =>
        selectRequiredVis<LineVis>(state, key, "line").channels,
      [key]
    );

    useAsyncEffect(async () => {
      if (client === null) return;
      const keys = Channels.toKeys(core);
      if (keys.length === 0) return;
      const channels = await client.channels.retrieve(keys);
      console.log(keys, channels);
      setChannels(new Channels(core, channels));
    }, [client, core]);

    return channels;
  }

  private static toKeys(core: ChannelsState): string[] {
    return Object.values(core)
      .flat()
      .filter((key) => key != null && key !== "");
  }

  get valid(): boolean {
    // assert that we have at least one x channel and one y channel
    return (
      Y_AXIS_KEYS.some((axis) => this.core[axis].length > 0) &&
      Y_AXIS_KEYS.some((axis) => {
        const v = this.core[axis];
        return v != null && v.length > 0;
      })
    );
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
