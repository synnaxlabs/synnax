// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  channel,
  type framer,
  NotFoundError,
  status as cstatus,
} from "@synnaxlabs/client";
import {
  bounds,
  DataType,
  type destructor,
  errors,
  MultiSeries,
  primitive,
  type Series,
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
  type StringSource,
  type StringSourceSpec,
  type Telem,
} from "@/telem/aether/telem";

/** The slice of a Synnax client that remote telemetry sources consume. */
export interface Client {
  feed: Pick<framer.Feed, "read" | "stream">;
  channels: {
    retrieve: (ch: channel.Key | channel.Name) => Promise<channel.Channel>;
  };
}

/** Reported by remote sources created while the cluster is disconnected. */
export const DISCONNECTED_STATUS: cstatus.Crude = {
  variant: "warning",
  message: "Core disconnected",
};

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

  private readonly client: Client | null;
  private removeStreamHandler: destructor.Destructor | null = null;
  private leadingBuffer: Series | null = null;
  private generation = 0;
  private valid = false;
  private readonly onStatusChange?: status.Adder;
  constructor(client: Client | null, props: unknown, options?: CreateOptions) {
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
    this.generation++;
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
    const generation = this.generation;
    this.valid = true;
    const { client } = this;
    if (client == null) {
      this.onStatusChange?.(DISCONNECTED_STATUS);
      return;
    }
    try {
      this.removeStreamHandler?.();
      const ch = await client.channels.retrieve(this.props.channel);
      const handler: framer.StreamHandler = (res) => {
        if (generation !== this.generation) return;
        const data = res.get(ch.key);
        if (data == null) return;
        const first = data.series.at(-1);
        if (first != null) {
          first.acquire();
          this.leadingBuffer?.release();
          this.leadingBuffer = first;
        }
        // Just because we didn't get a new buffer doesn't mean one wasn't allocated: an
        // empty update means the leading buffer was appended to in place. A frame that
        // holds nothing for this channel looks the same, so with no buffer yet there is
        // nothing to report.
        else if (this.leadingBuffer == null) return;
        this.notify();
      };
      if (generation !== this.generation) return;
      this.removeStreamHandler = client.feed.stream(handler, [ch.key]).close;
      // Opening the stream is not a sample. Notify only when a buffer already holds
      // one, so a consumer that counts arrivals does not count the open.
      if (this.leadingBuffer != null && this.leadingBuffer.length > 0) this.notify();
    } catch (e) {
      this.valid = false;
      this.onStatusChange?.(cstatus.fromException(e, "Failed to stream channel value"));
    }
  }
}

interface SelectedChannelProperties extends Pick<
  channel.Payload,
  "key" | "dataType" | "virtual"
> {
  isCalculated: boolean;
}

const fetchChannelProperties = async (
  client: Client,
  ch: channel.Key | channel.Name,
  fetchIndex: boolean,
): Promise<SelectedChannelProperties> => {
  const c = await client.channels.retrieve(ch);
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
  useIndexOfChannel: z.boolean().default(false),
});

export type ChannelDataProps = z.input<typeof channelDataSourcePropsZ>;

// ChannelData reads a fixed time range of data from a particular channel or its index.
export class ChannelData
  extends AbstractSource<typeof channelDataSourcePropsZ>
  implements SeriesSource
{
  static readonly TYPE = "series-source";
  private readonly client: Client | null;
  schema = channelDataSourcePropsZ;

  private data: MultiSeries = new MultiSeries();
  private valid: boolean = false;
  private generation = 0;
  private channel: SelectedChannelProperties | null = null;
  private readonly onStatusChange?: status.Adder;

  constructor(client: Client | null, props: unknown, options?: CreateOptions) {
    super(props);
    this.client = client;
    this.onStatusChange = options?.onStatusChange;
  }

  cleanup(): void {
    this.generation++;
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
    const generation = this.generation;
    this.valid = true;
    const { client } = this;
    if (client == null) {
      this.onStatusChange?.(DISCONNECTED_STATUS);
      return;
    }
    try {
      const { timeRange, channel, useIndexOfChannel } = this.props;
      const ch = await fetchChannelProperties(client, channel, useIndexOfChannel);
      if (generation !== this.generation) return;
      this.channel = ch;
      const series = await client.feed.read(timeRange, ch.key);
      if (generation !== this.generation) return;
      series.acquire();
      this.data = series;
      this.notify();
    } catch (e) {
      this.valid = false;
      this.onStatusChange?.(cstatus.fromException(e, "Failed to read channel data"));
    }
  }
}

const streamChannelDataPropsZ = z.object({
  channel: z.number().or(z.string()),
  useIndexOfChannel: z.boolean().default(false),
  timeSpan: TimeSpan.z,
  keepFor: TimeSpan.z.optional(),
});

export type StreamChannelDataProps = z.input<typeof streamChannelDataPropsZ>;

