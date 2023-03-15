import { useState } from "react";

import { Channel } from "@synnaxlabs/client";
import { useAsyncEffect } from "@synnaxlabs/pluto";

import { Channels } from "./channels";
import { Ranges } from "./ranges";

import {
  useTelemetryClient,
  TelemetryClient,
  TelemetryClientResponse,
} from "@/vis/telem";
import { AxisKey } from "@/vis/types";

const ZERO_DATA: InternalState = {
  y1: [],
  y2: [],
  y3: [],
  y4: [],
  x1: [],
  x2: [],
};

type InternalState = Record<AxisKey, TelemetryClientResponse[]>;

export class Data {
  private readonly entries: InternalState;
  readonly error: Error | null;

  constructor(entries: InternalState, error: Error | null) {
    this.entries = entries;
    this.error = error;
  }

  static zero(): Data {
    return new Data(ZERO_DATA, null);
  }

  static use(channels: Channels, ranges: Ranges): Data {
    const client = useTelemetryClient();
    const [data, setData] = useState<Data>(Data.zero());

    useAsyncEffect(async () => {
      if (client === null) return;
      const data = await Data.fetch(channels, ranges, client);
      setData(data);
    }, [channels]);

    return data;
  }

  static async fetch(
    channels: Channels,
    ranges: Ranges,
    client: TelemetryClient
  ): Promise<Data> {
    let entries: TelemetryClientResponse[] = [];
    let error: Error | null = null;
    try {
      entries = await client.retrieve({
        keys: channels.keys,
        ranges: ranges.array,
        bypassCache: ranges.isLive,
      });
    } catch (err) {
      error = err as Error;
    }
    const core = { ...ZERO_DATA };
    ranges.forEach((range) =>
      channels.forEachAxis((channels, axis) => {
        const keys = channels.map((c) => c.key);
        core[axis].push(
          ...entries.filter((e) => keys.includes(e.key) && e.range === range)
        );
      })
    );
    return new Data(core, error);
  }

  axis(key: AxisKey): TelemetryClientResponse[] {
    return this.entries[key];
  }

  forEachAxis(fn: (key: AxisKey, data: TelemetryClientResponse[]) => void): void {
    Object.entries(this.entries).forEach(([key, data]) => fn(key as AxisKey, data));
  }

  forEachChannel(
    fn: (ch: string, axis: AxisKey, data: TelemetryClientResponse[]) => void
  ): void {
    Object.entries(this.entries).forEach(([axis, data]) => {
      const keys = new Set(data.map((d) => d.key));
      keys.forEach((key) =>
        fn(
          key,
          axis as AxisKey,
          data.filter((d) => d.key === key)
        )
      );
    });
  }

  valid(): boolean {
    return this.error === null;
  }
}
