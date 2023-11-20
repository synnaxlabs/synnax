// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import {
  bounds,
  TimeRange,
  type Destructor,
  type Series,
  TimeSpan,
  TimeStamp,
  DataType,
} from "@synnaxlabs/x";
import { z } from "zod";

import { type client } from "@/telem/client";
import { telem } from "@/telem/core";

const fetchChannel = async (
  client: client.ChannelClient,
  channel: channel.Key,
  index: boolean,
): Promise<channel.Channel> => {
  if (!index) return await client.retrieveChannel(channel);
  const c = await client.retrieveChannel(channel);
  return await client.retrieveChannel(c.index);
};

const seriesSourcePropsZ = z.object({
  timeRange: TimeRange.z,
  channel: z.number(),
  index: z.boolean().optional().default(false),
});

export type SeriesSourceProps = z.input<typeof seriesSourcePropsZ>;

export class SeriesSource
  extends telem.AbstractSource<typeof seriesSourcePropsZ>
  implements telem.SeriesSource
{
  static readonly TYPE = "series-source";
  private readonly client: client.ReadClient & client.ChannelClient;
  private data: Series[] = [];
  valid: boolean = false;

  constructor(client: client.ReadClient & client.ChannelClient, props: unknown) {
    super(props);
    this.client = client;
  }

  cleanup(): void {
    this.valid = false;
  }

  async value(): Promise<[bounds.Bounds, Series[]]> {
    const chan = await fetchChannel(this.client, this.props.channel, this.props.index);
    if (!this.valid) await this.readFixed(chan.key);
    let b = bounds.max(this.data.map((d) => d.bounds));
    if (chan.dataType.equals(DataType.TIMESTAMP)) {
      b = {
        upper: Math.min(b.upper, this.props.timeRange.end.valueOf()),
        lower: Math.max(b.lower, this.props.timeRange.start.valueOf()),
      };
    }
    return [b, this.data];
  }

  private async readFixed(key: channel.Key): Promise<void> {
    const res = await this.client.read(this.props.timeRange, [key]);
    this.data = res[key].data;
    this.valid = true;
  }
}

const dynamicSeriesSourcePropsZ = z.object({
  channel: z.number(),
  index: z.boolean().optional().default(false),
  timeSpan: TimeSpan.z,
});

export type DynamicSeriesSourceProps = z.input<typeof dynamicSeriesSourcePropsZ>;

export class DynamicSeriesSource
  extends telem.AbstractSource<typeof dynamicSeriesSourcePropsZ>
  implements telem.SeriesSource
{
  static readonly TYPE = "dynamic-series-source";
  private readonly client: client.Client;
  private readonly data: Series[] = [];
  private stopStreaming?: Destructor;
  private valid: boolean = false;

  constructor(client: client.Client, props: unknown) {
    super(props);
    this.client = client;
  }

  async value(): Promise<[bounds.Bounds, Series[]]> {
    const { channel, index, timeSpan } = this.props;
    const ch = await fetchChannel(this.client, channel, index);
    if (!this.valid) await this.read(ch.key);
    let b = bounds.max(this.data.map((d) => d.bounds));
    if (ch.dataType.equals(DataType.TIMESTAMP)) {
      b = {
        upper: b.upper,
        lower: Math.max(b.lower, b.upper - timeSpan.valueOf()),
      };
    }
    return [b, this.data];
  }

  invalidate(): void {
    this.valid = false;
    this.notify();
  }

  private async read(key: channel.Key): Promise<void> {
    const tr = TimeStamp.now().spanRange(-this.props.timeSpan);
    const res = await this.client.read(tr, [key]);
    this.data.push(...res[key].data);
    await this.updateStreamHandler(key);
    this.valid = true;
  }

  private async updateStreamHandler(key: channel.Key): Promise<void> {
    this.stopStreaming?.();
    const handler: client.StreamHandler = (data) => {
      const d = data[key];
      this.data.push(...d.data);
      this.notify();
    };
    this.stopStreaming = await this.client.stream(handler, [key]);
  }

  cleanup(): void {
    this.stopStreaming?.();
    this.stopStreaming = undefined;
  }
}