export class StreamChannelData
  extends AbstractSource<typeof streamChannelDataPropsZ>
  implements SeriesSource
{
  static readonly TYPE = "dynamic-series-source";
  private readonly client: Client | null;
  private readonly data: MultiSeries = new MultiSeries([]);
  private readonly now: () => TimeStamp;
  private readonly onStatusChange?: status.Adder;

  private channel: SelectedChannelProperties | null = null;
  private stopStreaming?: destructor.Destructor;
  private valid: boolean = false;
  private generation = 0;
  schema = streamChannelDataPropsZ;

  constructor(
    client: Client | null,
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
    const generation = this.generation;
    this.valid = true;
    const { client } = this;
    if (client == null) {
      this.onStatusChange?.(DISCONNECTED_STATUS);
      return;
    }
    try {
      const { channel, useIndexOfChannel, timeSpan } = this.props;
      const fetched = await fetchChannelProperties(client, channel, useIndexOfChannel);
      if (generation !== this.generation) return;
      this.channel = fetched;
      const tr = this.now().spanRange(-timeSpan);
      if (!this.channel.virtual || this.channel.isCalculated)
        try {
          const res = await client.feed.read(tr, this.channel.key);
          if (generation !== this.generation) return;
          this.pushNew(res.series);
        } catch (e) {
          // Certain calculated channels can fail to read because they need access to
          // virtual channels that cannot be read from historically.
          if (
            e instanceof Error &&
            (e.message.includes("cannot open iterator on virtual channel") ||
              e.message.includes("cannot read from free channel"))
          )
            console.warn("failed to read calculated channel data", e);
          else throw errors.fromUnknown(e);
        }

      this.stopStreaming?.();
      const handler: framer.StreamHandler = (res) => {
        if (generation !== this.generation || this.channel == null) return;
        const series = res.get(this.channel.key);
        if (series == null) return;
        this.pushNew(series.series);
        this.notify();
        this.gcOutOfRangeData();
      };
      if (generation !== this.generation) return;
      this.stopStreaming = client.feed.stream(handler, [this.channel.key]).close;
      this.notify();
    } catch (e) {
      this.valid = false;
      this.onStatusChange?.(cstatus.fromException(e, "Failed to stream channel data"));
    }
  }

  // feed.read returns the live leading buffer that the stream's first delivery
  // repeats, so series already held by identity are skipped.
  private pushNew(series: Series[]): void {
    for (const s of series) {
      if (this.data.series.includes(s)) continue;
      s.acquire();
      this.data.push(s);
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
    this.generation++;
    this.stopStreaming?.();
    this.stopStreaming = undefined;
    this.data.release();
    this.valid = false;
  }
}

// StreamChannelStringValue reads the most recent value of a channel in real-time as
// text, preserving variable density data types rather than coercing to a number.
export class StreamChannelStringValue
  extends AbstractSource<typeof streamChannelValuePropsZ>
  implements StringSource
{
  static readonly TYPE = "stream-channel-string-value";
  schema = streamChannelValuePropsZ;

  private readonly client: Client | null;
  private removeStreamHandler: destructor.Destructor | null = null;
  private leadingBuffer: Series | null = null;
  private latest = "";
  // Buffer length the latest decode was taken at, or -1 to force a re-decode.
  private decodedAt = -1;
  // Bumped by cleanup to invalidate a read that is still awaiting.
  private generation = 0;
  private valid = false;
  private readonly onStatusChange?: status.Adder;
  constructor(client: Client | null, props: unknown, options?: CreateOptions) {
    super(props);
    this.client = client;
    this.onStatusChange = options?.onStatusChange;
  }

  cleanup(): void {
    this.generation++;
    this.removeStreamHandler?.();
    this.valid = false;
    this.leadingBuffer?.release();
    this.leadingBuffer = null;
    this.latest = "";
    this.decodedAt = -1;
    this.removeStreamHandler = null;
  }

  value(): string {
    // No valid channel has been set.
    if (primitive.isZero(this.props.channel)) return "";
    if (!this.valid) void this.read();
    const buffer = this.leadingBuffer;
    if (buffer == null || buffer.length === 0) return "";
    // Samples are appended into the leading buffer in place, so the length is what
    // marks new data. Decoding is gated on it because asString scans linearly.
    if (buffer.length !== this.decodedAt) {
      this.latest = buffer.asString(-1) ?? this.latest;
      this.decodedAt = buffer.length;
    }
    return this.latest;
  }

  private async read(): Promise<void> {
    const generation = this.generation;
    this.valid = true;
    const { client } = this;
    if (client == null) {
      this.onStatusChange?.(DISCONNECTED_STATUS);
      return;
    }
    try {
      this.removeStreamHandler?.();
      const ch = await client.channels.retrieve(this.props.channel);
      const handler: framer.StreamHandler = (res) => {
        if (generation !== this.generation) return;
        const data = res.get(ch.key);
        if (data == null) return;
        const leading = data.series.at(-1);
        if (leading != null) {
          leading.acquire();
          this.leadingBuffer?.release();
          this.leadingBuffer = leading;
          this.decodedAt = -1;
        }
        // An empty update means the leading buffer was appended to in place. A frame
        // that holds nothing for this channel looks the same, so with no buffer yet
        // there is nothing to report.
        else if (this.leadingBuffer == null) return;
        this.notify();
      };
      if (generation !== this.generation) return;
      this.removeStreamHandler = client.feed.stream(handler, [ch.key]).close;
      // Opening the stream is not a sample. Notify only when a buffer already holds
      // one, so a consumer that counts arrivals does not count the open.
      if (this.leadingBuffer != null && this.leadingBuffer.length > 0) this.notify();
    } catch (e) {
      this.valid = false;
      this.onStatusChange?.(
        cstatus.fromException(e, "Failed to stream channel string value"),
      );
    }
  }
}

type Constructor = new (
  client: Client | null,
  props: unknown,
  options?: CreateOptions,
) => Telem;

const REGISTRY: Record<string, Constructor> = {
  [ChannelData.TYPE]: ChannelData,
  [StreamChannelData.TYPE]: StreamChannelData,
  [StreamChannelValue.TYPE]: StreamChannelValue,
  [StreamChannelStringValue.TYPE]: StreamChannelStringValue,
};

export class RemoteFactory {
  type = "remote";
  private readonly client: Client | null;
  constructor(client: Client | null) {
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

export const streamChannelStringValue = (
  props: StreamChannelValueProps,
): StringSourceSpec => ({
  type: StreamChannelStringValue.TYPE,
  props,
  variant: "source",
  valueType: "string",
});
