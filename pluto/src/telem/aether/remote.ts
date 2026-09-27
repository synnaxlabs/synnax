// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  AccessDeniedError,
  channel,
  framer,
  NotFoundError,
  status as cstatus,
  ValidationError,
} from "@synnaxlabs/client";
import {
  aggregationZ,
  bounds,
  breaker,
  DataType,
  type destructor,
  errors,
  MultiSeries,
  primitive,
  type Series,
  sync,
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
  type ValueProps,
} from "@/telem/aether/telem";
import {
  choose,
  detailZ,
  indexes,
  level,
  pointLimit,
  type Position,
} from "@/telem/aether/tiles";

/** The slice of a Synnax client that remote telemetry sources consume. */
export interface Client {
  feed: Pick<framer.Feed, "read" | "stream" | "readLatest" | "readTile">;
  channels: {
    retrieve: (ch: channel.Key | channel.Name) => Promise<channel.Channel>;
  };
}

const readLatestSeries = async (
  client: Client,
  ch: channel.Channel,
  onStatusChange?: status.Adder,
): Promise<Series | null> => {
  if (ch.virtual && !channel.isCalculated(ch)) return null;
  try {
    const latest = (await client.feed.readLatest(ch.key)).series.at(-1);
    if (latest == null || latest.length === 0) return null;
    return latest;
  } catch (e) {
    onStatusChange?.(cstatus.fromException(e, "Failed to read latest value"));
    return null;
  }
};

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
  private sampleTime_: TimeStamp | null = null;
  private generation = 0;
  private valid = false;
  private readonly onStatusChange?: status.Adder;
  constructor(client: Client | null, props: unknown, options?: CreateOptions) {
    super(props);
    this.client = client;
    this.onStatusChange = options?.onStatusChange;
  }

  sampleTime(): TimeStamp | null {
    return this.sampleTime_;
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
    this.removeStreamHandler?.();
    // Set valid to false so if we read again, we know to update the buffer.
    this.valid = false;
    this.leadingBuffer?.release();
    this.leadingBuffer = null;
    this.sampleTime_ = null;
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

  /** Never rejects: a failure invalidates the read and reaches onStatusChange. */
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
          this.sampleTime_ = null;
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
      if (this.leadingBuffer != null && this.leadingBuffer.length > 0) {
        this.notify();
        return;
      }
      const latest = await readLatestSeries(client, ch, this.onStatusChange);
      if (latest == null) return;
      if (generation !== this.generation || this.leadingBuffer != null) return;
      latest.acquire();
      this.leadingBuffer = latest;
      this.sampleTime_ = latest.timeRange.isZero ? null : latest.timeRange.end;
      this.notify();
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

// Spans within the live buffer draw immediately and skip the loading state.
const LOADING_MIN_SPAN = TimeSpan.minutes(1).valueOf();

const channelDataSourcePropsZ = z.object({
  timeRange: TimeRange.z,
  channel: z.number().or(z.string()),
  useIndexOfChannel: z.boolean().default(false),
});

export type ChannelDataProps = z.input<typeof channelDataSourcePropsZ>;

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
  private readonly skipLoading: boolean;

  constructor(client: Client | null, props: unknown, options?: CreateOptions) {
    super(props);
    this.client = client;
    this.onStatusChange = options?.onStatusChange;
    const { channel, timeRange } = this.props;
    this.skipLoading = channel === 0 || timeRange.span.isZero;
    this.loading_ = true;
  }

  cleanup(): void {
    this.generation++;
    this.data.release();
    this.valid = false;
    this.loading_ = false;
    this.channel = null;
  }

  loading(): boolean {
    if (this.skipLoading || !this.loading_) return false;
    if (!this.valid) void this.read();
    return this.loading_;
  }

  value(): [bounds.Bounds, MultiSeries] {
    const { channel, timeRange } = this.props;
    // If either of these conditions is true, leave the telem invalid
    // and return an empty array.
    if (timeRange.span.isZero || channel === 0) return [bounds.INVALID, this.data];
    if (!this.valid) void this.read();
    const { channel: ch, data } = this;
    if (ch == null) return [bounds.INVALID, this.data];
    let b = data.bounds;
    if (ch.dataType.equals(DataType.TIMESTAMP)) {
      b = bounds.min([b, timeRange.numericBounds]);
      // A reversed intersection means the data lies outside the requested range.
      if (b.lower > b.upper) b = bounds.INVALID;
    }
    return [b, data];
  }

  /** Never rejects: a failure invalidates the read and reaches onStatusChange. */
  private async read(): Promise<void> {
    const generation = this.generation;
    this.valid = true;
    const { client } = this;
    try {
      if (client == null) {
        this.onStatusChange?.(DISCONNECTED_STATUS);
        return;
      }
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
    } finally {
      this.declareLoaded();
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
  private readonly skipLoading: boolean;
  private generation = 0;
  private readonly breaker: breaker.Breaker;
  private readonly retryNotifier = new sync.Notifier();
  private lastFailure?: string;
  schema = streamChannelDataPropsZ;

  constructor(
    client: Client | null,
    props: unknown,
    options?: CreateOptions,
    now: () => TimeStamp = () => TimeStamp.now(),
    breakerConfig?: breaker.Config,
  ) {
    super(props);
    this.client = client;
    this.now = now;
    this.onStatusChange = options?.onStatusChange;
    const { channel, timeSpan } = this.props;
    this.skipLoading = channel === 0 || timeSpan.valueOf() <= LOADING_MIN_SPAN;
    this.loading_ = true;
    this.breaker = new breaker.Breaker({
      baseInterval: TimeSpan.seconds(1),
      // A tall interval cap: live data flows independently of this loop, and a
      // back-fill loses value as the window fills with live samples.
      maxInterval: TimeSpan.seconds(30),
      maxRetries: Infinity,
      scale: 2,
      jitter: 0.25,
      // cleanup interrupts a pending backoff so a retired source's retry loop ends
      // promptly instead of sleeping through it
      sleepFn: async (duration) => {
        await this.retryNotifier.wait(duration);
      },
      ...breakerConfig,
    });
  }

  loading(): boolean {
    if (this.skipLoading || !this.loading_) return false;
    if (!this.valid) void this.read();
    return this.loading_;
  }

  value(): [bounds.Bounds, MultiSeries] {
    const { channel, timeSpan } = this.props;
    if (channel === 0) return [bounds.INVALID, this.data];
    if (!this.valid) void this.read();
    const { data, channel: ch } = this;
    const now = this.now();
    if (ch != null && ch.dataType.isVariable) return [bounds.INVALID, this.data];
    const filtered = data.series
      .filter((d) => d.timeRange.end.after(now.sub(timeSpan)))
      .map((d) => d.bounds);
    const b = bounds.max(filtered);
    if (ch != null && ch.dataType.equals(DataType.TIMESTAMP))
      b.lower = Math.max(b.lower, b.upper - Number(timeSpan.valueOf()));
    return [b, this.data];
  }

  /**
   * Never rejects. A connectivity failure retries under the breaker; a definitive
   * rejection parks the source. Every distinct failure reaches onStatusChange.
   */
  private async read(): Promise<void> {
    const generation = this.generation;
    this.valid = true;
    const { client } = this;
    if (client == null) {
      this.declareLoaded();
      this.onStatusChange?.(DISCONNECTED_STATUS);
      return;
    }
    while (generation === this.generation)
      try {
        await this.attempt(generation, client);
        this.declareLoaded();
        this.breaker.reset();
        this.lastFailure = undefined;
        return;
      } catch (e) {
        // Declare loaded on the first failure so retries never hold loading forever.
        this.declareLoaded();
        // Retrying only fixes connectivity; a definitive rejection recurs on every
        // attempt.
        if (
          AccessDeniedError.matches(e) ||
          ValidationError.matches(e) ||
          NotFoundError.matches(e)
        )
          return;
        if (!(await this.breaker.wait())) return;
      }
  }

  // One full read attempt: it opens the live stream, then runs the historical
  // back-fill, so a stalled or failed back-fill never blocks live data. Throws the
  // failure after posting its status.
  private async attempt(generation: number, client: Client): Promise<void> {
    const { channel, useIndexOfChannel, timeSpan } = this.props;
    let fetched: SelectedChannelProperties;
    try {
      fetched = await fetchChannelProperties(client, channel, useIndexOfChannel);
      if (generation !== this.generation) return;
      this.channel = fetched;
      const handler: framer.StreamHandler = (res) => {
        if (generation !== this.generation || this.channel == null) return;
        const series = res.get(this.channel.key);
        if (series == null) return;
        this.pushNew(series.series);
        this.notify();
        this.gcOutOfRangeData();
      };
      this.stopStreaming?.();
      this.stopStreaming = client.feed.stream(handler, [fetched.key]).close;
    } catch (e) {
      this.reportFailure(e, "Failed to stream channel data");
      throw errors.fromUnknown(e);
    }
    if (!fetched.virtual || fetched.isCalculated)
      try {
        const res = await client.feed.read(
          this.now().spanRange(-timeSpan),
          fetched.key,
        );
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
        else {
          this.reportFailure(e, "Failed to read channel data");
          throw errors.fromUnknown(e);
        }
      }
    this.notify();
  }

  // Retries repeat at the breaker's pace, so an incident posts one status when it
  // starts and again only when the failure changes. Success clears the memory.
  private reportFailure(e: unknown, message: string): void {
    const failure = cstatus.fromException(e, message);
    const key = `${failure.message}: ${failure.description}`;
    if (this.lastFailure === key) return;
    this.lastFailure = key;
    this.onStatusChange?.(failure);
  }

  // feed.read returns the live leading buffer that the stream's first delivery repeats,
  // so series already held by identity are skipped. A late back-fill is inserted by
  // alignment: consumers assume the array is chronological.
  private pushNew(series: Series[]): void {
    for (const s of series) {
      if (this.data.series.includes(s)) continue;
      s.acquire();
      const at = this.data.series.findIndex((held) => held.alignment > s.alignment);
      if (at === -1) this.data.push(s);
      else this.data.series.splice(at, 0, s);
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
    this.retryNotifier.notify();
    this.stopStreaming?.();
    this.stopStreaming = undefined;
    this.data.release();
    this.valid = false;
    // No notify so a read settling after cleanup cannot wake stale observers.
    this.loading_ = false;
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
  private sampleTime_: TimeStamp | null = null;
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
    this.sampleTime_ = null;
    this.latest = "";
    this.decodedAt = -1;
    this.removeStreamHandler = null;
  }

  sampleTime(): TimeStamp | null {
    return this.sampleTime_;
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

  /** Never rejects: a failure invalidates the read and reaches onStatusChange. */
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
          this.sampleTime_ = null;
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
      if (this.leadingBuffer != null && this.leadingBuffer.length > 0) {
        this.notify();
        return;
      }
      const latest = await readLatestSeries(client, ch, this.onStatusChange);
      if (latest == null) return;
      if (generation !== this.generation || this.leadingBuffer != null) return;
      latest.acquire();
      this.leadingBuffer = latest;
      this.sampleTime_ = latest.timeRange.isZero ? null : latest.timeRange.end;
      this.decodedAt = -1;
      this.notify();
    } catch (e) {
      this.valid = false;
      this.onStatusChange?.(
        cstatus.fromException(e, "Failed to stream channel string value"),
      );
    }
  }
}

const tiledChannelDataPropsZ = z.object({
  channel: z.number().or(z.string()),
  useIndexOfChannel: z.boolean().default(false),
  /** The line's x channel, or zero when x is the index of the line's y channel. */
  xChannel: z.number().or(z.string()).default(0),
  /** The home view of a static line. */
  timeRange: TimeRange.z.optional(),
  /** The span of the home view of a live line, which ends now. */
  timeSpan: TimeSpan.z.optional(),
  aggregation: aggregationZ.default("min_max"),
  detail: detailZ.default("medium"),
});

export type TiledChannelDataProps = z.input<typeof tiledChannelDataPropsZ>;

interface HeldTile {
  spec: framer.TileSpec;
  series: MultiSeries;
}

// The width tiles are sized for before a line reports its own.
const DEFAULT_WIDTH = 1000;

const tileKey = (spec: framer.TileSpec): string => {
  const { level, index, pointLimit, aggregation, start, end } = spec;
  return [level, index, pointLimit, aggregation, start?.valueOf(), end?.valueOf()].join(
    "/",
  );
};

const viewRange = ({ lower, upper }: bounds.Bounds): TimeRange =>
  new TimeRange(BigInt(Math.max(Math.floor(lower), 0)), BigInt(Math.ceil(upper)));

/**
 * A series source for a line whose x channel is an index. It draws tiles of reduced
 * data that cover the view the line passes to value(), and its home view when the line
 * passes none. A live line streams the recent part of its home view at full
 * resolution. A line whose x channel is not an index reads its home view once, reduced
 * with decimate.
 */
export class TiledChannelData
  extends AbstractSource<typeof tiledChannelDataPropsZ>
  implements SeriesSource
{
  static readonly TYPE = "tiled-series-source";
  schema = tiledChannelDataPropsZ;

  private readonly client: Client | null;
  private readonly now: () => TimeStamp;
  private readonly onStatusChange?: status.Adder;
  private generation = 0;
  private valid = false;
  private channel: SelectedChannelProperties | null = null;
  // xy is true when the line's x channel is not an index.
  private xy = false;
  // held maps tile keys to the tiles this source has acquired.
  private readonly held = new Map<string, HeldTile>();
  // pending holds the keys of tiles being fetched.
  private readonly pending = new Set<string>();
  private width = DEFAULT_WIDTH;
  private homeLoaded = false;
  // raw holds the acquired full-resolution series of a live line, in time order.
  private readonly raw = new MultiSeries([]);
  private rawLoaded = false;
  private stopStreaming?: destructor.Destructor;
  private lastFailure?: string;

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

  private get live(): boolean {
    return this.props.timeSpan != null;
  }

  loading(): boolean {
    if (primitive.isZero(this.props.channel) || this.homeLoaded) return false;
    this.value();
    return !this.homeLoaded;
  }

  fetching(): boolean {
    return this.homeLoaded && this.pending.size > 0;
  }

  value(props?: ValueProps): [bounds.Bounds, MultiSeries] {
    if (primitive.isZero(this.props.channel))
      return [bounds.INVALID, new MultiSeries()];
    if (!this.valid) void this.resolve();
    const ch = this.channel;
    if (ch == null) return [bounds.INVALID, new MultiSeries()];
    const home = this.home();
    const view = props?.view;
    if (view != null) this.width = view.width;
    const drawsView =
      view != null &&
      !this.xy &&
      bounds.isFinite(view.bounds) &&
      bounds.span(view.bounds) > 0;
    const series = this.select(drawsView ? viewRange(view.bounds) : home, drawsView);
    if (!drawsView) this.homeLoaded ||= this.loaded(home);
    return [this.bounds(ch, home, series), series];
  }

  private home(): TimeRange {
    const { timeRange, timeSpan } = this.props;
    if (timeSpan == null) return timeRange ?? TimeRange.ZERO;
    const now = this.now();
    return new TimeRange(now.sub(timeSpan), now);
  }

  // rawFrom is where a live line's full-resolution data starts. It sits on a tile
  // boundary of the home level, so no tile of the home level or finer crosses it.
  private rawFrom(): TimeStamp {
    const home = this.home();
    if (this.xy) return home.start;
    const span = framer.tileSpan(level(home)).valueOf();
    return new TimeStamp((home.start.valueOf() / span) * span);
  }

  private targets(view: TimeRange): framer.TileSpec[] {
    const { key } = this.channel!;
    const aggregation = this.xy ? "decimate" : this.props.aggregation;
    // Tiles are cut to [start, end): the home view for an xy line, and the start of
    // full-resolution data for a live line.
    let start: TimeStamp | undefined;
    let end: TimeStamp | undefined;
    let tiled = view;
    if (this.xy) [start, end] = [this.home().start, this.home().end];
    else if (this.live) {
      end = this.rawFrom();
      if (!end.after(view.start)) return [];
      if (view.end.after(end)) tiled = new TimeRange(view.start, end);
    }
    const l = level(view);
    const limit = pointLimit({
      level: l,
      view,
      width: this.width,
      detail: this.props.detail,
      aggregation,
    });
    return indexes(tiled, l).map((index) => {
      const spec: framer.TileSpec = {
        key,
        level: l,
        index,
        pointLimit: limit,
        aggregation,
      };
      const tr = framer.tileRange(spec);
      if (start != null && tr.start.before(start)) spec.start = start;
      if (end != null && tr.end.after(end)) spec.end = end;
      return spec;
    });
  }

  private find(p: Position, preferred: Map<string, string>): HeldTile | undefined {
    const want = preferred.get(`${p.level}/${p.index}`);
    if (want != null) {
      const exact = this.held.get(want);
      if (exact != null) return exact;
    }
    for (const tile of this.held.values())
      if (tile.spec.level === p.level && tile.spec.index === p.index) return tile;
    return undefined;
  }

  private select(view: TimeRange, prune: boolean): MultiSeries {
    const targets = this.targets(view);
    targets.forEach((spec) => this.request(spec));
    const preferred = new Map(
      targets.map((t) => [`${t.level}/${t.index}`, tileKey(t)]),
    );
    const out: Series[] = [];
    if (targets.length > 0) {
      const l = targets[0].level;
      const chosen = choose(
        l,
        targets.map((t) => t.index),
        (p) => this.find(p, preferred) != null,
      ).map((p) => this.find(p, preferred)!);
      chosen.forEach((tile) => out.push(...tile.series.series));
      if (prune) {
        // The home view's tiles stay held, and so do the tiles that stand in for them
        // until they arrive.
        const home = this.targets(this.home());
        const homePreferred = new Map(
          home.map((t) => [`${t.level}/${t.index}`, tileKey(t)]),
        );
        const keep = new Set([
          ...chosen.map((t) => tileKey(t.spec)),
          ...preferred.values(),
          ...homePreferred.values(),
          ...home.flatMap((t) => {
            const found = this.find(t, homePreferred);
            return found == null ? [] : [tileKey(found.spec)];
          }),
        ]);
        for (const [key, tile] of this.held)
          if (!keep.has(key)) {
            tile.series.release();
            this.held.delete(key);
          }
      }
    }
    if (this.live) {
      this.gcRaw();
      out.push(...this.raw.series);
    }
    return new MultiSeries(out);
  }

  private loaded(home: TimeRange): boolean {
    if (this.live && !this.rawLoaded) return false;
    return this.targets(home).every((spec) => this.held.has(tileKey(spec)));
  }

  private bounds(
    ch: SelectedChannelProperties,
    home: TimeRange,
    series: MultiSeries,
  ): bounds.Bounds {
    if (!ch.dataType.equals(DataType.TIMESTAMP) || this.xy) return series.bounds;
    if (!this.live) return home.numericBounds;
    const latest = this.raw.series.length > 0 ? this.raw.bounds.upper : NaN;
    const upper = Number.isFinite(latest) ? latest : Number(home.end.valueOf());
    return { lower: upper - Number(this.props.timeSpan!.valueOf()), upper };
  }

  /** Never rejects: a failure reaches onStatusChange, and a later value() retries. */
  private request(spec: framer.TileSpec): void {
    const key = tileKey(spec);
    const { client } = this;
    if (client == null || this.held.has(key) || this.pending.has(key)) return;
    const generation = this.generation;
    this.pending.add(key);
    client.feed
      .readTile(spec)
      .then((series) => {
        if (generation !== this.generation) return;
        this.pending.delete(key);
        series.acquire();
        this.held.set(key, { spec, series });
        this.lastFailure = undefined;
        this.notify();
      })
      .catch((e: unknown) => {
        if (generation !== this.generation) return;
        this.pending.delete(key);
        this.reportFailure(e, "Failed to read channel data");
      });
  }

  /** Never rejects: a failure invalidates the source and reaches onStatusChange. */
  private async resolve(): Promise<void> {
    const generation = this.generation;
    this.valid = true;
    const { client } = this;
    if (client == null) {
      this.onStatusChange?.(DISCONNECTED_STATUS);
      return;
    }
    try {
      const { channel, useIndexOfChannel, xChannel } = this.props;
      const ch = await fetchChannelProperties(client, channel, useIndexOfChannel);
      const xy =
        !primitive.isZero(xChannel) &&
        !(await client.channels.retrieve(xChannel)).isIndex;
      if (generation !== this.generation) return;
      this.channel = ch;
      this.xy = xy;
      if (this.live) await this.stream(generation, client, ch);
      if (generation !== this.generation) return;
      this.notify();
    } catch (e) {
      if (generation !== this.generation) return;
      this.valid = false;
      this.reportFailure(e, "Failed to read channel data");
    }
  }

  private async stream(
    generation: number,
    client: Client,
    ch: SelectedChannelProperties,
  ): Promise<void> {
    const handler: framer.StreamHandler = (res) => {
      if (generation !== this.generation) return;
      const series = res.get(ch.key);
      if (series == null) return;
      this.pushRaw(series.series);
      this.notify();
    };
    this.stopStreaming?.();
    this.stopStreaming = client.feed.stream(handler, [ch.key]).close;
    if (!ch.virtual || ch.isCalculated) {
      const res = await client.feed.read(
        new TimeRange(this.rawFrom(), this.now()),
        ch.key,
      );
      if (generation !== this.generation) return;
      this.pushRaw(res.series);
    }
    this.rawLoaded = true;
  }

  // feed.read returns the live leading buffer that the stream's first delivery repeats,
  // so series already held by identity are skipped. A late read is inserted by
  // alignment, keeping the series in time order.
  private pushRaw(series: Series[]): void {
    for (const s of series) {
      if (this.raw.series.includes(s)) continue;
      s.acquire();
      const at = this.raw.series.findIndex((held) => held.alignment > s.alignment);
      if (at === -1) this.raw.push(s);
      else this.raw.series.splice(at, 0, s);
    }
  }

  private gcRaw(): void {
    const rawFrom = this.rawFrom();
    const keep = this.raw.series.filter((s) => {
      if (s.timeRange.end.after(rawFrom)) return true;
      s.release();
      return false;
    });
    this.raw.series.splice(0, this.raw.series.length, ...keep);
  }

  private reportFailure(e: unknown, message: string): void {
    const failure = cstatus.fromException(e, message);
    const key = `${failure.message}: ${failure.description}`;
    if (this.lastFailure === key) return;
    this.lastFailure = key;
    this.onStatusChange?.(failure);
  }

  cleanup(): void {
    this.generation++;
    this.stopStreaming?.();
    this.stopStreaming = undefined;
    this.held.forEach((tile) => tile.series.release());
    this.held.clear();
    this.pending.clear();
    this.raw.release();
    this.raw.series.splice(0, this.raw.series.length);
    this.valid = false;
    this.channel = null;
    this.homeLoaded = false;
    this.rawLoaded = false;
    this.lastFailure = undefined;
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
  [TiledChannelData.TYPE]: TiledChannelData,
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

export const tiledChannelData = (props: TiledChannelDataProps): SeriesSourceSpec => ({
  type: TiledChannelData.TYPE,
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
