import { Channel } from "@synnaxlabs/client";

import { LineSVis } from "..";

import { TelemetryClient, TelemetryClientResponse } from "@/features/vis/telem/client";
import { AxisKey } from "@/features/vis/types";

const ZERO_DATA: InternalDataState = {
  y1: [],
  y2: [],
  y3: [],
  y4: [],
  x1: [],
  x2: [],
};

type InternalDataState = Record<AxisKey, TelemetryClientResponse[]>;

export class Data {
  private readonly entries: InternalDataState;
  private readonly channels: Record<string, Channel>;
  readonly error: Error | null;

  constructor(entries: InternalDataState, channels: Channel[], error: Error | null) {
    this.entries = entries;
    this.error = error;
    this.channels = Object.fromEntries(channels.map((c) => [c.key, c]));
  }

  static initial(): Data {
    return new Data({ ...ZERO_DATA }, [], null);
  }

  static async fetch(
    client: TelemetryClient,
    vis: LineSVis,
    isLive: boolean
  ): Promise<Data> {
    const keys = Object.values(vis.channels)
      .flat()
      .filter((key) => key.length > 0);
    const ranges = Object.values(vis.ranges).flat();
    let entries: TelemetryClientResponse[] = [];
    let error: Error | null = null;
    let channels: Channel[] = [];
    try {
      entries = await client.retrieve({
        keys,
        ranges,
        bypassCache: isLive,
      });
      channels = await client.retrieveChannels(keys);
    } catch (err) {
      error = err as Error;
    }
    const data = { ...ZERO_DATA };
    Object.values(vis.ranges).forEach((ranges) =>
      ranges.forEach((range) =>
        Object.entries(vis.channels).forEach(([axis, channelKeys]) => {
          if (!Array.isArray(channelKeys)) channelKeys = [channelKeys as string];
          data[axis as AxisKey] = data[axis as AxisKey].concat(
            entries.filter(
              ({ key, range: r }) => channelKeys.includes(key) && r === range
            )
          );
        })
      )
    );
    return new Data(data, channels, error);
  }

  axis(key: AxisKey): TelemetryClientResponse[] {
    return this.entries[key];
  }

  forEachAxis(fn: (key: AxisKey, data: TelemetryClientResponse[]) => void): void {
    Object.entries(this.entries).forEach(([key, data]) => fn(key as AxisKey, data));
  }

  forEachChannel(
    fn: (ch: Channel, axis: AxisKey, data: TelemetryClientResponse[]) => void
  ): void {
    Object.entries(this.entries).forEach(([axis, data]) => {
      const keys = new Set(data.map((d) => d.key));
      keys.forEach((key) =>
        fn(
          this.channels[key],
          axis as AxisKey,
          data.filter((d) => d.key === key)
        )
      );
    });
  }
}
