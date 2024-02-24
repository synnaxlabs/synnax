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
  type Series,
  TimeSpan,
  TimeStamp,
  DataType,
  addSamples,
  type AsyncDestructor,
} from "@synnaxlabs/x";
import { z } from "zod";

import {
  AbstractSource,
  type Spec,
  type NumberSource,
  type Telem,
  type SeriesSourceSpec,
  type NumberSourceSpec,
  type SeriesSource,
} from "@/telem/aether/telem";
import { type client } from "@/telem/client";

export const streamChannelValuePropsZ = z.object({
  channel: z.number(),
});

export type StreamChannelValueProps = z.infer<typeof streamChannelValuePropsZ>;

export class StreamChannelValue
  extends AbstractSource<typeof streamChannelValuePropsZ>
  implements NumberSource
{
  removeStreamHandler: AsyncDestructor | null = null;

  static readonly TYPE = "range-point";

  schema = streamChannelValuePropsZ;

  private valid = false;
  private leadingBuffer: Series | null = null;
  private readonly client: client.Client;

  constructor(client: client.Client, props: unknown) {
    super(props);
    this.client = client;
  }

  async cleanup(): Promise<void> {
    await this.removeStreamHandler?.();
    this.valid = false;
    this.leadingBuffer?.release();
    this.leadingBuffer = null;
    this.removeStreamHandler = null;
  }

  async value(): Promise<number> {
    if (this.props.channel === 0) return 0;
    if (!this.valid) await this.read();
    if (this.leadingBuffer == null || this.leadingBuffer.length === 0) return 0;
    const v = this.leadingBuffer.data[this.leadingBuffer.length - 1];
    return Number(addSamples(v, this.leadingBuffer.sampleOffset));
  }

  async read(): Promise<void> {
    try {
      const { channel } = this.props;
      const now = TimeStamp.now()
        .sub(TimeStamp.seconds(10))
        .spanRange(TimeSpan.seconds(20));
      const res = await this.client.read(now, [channel]);
      const newData = res[channel].data;
      if (newData.length === 0) return;
      const first = newData[newData.length - 1];
      first.acquire();
      this.leadingBuffer = first;
      await this.updateStreamHandler();
      this.valid = true;
    } catch (e) {
      console.error(e);
    }
  }

  private async updateStreamHandler(): Promise<void> {
    await this.removeStreamHandler?.();
    const { channel } = this.props;
    const handler: client.StreamHandler = (data) => {
      const res = data[channel];
      const newData = res.data;
      if (newData.length !== 0) {
        const first = newData[newData.length - 1];
        first.acquire();
        this.leadingBuffer?.release();
        this.leadingBuffer = first;
      }
      // Just because we didn't
      this.notify?.();
    };
    this.removeStreamHandler = await this.client.stream(handler, [channel]);
  }
}

const fetchChannel = async (
  client: client.ChannelClient,
  channel: channel.Key,
  index: boolean,
): Promise<channel.Channel> => {
  if (!index) return await client.retrieveChannel(channel);
  const c = await client.retrieveChannel(channel);
  return await client.retrieveChannel(c.index);
};

const channelDataSourcePropsZ = z.object({
  timeRange: TimeRange.z,
  channel: z.number(),
  index: z.boolean().optional().default(false),
});

export type ChannelDataProps = z.input<typeof channelDataSourcePropsZ>;

export class ChannelData
  extends AbstractSource<typeof channelDataSourcePropsZ>
  implements ChannelData
{
  static readonly TYPE = "series-source";
  private readonly client: client.ReadClient & client.ChannelClient;
  private data: Series[] = [];
  schema = channelDataSourcePropsZ;
  valid: boolean = false;

  constructor(client: client.ReadClient & client.ChannelClient, props: unknown) {
    super(props);
    this.client = client;
  }

  async cleanup(): Promise<void> {
    this.data.forEach((d) => d.release());
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
    const newData = res[key].data;
    newData.forEach((d) => d.acquire());
    this.data = newData;
    this.valid = true;
  }
}

