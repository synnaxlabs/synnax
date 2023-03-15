import { useState } from "react";

import { Channel } from "@synnaxlabs/client";
import { useAsyncEffect } from "@synnaxlabs/pluto";

import { LineVis } from "./core";

import { useClusterClient } from "@/cluster";
import { useMemoSelect } from "@/hooks";
import { LayoutStoreState } from "@/layout";
import { selectRequiredVis, VisualizationStoreState } from "@/vis/store";
import { AxisKey, AXIS_KEYS, XAxisRecord, YAxisRecord } from "@/vis/types";

// This is what lives in redux
export type ChannelsCoreState = XAxisRecord<string> & YAxisRecord<readonly string[]>;

export const ZERO_CORE_CHANNELS_STATE = {
  x1: "",
  x2: "",
  y1: [] as readonly string[],
  y2: [] as readonly string[],
  y3: [] as readonly string[],
  y4: [] as readonly string[],
};

export class Channels {
  readonly core: ChannelsCoreState;
  readonly channels: Channel[];

  constructor(core: ChannelsCoreState, channels: Channel[]) {
    this.core = core;
    this.channels = channels;
  }

  static zero(): Channels {
    return new Channels(ZERO_CORE_CHANNELS_STATE, []);
  }

  static use(key: string): Channels {
    const [channels, setChannels] = useState<Channels>(Channels.zero());
    const client = useClusterClient();

    const core = useMemoSelect(
      (state: VisualizationStoreState & LayoutStoreState) =>
        selectRequiredVis<LineVis>(state, "line", key).channels,
      [key]
    );

    useAsyncEffect(async () => {
      if (client === null) return;
      const channels = await client.channels.retrieve(Channels.toKeys(core));
      setChannels(new Channels(core, channels));
    }, [core]);

    return channels;
  }

  private static toKeys(core: ChannelsCoreState): string[] {
    return Object.values(core).flat();
  }

  get keys(): string[] {
    return Channels.toKeys(this.core);
  }

  axis(key: AxisKey): Channel[] {
    return this.channels.filter((c) => this.core[key].includes(c.key));
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
