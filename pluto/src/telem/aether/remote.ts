// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, NotFoundError } from "@synnaxlabs/client";
import {
  bounds,
  DataType,
  type Destructor,
  MultiSeries,
  primitive,
  type Series,
  status as xstatus,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { z } from "zod";

import { type status } from "@/status/aether";
import { type CreateOptions } from "@/telem/aether/factory";
import {
  AbstractSource,
  type NumberSource,
  type NumberSourceSpec,
  type SeriesSource,
  type SeriesSourceSpec,
  type Spec,
  type Telem,
} from "@/telem/aether/telem";
import { type client } from "@/telem/client";

export const streamChannelValuePropsZ = z.object({
  channel: z.number().or(z.string()),
});

export type StreamChannelValueProps = z.infer<typeof streamChannelValuePropsZ>;

// StreamChannelValue is an implementation of NumberSource that reads and returns the
// most recent value of a channel in real-time.
export class StreamChannelValue
  extends AbstractSource<typeof streamChannelValuePropsZ>
  implements NumberSource
{
  static readonly TYPE = "stream-channel-value";
  schema = streamChannelValuePropsZ;

  private readonly client: client.Client;
  private removeStreamHandler: Destructor | null = null;
  private leadingBuffer: Series | null = null;
  private valid = false;
  private readonly onStatusChange?: status.Adder;
  constructor(client: client.Client, props: unknown, options?: CreateOptions) {
    super(props);
    this.client = client;
    this.onStatusChange = options?.onStatusChange;
  }

  /** @returns the leading series buffer for testing purposes. */
  get testingOnlyLeadingBuffer(): Series | null {
    return this.leadingBuffer;
  }

  /** @returns the internal valid flag for testing purposes */
  get testingOnlyValid(): boolean {
    return this.valid;
  }

  cleanup(): void {
    // Start off by stopping telemetry streaming.
    this.removeStreamHandler?.();
    // Set valid to false so if we read again, we know to update the buffer.
    this.valid = false;
    // Release the leading buffer.
    this.leadingBuffer?.release();
    // Clear out references.
    this.leadingBuffer = null;
    this.removeStreamHandler = null;
  }

  value(): number {
    // No valid channel has been set.
    if (primitive.isZero(this.props.channel)) return NaN;
    if (!this.valid) void this.read();
    // No data has been received and no recent samples were fetched on initialization.
    if (this.leadingBuffer == null || this.leadingBuffer.length === 0) return NaN;
    return this.leadingBuffer.at(-1, true) as number;
  }

  private async read(): Promise<void> {
    try {
      this.valid = true;
      this.removeStreamHandler?.();
      const ch = await this.client.retrieveChannel(this.props.channel);
      const handler: client.StreamHandler = (res) => {
        const data = res.get(ch.key);
        if (data == null) return;
        const first = data.series.at(-1);
        if (first != null) {
          first.acquire();
          this.leadingBuffer?.release();
          this.leadingBuffer = first;
        }
        // Just because we didn't get a new buffer doesn't mean one wasn't allocated.
        this.notify();
      };
      this.removeStreamHandler = await this.client.stream(handler, [ch.key]);
      this.notify();
    } catch (e) {
      this.valid = false;
      this.onStatusChange?.(xstatus.fromException(e, "failed to stream channel value"));
    }
  }
}

interface SelectedChannelProperties
  extends Pick<channel.Payload, "key" | "dataType" | "virtual"> {
  isCalculated: boolean;
}

const fetchChannelProperties = async (
  client: client.ChannelClient,
  ch: channel.KeyOrName,
  fetchIndex: boolean,
): Promise<SelectedChannelProperties> => {
  const c = await client.retrieveChannel(ch);
  const isCalculated = channel.isCalculated(c);
  if (!fetchIndex || c.isIndex)
    return { key: c.key, dataType: c.dataType, virtual: c.virtual, isCalculated };
  if (c.virtual && !isCalculated)
    throw new NotFoundError("cannot use virtual channels as a data source");
  return { key: c.index, dataType: DataType.TIMESTAMP, virtual: false, isCalculated };
};

const channelDataSourcePropsZ = z.object({
  timeRange: TimeRange.z,
  channel: z.number().or(z.string()),
  useIndexOfChannel: z.boolean().optional().default(false),
});

export type ChannelDataProps = z.input<typeof channelDataSourcePropsZ>;

// ChannelData reads a fixed time range of data from a particular channel or its index.
export class ChannelData
  extends AbstractSource<typeof channelDataSourcePropsZ>
  implements SeriesSource
{
  static readonly TYPE = "series-source";
  private readonly client: client.ReadClient & client.ChannelClient;
  schema = channelDataSourcePropsZ;

  private data: MultiSeries = new MultiSeries();
  private valid: boolean = false;
  private channel: SelectedChannelProperties | null = null;
  private readonly onStatusChange?: status.Adder;

  constructor(
    client: client.ReadClient & client.ChannelClient,
    props: unknown,
    options?: CreateOptions,
  ) {
    super(props);
    this.client = client;
    this.onStatusChange = options?.onStatusChange;
  }

  cleanup(): void {
    this.data.release();
    this.valid = false;
    this.channel = null;
  }

  value(): [bounds.Bounds, MultiSeries] {
    const { channel, timeRange } = this.props;
    // If either of these conditions is true, leave the telem invalid
    // and return an empty array.
    if (timeRange.span.isZero || channel === 0) return [bounds.ZERO, this.data];
    if (!this.valid) void this.read();
    const { channel: ch, data } = this;
    if (ch == null) return [bounds.ZERO, this.data];
    let b = data.bounds;
    if (ch.dataType.equals(DataType.TIMESTAMP))
      b = bounds.min([b, timeRange.numericBounds]);
    return [b, data];
  }

  private async read(): Promise<void> {
    try {
      this.valid = true;
      const { timeRange, channel, useIndexOfChannel } = this.props;
      this.channel = await fetchChannelProperties(
        this.client,
        channel,
        useIndexOfChannel,
      );
      const series = await this.client.read(timeRange, this.channel.key);
      series.acquire();
      this.data = series;
      this.notify();
    } catch (e) {
      this.valid = false;
      this.onStatusChange?.(xstatus.fromException(e, "failed to read channel data"));
    }
  }
}

const streamChannelDataPropsZ = z.object({
  channel: z.number().or(z.string()),
  useIndexOfChannel: z.boolean().optional().default(false),
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
  private readonly data: MultiSeries = new MultiSeries([]);
  private readonly now: () => TimeStamp;
  private readonly onStatusChange?: status.Adder;

  private channel: SelectedChannelProperties | null = null;
  private stopStreaming?: Destructor;
  private valid: boolean = false;
  schema = streamChannelDataPropsZ;

  constructor(
    client: client.Client,
    props: unknown,
    options?: CreateOptions,
    now: () => TimeStamp = () => TimeStamp.now(),
  ) {
    super(props);
    this.client = client;
    this.now = now;
    this.onStatusChange = options?.onStatusChange;
  }

  value(): [bounds.Bounds, MultiSeries] {
    const { channel, timeSpan } = this.props;
    if (channel === 0) return [bounds.ZERO, this.data];
    if (!this.valid) void this.read();
    const { data, channel: ch } = this;
    const now = this.now();
    if (ch != null && ch.dataType.isVariable) return [bounds.ZERO, this.data];
    const filtered = data.series
      .filter((d) => d.timeRange.end.after(now.sub(timeSpan)))
      .map((d) => d.bounds);
    const b = bounds.max(filtered);
    if (ch != null && ch.dataType.equals(DataType.TIMESTAMP))
      b.lower = Math.max(b.lower, b.upper - Number(timeSpan.valueOf()));
    return [b, this.data];
  }

  private async read(): Promise<void> {
    try {
      this.valid = true;
      const { channel, useIndexOfChannel, timeSpan } = this.props;
      this.channel = await fetchChannelProperties(
        this.client,
        channel,
        useIndexOfChannel,
      );
      const tr = this.now().spanRange(-timeSpan);
      if (!this.channel.virtual || this.channel.isCalculated) {
        const res = await this.client.read(tr, this.channel.key);
        res.acquire();
        this.data.push(res);
      }
      this.stopStreaming?.();
      const handler: client.StreamHandler = (res) => {
        if (this.channel == null) return;
        const series = res.get(this.channel.key);
        if (series == null) return;
        series.acquire();
        this.data.push(series);
        this.notify();
        this.gcOutOfRangeData();
      };
      this.stopStreaming = await this.client.stream(handler, [this.channel.key]);
      this.notify();
    } catch (e) {
      this.valid = false;
      this.onStatusChange?.(xstatus.fromException(e, "failed to stream channel data"));
    }
  }

  private gcOutOfRangeData(): void {
    const threshold = this.now().sub(this.props.keepFor ?? this.props.timeSpan);
    const toGC = this.data.series.findIndex((d) => d.timeRange.end.before(threshold));
    if (toGC === -1) return;
    this.data.series.splice(toGC, 1).forEach((d) => d.release());
    this.gcOutOfRangeData();
  }

  cleanup(): void {
    this.stopStreaming?.();
    this.stopStreaming = undefined;
    this.data.release();
    this.valid = false;
  }
}

type Constructor = new (
  client: client.Client,
  props: unknown,
  options?: CreateOptions,
) => Telem;

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

  create(spec: Spec, options?: CreateOptions): Telem | null {
    const V = REGISTRY[spec.type];
    if (V == null) return null;
    return new V(this.client, spec.props, options);
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
): NumberSourceSpec => ({
  type: StreamChannelValue.TYPE,
  props,
  variant: "source",
  valueType: "number",
});