const streamChannelDataPropsZ = z.object({
  channel: z.number(),
  index: z.boolean().optional().default(false),
  timeSpan: TimeSpan.z,
  keepFor: TimeSpan.z.optional(),
});

export type StreamChannelDataProps = z.input<typeof streamChannelDataPropsZ>;

export class StreamChannelData
  extends AbstractSource<typeof streamChannelDataPropsZ>
  implements SeriesSource
{
  static readonly TYPE = "dynamic-series-source";
  private readonly client: client.Client;
  private readonly data: Series[] = [];
  private stopStreaming?: AsyncDestructor;
  private valid: boolean = false;
  schema = streamChannelDataPropsZ;

  constructor(client: client.Client, props: unknown) {
    super(props);
    this.client = client;
  }

  async value(): Promise<[bounds.Bounds, Series[]]> {
    const { channel, index, timeSpan } = this.props;
    const now = TimeStamp.now();
    const ch = await fetchChannel(this.client, channel, index);
    if (!this.valid) await this.read(ch.key);
    let b = bounds.max(
      this.data
        .filter((d) => d.timeRange.end.after(now.sub(timeSpan)))
        .map((d) => d.bounds),
    );
    if (ch.dataType.equals(DataType.TIMESTAMP)) {
      b = {
        upper: b.upper,
        lower: Math.max(b.lower, b.upper - timeSpan.valueOf()),
      };
    }
    return [b, this.data];
  }

  private async read(key: channel.Key): Promise<void> {
    const tr = TimeStamp.now().spanRange(-this.props.timeSpan);
    const res = await this.client.read(tr, [key]);
    const newData = res[key].data;
    newData.forEach((d) => d.acquire());
    this.data.push(...res[key].data);
    await this.updateStreamHandler(key);
    this.valid = true;
  }

  private async updateStreamHandler(key: channel.Key): Promise<void> {
    if (this.stopStreaming != null) await this.stopStreaming();
    const handler: client.StreamHandler = (res) => {
      const newData = res[key].data;
      newData.forEach((d) => d.acquire());
      this.data.push(...newData);
      this.gcOutOfRangeData();
      this.notify();
    };
    this.stopStreaming = await this.client.stream(handler, [key]);
  }

  private gcOutOfRangeData(): void {
    const threshold = TimeStamp.now().sub(this.props.keepFor ?? this.props.timeSpan);
    const toGC = this.data.findIndex((d) => d.timeRange.end.before(threshold));
    if (toGC === -1) return;
    this.data.splice(toGC, 1).forEach((d) => d.release());
    this.gcOutOfRangeData();
  }

  async cleanup(): Promise<void> {
    await this.stopStreaming?.();
    this.stopStreaming = undefined;
    this.data.forEach((d) => d.release());
  }
}

type Constructor = new (client: client.Client, props: unknown) => Telem;

const REGISTRY: Record<string, Constructor> = {
  [ChannelData.TYPE]: ChannelData,
  [StreamChannelData.TYPE]: StreamChannelData,
  [StreamChannelValue.TYPE]: StreamChannelValue,
};

export class RemoteFactory implements RemoteFactory {
  type = "remote";
  private readonly client: client.Client;
  constructor(client: client.Client) {
    this.client = client;
  }

  create(spec: Spec): Telem | null {
    if (!(spec.type in REGISTRY)) return null;
    const t = new REGISTRY[spec.type](this.client, spec.props);
    return t;
  }
}

export const channelData = (props: ChannelDataProps): SeriesSourceSpec => ({
  type: ChannelData.TYPE,
  props,
  variant: "source",
  valueType: "series",
});

export const streamChannelData = (props: StreamChannelDataProps): SeriesSourceSpec => ({
  type: StreamChannelData.TYPE,
  props,
  variant: "source",
  valueType: "series",
});

export const streamChannelValue = (
  props: Omit<StreamChannelValueProps, "units">,
): NumberSourceSpec => {
  return {
    type: StreamChannelValue.TYPE,
    props,
    variant: "source",
    valueType: "number",
  };
};
