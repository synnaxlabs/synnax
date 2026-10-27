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
  type Aggregation,
  aggregationZ,
  bounds,
  breaker,
  DataType,
  type destructor,
  errors,
  MultiSeries,
  primitive,
  resolveWindow,
  type Series,
  sync,
  TimeRange,
  TimeSpan,
  TimeStamp,
  windowZ,
} from "@synnaxlabs/x";
import { z } from "zod";

import { type status } from "@/status/aether";
import { type CreateOptions } from "@/telem/aether/factory";
import {
  AbstractSource,
  LOADING_STATUS,
  type NumberSource,
  type NumberSourceSpec,
  type Result,
  type SeriesSource,
  type SeriesSourceSpec,
  type Spec,
  type StringSource,
  type StringSourceSpec,
  success,
  SUCCESS_STATUS,
  type Telem,
  type ValueProps,
} from "@/telem/aether/telem";
import {
  choose,
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
  /** @returns the current time on the Core's clock, which stamps every sample. */
  now: () => TimeStamp;
}

/** Reported by remote sources created while the cluster is disconnected. */
export const DISCONNECTED_STATUS: cstatus.Crude = {
  variant: "warning",
  message: "Core disconnected",
};

// Retrying only fixes connectivity; a definitive rejection recurs on every attempt.
const isDefinitive = (e: unknown): boolean =>
  AccessDeniedError.matches(e) ||
  ValidationError.matches(e) ||
  NotFoundError.matches(e);

/**
 * Backs off a source's read retries. A tall interval cap suits a source whose retries
 * only refresh history. Notifying the notifier interrupts a pending backoff, so a
 * retired source's retry loop ends promptly.
 */
const newRetryBreaker = (
  notifier: sync.Notifier,
  config?: breaker.Config,
): breaker.Breaker =>
  new breaker.Breaker({
    baseInterval: TimeSpan.seconds(1),
    maxInterval: TimeSpan.seconds(30),
    maxRetries: Infinity,
    scale: 2,
    jitter: 0.25,
    sleepFn: async (duration) => await notifier.wait(duration),
    ...config,
  });

/**
 * Posts a source's failures. Retries repeat at the breaker's pace, so an incident
 * posts one status when it starts and again only when the failure changes.
 */
class FailureReporter {
  /** The last failure since the last reset. */
  status?: cstatus.Status;
  constructor(private readonly onStatusChange?: status.Adder) {}

  report(e: unknown, message: string): void {
    const prev = this.status;
    this.status = cstatus.fromException(e, message);
    if (
      prev?.message === this.status.message &&
      prev.description === this.status.description
    )
      return;
    this.onStatusChange?.(this.status);
  }

  reset(): void {
    this.status = undefined;
  }
}

// Loading until a source's first read lands, then its last failure, if any.
const sourceStatus = (loading: boolean, failures: FailureReporter): cstatus.Status =>
  loading ? LOADING_STATUS : (failures.status ?? SUCCESS_STATUS);

// A failure goes to onError and reads as null.
const readLatestSeries = async (
  client: Client,
  ch: channel.Channel,
  onError: (e: unknown) => void,
): Promise<Series | null> => {
  if (ch.virtual && !channel.isCalculated(ch)) return null;
  try {
    const latest = (await client.feed.readLatest(ch.key)).series.at(-1);
    if (latest == null || latest.length === 0) return null;
    return latest;
  } catch (e) {
    onError(e);
    return null;
  }
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
  private time?: TimeStamp;
  private generation = 0;
  private valid = false;
  private loading = true;
  private readonly failures: FailureReporter;
  private readonly onStatusChange?: status.Adder;
  constructor(client: Client | null, props: unknown, options?: CreateOptions) {
    super(props);
    this.client = client;
    this.onStatusChange = options?.onStatusChange;
    this.failures = new FailureReporter(this.onStatusChange);
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
    this.time = undefined;
    this.removeStreamHandler = null;
    this.loading = true;
    this.failures.reset();
  }

  value(): Result<number> {
    // No valid channel has been set.
    if (primitive.isZero(this.props.channel)) return success(NaN);
    if (!this.valid) void this.read();
    const status = sourceStatus(this.loading, this.failures);
    // No data has been received and no recent samples were fetched on initialization.
    if (this.leadingBuffer == null || this.leadingBuffer.length === 0)
      return { data: NaN, status };
    const v = this.leadingBuffer.at(-1, true);
    // A boolean channel reads as 0 or 1, and an int64 channel as its nearest number.
    const data = Number(v);
    return { data, status, time: this.time };
  }

  /** Never rejects: a failure invalidates the read and reaches onStatusChange. */
  private async read(): Promise<void> {
    const generation = this.generation;
    this.valid = true;
    const { client } = this;
    if (client == null) {
      this.loading = false;
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
          this.time = undefined;
        }
        // Just because we didn't get a new buffer doesn't mean one wasn't allocated: an
        // empty update means the leading buffer was appended to in place. A frame that
        // holds nothing for this channel looks the same, so with no buffer yet there is
        // nothing to report.
        else if (this.leadingBuffer == null) return;
        this.loading = false;
        this.failures.reset();
        this.notify();
      };
      if (generation !== this.generation) return;
      this.removeStreamHandler = client.feed.stream(handler, [ch.key]).close;
      this.failures.reset();
      // Opening the stream is not a sample. Notify only when a buffer already holds
      // one, so a consumer that counts arrivals does not count the open.
      if (this.leadingBuffer != null && this.leadingBuffer.length > 0) {
        this.notify();
        return;
      }
      const latest = await readLatestSeries(client, ch, (e) => {
        // A live sample or a cleanup makes the failed read moot.
        if (generation === this.generation && this.leadingBuffer == null)
          this.failures.report(e, "Failed to read latest value");
      });
      if (generation !== this.generation) return;
      this.loading = false;
      if (latest == null || this.leadingBuffer != null) return;
      latest.acquire();
      this.leadingBuffer = latest;
      this.time = latest.timeRange.isZero ? undefined : latest.timeRange.end;
      this.notify();
    } catch (e) {
      if (generation !== this.generation) return;
      this.valid = false;
      this.loading = false;
      this.failures.report(e, "Failed to stream channel value");
    }
  }
}

interface SelectedChannelProperties extends Pick<
  channel.Payload,
  "key" | "dataType" | "virtual"
> {
  isCalculated: boolean;
}

const selectChannel = (
  c: channel.Channel,
  useIndex: boolean,
): SelectedChannelProperties => {
  const isCalculated = channel.isCalculated(c);
  if (!useIndex || c.isIndex)
    return { key: c.key, dataType: c.dataType, virtual: c.virtual, isCalculated };
  if (c.virtual && !isCalculated)
    throw new NotFoundError("cannot use virtual channels as a data source");
  return { key: c.index, dataType: DataType.TIMESTAMP, virtual: false, isCalculated };
};

// pairOf returns the channel that a line draws the source's channel against: y for x,
// and x for y. It returns null for an x source whose line did not give its y.
const pairOf = (
  c: channel.Channel,
  useIndexOfChannel: boolean,
  x: channel.Channel | null,
  y: channel.Key,
): channel.Key | null => {
  if (useIndexOfChannel) return c.key;
  if (x == null) return primitive.isZero(c.index) ? c.key : c.index;
  if (x.key !== c.key) return x.key;
  return primitive.isZero(y) ? null : y;
};

// How long a seam waits before its read, matching the margin the cache gives commits.
const SEAM_SETTLE = TimeSpan.seconds(1);

// The span a live source's back-fill and stream both miss.
interface Seam {
  backfillEnd?: TimeStamp;
  liveStart?: TimeStamp;
  filled: boolean;
}

interface LiveWindowConfig {
  now: () => TimeStamp;
  /** @returns where the back-fill starts. */
  start: () => TimeStamp;
  /** A held series that ends at or before cutoff() is released. */
  cutoff: () => TimeStamp;
  failures: FailureReporter;
  breakerConfig?: breaker.Config;
  notify: () => void;
  /** @returns the data type of the series the source holds, or undefined if none. */
  dataType: () => DataType | undefined;
  /** Drops everything the source holds when the channel's data type changes. */
  restart: () => void;
}

/** The newest series of a live source at full resolution, held in time order. */
class LiveWindow {
  readonly data = new MultiSeries([]);
  private generation = 0;
  private stopStreaming?: destructor.Destructor;
  private readonly notifier = new sync.Notifier();
  // seam belongs to the open stream. Its liveStart is where the line's streamed series
  // start to pair.
  private seam: Seam = { filled: false };
  // streamed holds the series the stream delivered.
  private readonly streamed = new WeakSet<Series>();

  constructor(private readonly config: LiveWindowConfig) {}

  /**
   * Opens the stream, then back-fills from start(), so a stalled or failed back-fill
   * never blocks live data. Resolves when the back-fill lands.
   * @param pair - The channel the line draws this one against, or null if unknown.
   * @throws {Error} if the back-fill fails. The caller reports the failure.
   */
  async open(
    client: Client,
    ch: SelectedChannelProperties,
    pair: channel.Key | null,
  ): Promise<void> {
    const generation = this.generation;
    const seam: Seam = { filled: false };
    this.seam = seam;
    const keys = pair == null || pair === ch.key ? [ch.key] : [ch.key, pair];
    // firsts holds where the first streamed series of each key starts.
    const firsts = new Map<channel.Key, TimeStamp>();
    const handler: framer.StreamHandler = (res) => {
      if (generation !== this.generation) return;
      const series = res.get(ch.key);
      if (series != null) {
        for (const s of series.series) this.streamed.add(s);
        if (!this.push(series.series)) return;
      }
      for (const key of keys) {
        if (firsts.has(key)) continue;
        const first = res.get(key)?.series ?? [];
        if (first.length > 0)
          firsts.set(key, TimeStamp.min(...first.map((s) => s.timeRange.start)));
      }
      if (seam.liveStart == null && firsts.size === keys.length) {
        seam.liveStart = TimeStamp.max(...firsts.values());
        void this.fillSeam(generation, client, ch.key, seam);
      }
      if (series == null) return;
      this.config.notify();
      this.prune();
    };
    this.stopStreaming?.();
    const streamStart = this.config.now();
    this.stopStreaming = client.feed.stream(handler, keys).close;
    if (ch.virtual && !ch.isCalculated) return;
    try {
      // Streamed series never pair with fetched ones, so x and y end their back-fills
      // where the later of their streams starts. A stream that has not started counts
      // as starting where this one opened. Without a known pair, the back-fill ends
      // there.
      const held = this.data.series.reduce(
        (start, s) => (s.timeRange.start.before(start) ? s.timeRange.start : start),
        streamStart,
      );
      const starts = keys.map((k) =>
        k === ch.key ? held : (firsts.get(k) ?? streamStart),
      );
      const liveStart = pair == null ? streamStart : TimeStamp.max(...starts);
      // Live data held across a long retry can start before the window.
      const start = this.config.start();
      if (!liveStart.after(start)) return;
      const res = await client.feed.read(new TimeRange(start, liveStart), ch.key);
      if (generation !== this.generation || !this.push(res.series)) return;
      seam.backfillEnd = liveStart;
      void this.fillSeam(generation, client, ch.key, seam);
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
  }

  // Reads the span between the back-fill's end and the later first streamed series of
  // the channel and its pair once both are known. The stream delivers only samples
  // written after it opens, which is later than the back-fill's end.
  private async fillSeam(
    generation: number,
    client: Client,
    key: channel.Key,
    seam: Seam,
  ): Promise<void> {
    const { backfillEnd, liveStart } = seam;
    if (backfillEnd == null || liveStart == null || seam.filled) return;
    seam.filled = true;
    if (!liveStart.after(backfillEnd)) return;
    // Samples written just before the stream opened may still be committing.
    const settled = liveStart.add(SEAM_SETTLE);
    const { now } = this.config;
    // A wake saved by an earlier clear ends a wait early, so recheck the clock.
    for (let t = now(); t.before(settled); t = now()) {
      await this.notifier.wait(settled.span(t));
      if (generation !== this.generation) return;
    }
    const brk = newRetryBreaker(this.notifier, this.config.breakerConfig);
    while (generation === this.generation)
      try {
        const res = await client.feed.read(new TimeRange(backfillEnd, liveStart), key);
        if (generation !== this.generation) return;
        if (this.push(res.series)) this.config.notify();
        return;
      } catch (e) {
        this.config.failures.report(e, "Failed to read channel data");
        if (isDefinitive(e) || !(await brk.wait())) return;
      }
  }

  // feed.read returns the live leading buffer that the stream's first delivery repeats,
  // so series already held by identity are skipped. Consumers assume the array is
  // chronological, so a late back-fill is inserted by time. Alignment would misorder
  // the buffers of a restarted Core, which resets its leading alignment. Returns false
  // when a series of a new data type restarts the source.
  private push(series: Series[]): boolean {
    const held = this.config.dataType();
    // Series of two data types cannot share a MultiSeries.
    if (held != null && series.some((s) => !s.dataType.equals(held))) {
      this.config.restart();
      return false;
    }
    for (const s of series) {
      if (this.data.series.includes(s)) continue;
      s.acquire();
      const at = this.data.series.findIndex(
        (held) =>
          held.timeRange.start.after(s.timeRange.start) ||
          (held.timeRange.start.equals(s.timeRange.start) &&
            held.alignment > s.alignment),
      );
      if (at === -1) this.data.push(s);
      else this.data.series.splice(at, 0, s);
    }
    return true;
  }

  /**
   * @returns where the held series that end after the boundary start to draw, or
   * TimeStamp.MAX if none does. A streamed series draws from the line's live start at
   * the earliest, since its earlier samples have nothing to pair with.
   */
  drawsFrom(boundary: TimeStamp): TimeStamp {
    const { liveStart } = this.seam;
    let from = TimeStamp.MAX;
    for (const s of this.data.series) {
      const { start, end } = s.timeRange;
      if (!start.before(from)) break;
      if (!end.after(boundary)) continue;
      if (!this.streamed.has(s)) from = start;
      else if (liveStart != null)
        from = TimeStamp.min(from, TimeStamp.max(start, liveStart));
    }
    return from;
  }

  /** Releases the series that end at or before the cutoff. */
  prune(): void {
    const cutoff = this.config.cutoff();
    const kept = this.data.series.filter((s) => {
      if (s.timeRange.end.after(cutoff)) return true;
      s.release();
      return false;
    });
    this.data.series.splice(0, this.data.series.length, ...kept);
  }

  /** Stops the stream and releases every held series. Reads in flight are dropped. */
  clear(): void {
    this.generation++;
    this.notifier.notify();
    this.stopStreaming?.();
    this.stopStreaming = undefined;
    this.data.release();
    this.data.series.splice(0);
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
  private time?: TimeStamp;
  private latest = "";
  // Buffer length the latest decode was taken at, or -1 to force a re-decode.
  private decodedAt = -1;
  // Bumped by cleanup to invalidate a read that is still awaiting.
  private generation = 0;
  private valid = false;
  private loading = true;
  private readonly failures: FailureReporter;
  private readonly onStatusChange?: status.Adder;
  constructor(client: Client | null, props: unknown, options?: CreateOptions) {
    super(props);
    this.client = client;
    this.onStatusChange = options?.onStatusChange;
    this.failures = new FailureReporter(this.onStatusChange);
  }

  cleanup(): void {
    this.generation++;
    this.removeStreamHandler?.();
    this.valid = false;
    this.leadingBuffer?.release();
    this.leadingBuffer = null;
    this.time = undefined;
    this.latest = "";
    this.decodedAt = -1;
    this.removeStreamHandler = null;
    this.loading = true;
    this.failures.reset();
  }

  value(): Result<string> {
    // No valid channel has been set.
    if (primitive.isZero(this.props.channel)) return success("");
    if (!this.valid) void this.read();
    const status = sourceStatus(this.loading, this.failures);
    const buffer = this.leadingBuffer;
    if (buffer == null || buffer.length === 0) return { data: "", status };
    // Samples are appended into the leading buffer in place, so the length is what
    // marks new data. Decoding is gated on it because asString scans linearly.
    if (buffer.length !== this.decodedAt) {
      this.latest = buffer.asString(-1) ?? this.latest;
      this.decodedAt = buffer.length;
    }
    return { data: this.latest, status, time: this.time };
  }

  /** Never rejects: a failure invalidates the read and reaches onStatusChange. */
  private async read(): Promise<void> {
    const generation = this.generation;
    this.valid = true;
    const { client } = this;
    if (client == null) {
      this.loading = false;
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
          this.time = undefined;
          this.decodedAt = -1;
        }
        // An empty update means the leading buffer was appended to in place. A frame
        // that holds nothing for this channel looks the same, so with no buffer yet
        // there is nothing to report.
        else if (this.leadingBuffer == null) return;
        this.loading = false;
        this.failures.reset();
        this.notify();
      };
      if (generation !== this.generation) return;
      this.removeStreamHandler = client.feed.stream(handler, [ch.key]).close;
      this.failures.reset();
      // Opening the stream is not a sample. Notify only when a buffer already holds
      // one, so a consumer that counts arrivals does not count the open.
      if (this.leadingBuffer != null && this.leadingBuffer.length > 0) {
        this.notify();
        return;
      }
      const latest = await readLatestSeries(client, ch, (e) => {
        // A live sample or a cleanup makes the failed read moot.
        if (generation === this.generation && this.leadingBuffer == null)
          this.failures.report(e, "Failed to read latest value");
      });
      if (generation !== this.generation) return;
      this.loading = false;
      if (latest == null || this.leadingBuffer != null) return;
      latest.acquire();
      this.leadingBuffer = latest;
      this.time = latest.timeRange.isZero ? undefined : latest.timeRange.end;
      this.decodedAt = -1;
      this.notify();
    } catch (e) {
      if (generation !== this.generation) return;
      this.valid = false;
      this.loading = false;
      this.failures.report(e, "Failed to stream channel string value");
    }
  }
}

const tiledChannelDataPropsZ = z.object({
  channel: z.number().or(z.string()),
  useIndexOfChannel: z.boolean().default(false),
  /** The line's x channel, or zero when x is the index of the line's y channel. */
  xChannel: z.number().or(z.string()).default(0),
  /** The key of the line's y channel, which an x source draws against. */
  yChannel: z.number().default(0),
  /** The home view: a fixed range, or a span that ends now for a live line. */
  window: windowZ,
  aggregation: aggregationZ.default("min_max"),
});

export type TiledChannelDataProps = z.input<typeof tiledChannelDataPropsZ>;

interface HeldTile {
  spec: framer.TileSpec;
  series: MultiSeries;
}

// The points tiles are sized for before a read gives its own.
const DEFAULT_POINTS = 1000;

const viewRange = ({ lower, upper }: bounds.Bounds): TimeRange =>
  new TimeRange(BigInt(Math.max(Math.floor(lower), 0)), BigInt(Math.ceil(upper)));

/**
 * A series source that draws tiles of reduced data covering the view passed to value(),
 * or the home view when none is passed. A live line streams its newest data at full
 * resolution. An xy line reads its home view once, reduced with decimate.
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
  private readonly held = new Map<string, HeldTile>();
  private readonly pending = new Set<string>();
  private points = DEFAULT_POINTS;
  // window holds a live line's full-resolution series from boundary on.
  private readonly window: LiveWindow;
  // boundary is where a live line's tiles end and its full-resolution data begins. It
  // trails rawFrom until the tiles that replace the series before rawFrom arrive, so
  // their span never goes blank. It trails by at most one tile of the home level.
  private boundary = TimeStamp.ZERO;
  // drawn is the view of the last read that drew.
  private drawn?: TimeRange;
  private rawLoaded = false;
  // loading is true until the home view first loads or a read fails.
  private loading = true;
  private readonly failures: FailureReporter;
  private readonly retryNotifier = new sync.Notifier();
  // breaker paces retries of the channel and live reads, and tileBreaker paces retries
  // of failed tiles.
  private readonly breaker: breaker.Breaker;
  private readonly tileBreaker: breaker.Breaker;
  // retrying holds the keys of failed tiles that wait for tileBreaker, and parked holds
  // the keys of tiles a definitive rejection stopped.
  private readonly retrying = new Set<string>();
  private readonly parked = new Set<string>();

  constructor(
    client: Client | null,
    props: unknown,
    options?: CreateOptions,
    now: () => TimeStamp = client?.now ?? (() => TimeStamp.now()),
    breakerConfig?: breaker.Config,
  ) {
    super(props);
    this.client = client;
    this.now = now;
    this.onStatusChange = options?.onStatusChange;
    this.failures = new FailureReporter(options?.onStatusChange);
    this.breaker = newRetryBreaker(this.retryNotifier, breakerConfig);
    this.tileBreaker = newRetryBreaker(this.retryNotifier, breakerConfig);
    this.window = new LiveWindow({
      now,
      start: () => this.rawFrom(),
      cutoff: () => this.cutoff(),
      failures: this.failures,
      breakerConfig,
      notify: () => this.notify(),
      dataType: () => this.heldDataType(),
      restart: () => this.restart(),
    });
  }

  private get live(): boolean {
    return this.props.window.variant === "dynamic";
  }

  value({ bounds: view, points, extremesKept = false }: ValueProps = {}): Result<
    [bounds.Bounds, MultiSeries]
  > {
    if (points != null) this.points = points;
    if (primitive.isZero(this.props.channel))
      return success([bounds.INVALID, new MultiSeries()]);
    if (!this.valid) void this.resolve();
    const ch = this.channel;
    if (ch == null) return this.result([bounds.INVALID, new MultiSeries()]);
    const home = this.home();
    const drawsView =
      view != null && !this.xy && bounds.isFinite(view) && bounds.span(view) > 0;
    // Any read with bounds draws, so it releases the tiles the line no longer needs.
    const series = this.select(
      ch.key,
      drawsView ? viewRange(view) : home,
      view != null,
      view == null || points != null,
      extremesKept,
    );
    if (this.loading && this.loaded(ch.key, home)) this.loading = false;
    return this.result([this.bounds(ch, home, series), series]);
  }

  private result(
    data: [bounds.Bounds, MultiSeries],
  ): Result<[bounds.Bounds, MultiSeries]> {
    const loading = this.loading || this.pending.size > 0;
    return {
      data,
      status: sourceStatus(loading, this.failures),
      aggregation: this.aggregation(),
    };
  }

  private home(): TimeRange {
    return resolveWindow(this.props.window, this.now());
  }

  // rawFrom is where a live line's full-resolution data starts. It sits on a tile
  // boundary of the home level, so no tile of the home level or finer crosses it.
  private rawFrom(): TimeStamp {
    const home = this.home();
    if (this.xy) return home.start;
    const span = framer.tileSpan(level(home)).valueOf();
    return new TimeStamp((home.start.valueOf() / span) * span);
  }

  // cutoff returns the boundary after it moves to at most one home level tile behind
  // rawFrom. The stream prunes with it, so a line that never draws stops growing.
  private cutoff(): TimeStamp {
    const home = this.home();
    const floor = this.xy
      ? home.start
      : this.rawFrom().sub(framer.tileSpan(level(home)));
    this.boundary = TimeStamp.max(this.boundary, floor);
    return this.boundary;
  }

  // tilesEnd is where a live line's tiles stop for the boundary: the boundary, or where
  // a held raw series that crosses it starts to draw, so no span draws twice.
  private tilesEnd(boundary: TimeStamp = this.boundary): TimeStamp {
    return TimeStamp.min(boundary, this.window.drawsFrom(boundary));
  }

  private aggregation(): Aggregation {
    return this.xy ? "decimate" : this.props.aggregation;
  }

  private targets(
    key: channel.Key,
    view: TimeRange,
    extremesKept = false,
    boundary?: TimeStamp,
  ): framer.TileSpec[] {
    let aggregation = this.aggregation();
    // Min-max keeps the extremes that the other aggregations drop. An xy line draws its
    // home tiles at every zoom, so they bound it.
    if (extremesKept && !this.xy) aggregation = "min_max";
    // Tiles are cut to [start, end): the home view for an xy line, and the start of
    // full-resolution data for a live line.
    let start: TimeStamp | undefined;
    let end: TimeStamp | undefined;
    let tiled = view;
    if (this.xy) {
      if (this.live) return [];
      [start, end] = [this.home().start, this.home().end];
    } else if (this.live) {
      end = this.tilesEnd(boundary);
      if (!end.after(view.start)) return [];
      if (view.end.after(end)) tiled = new TimeRange(view.start, end);
    }
    const l = level(view);
    const limit = pointLimit({
      level: l,
      view,
      points: this.points,
      drawn: this.aggregation(),
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

  // find returns the held tile at the position: the preferred one, then one of the
  // aggregation, then one of another.
  private find(
    p: Position,
    preferred: Map<string, string>,
    aggregation: Aggregation,
    tiles: Iterable<HeldTile> = this.held.values(),
  ): HeldTile | undefined {
    const want = preferred.get(`${p.level}/${p.index}`);
    if (want != null) {
      const exact = this.held.get(want);
      if (exact != null) return exact;
    }
    let other: HeldTile | undefined;
    for (const tile of tiles) {
      if (tile.spec.level !== p.level || tile.spec.index !== p.index) continue;
      if (tile.spec.aggregation === aggregation) return tile;
      other ??= tile;
    }
    return other;
  }

  // advance moves the boundary to rawFrom once the tiles that replace the raw series
  // before rawFrom in the drawn view arrive. It returns the keys of those tiles while
  // any is missing.
  private advance(key: channel.Key, reads: boolean): string[] {
    const rawFrom = this.rawFrom();
    if (!rawFrom.after(this.boundary)) return [];
    const next = this.targets(key, this.drawn ?? this.home(), false, rawFrom);
    const keys = next.map((spec) => framer.tileRequestKey(spec));
    if (keys.every((k) => this.held.has(k))) {
      this.boundary = rawFrom;
      return [];
    }
    if (reads) next.forEach((spec) => this.request(spec));
    return keys;
  }

  private select(
    key: channel.Key,
    view: TimeRange,
    prune: boolean,
    reads: boolean,
    extremesKept: boolean,
  ): MultiSeries {
    if (prune) this.drawn = view;
    const next = this.live ? this.advance(key, reads) : [];
    if (this.live) this.window.prune();
    const targets = this.targets(key, view, extremesKept);
    if (reads) targets.forEach((spec) => this.request(spec));
    const preferred = new Map(
      targets.map((t) => [`${t.level}/${t.index}`, framer.tileRequestKey(t)]),
    );
    // A held tile that reaches past tilesEnd covers raw series the line still draws, so
    // it waits until the boundary moves.
    const end = this.live ? this.tilesEnd() : null;
    const drawable = [...this.held.values()].filter(
      ({ spec }) => end == null || !framer.tileRange(spec).end.after(end),
    );
    const chosen =
      targets.length === 0
        ? []
        : choose(
            targets[0].level,
            targets.map((t) => t.index),
            drawable.map((t) => t.spec),
          ).map((p) => this.find(p, preferred, targets[0].aggregation, drawable)!);
    const out = chosen.flatMap((tile) => tile.series.series);
    if (prune) {
      // The tiles of the home view and of its bounds stay held, and so do the tiles
      // that stand in for them until they arrive.
      const keep = new Set([
        ...chosen.map((t) => framer.tileRequestKey(t.spec)),
        ...preferred.values(),
        ...next,
        ...this.homeKeys(key, false),
        ...this.homeKeys(key, true),
      ]);
      for (const [tileKey, tile] of this.held)
        if (!keep.has(tileKey)) {
          tile.series.release();
          this.held.delete(tileKey);
        }
    }
    if (this.live) out.push(...this.window.data.series);
    return new MultiSeries(out);
  }

  // homeKeys returns the keys of the home tiles, and of the held tiles that stand in
  // for them.
  private homeKeys(key: channel.Key, extremesKept: boolean): string[] {
    const targets = this.targets(key, this.home(), extremesKept);
    const preferred = new Map(
      targets.map((t) => [`${t.level}/${t.index}`, framer.tileRequestKey(t)]),
    );
    return [
      ...preferred.values(),
      ...targets.flatMap((t) => {
        const found = this.find(t, preferred, t.aggregation);
        return found == null ? [] : [framer.tileRequestKey(found.spec)];
      }),
    ];
  }

  private loaded(key: channel.Key, home: TimeRange): boolean {
    if (this.live && !this.rawLoaded) return false;
    // Loading also waits for the min_max tiles that a bounds read has in flight.
    const bounding = this.targets(key, home, true).some((spec) =>
      this.pending.has(framer.tileRequestKey(spec)),
    );
    return (
      !bounding &&
      this.targets(key, home).every((spec) =>
        this.held.has(framer.tileRequestKey(spec)),
      )
    );
  }

  private bounds(
    ch: SelectedChannelProperties,
    home: TimeRange,
    series: MultiSeries,
  ): bounds.Bounds {
    if (!ch.dataType.equals(DataType.TIMESTAMP) || this.xy) return series.bounds;
    if (!this.live) return home.numericBounds;
    const { data } = this.window;
    const latest = data.series.length > 0 ? data.bounds.upper : NaN;
    const upper = Number.isFinite(latest) ? latest : Number(home.end.valueOf());
    return { lower: upper - Number(home.span.valueOf()), upper };
  }

  // A failed tile waits for tileBreaker before a later value() call requests it again.
  // A definitive rejection parks it until cleanup.
  private request(spec: framer.TileSpec): void {
    const key = framer.tileRequestKey(spec);
    const { client } = this;
    if (
      client == null ||
      this.held.has(key) ||
      this.pending.has(key) ||
      this.retrying.has(key) ||
      this.parked.has(key)
    )
      return;
    const generation = this.generation;
    this.pending.add(key);
    client.feed
      .readTile(spec, this.heldDataType())
      .then((series) => {
        if (generation !== this.generation) return;
        this.pending.delete(key);
        const held = this.heldDataType();
        // Series of two data types cannot share a MultiSeries.
        if (held != null && series.series.some((s) => !s.dataType.equals(held)))
          return this.restart();
        series.acquire();
        this.held.set(key, { spec, series });
        this.tileBreaker.reset();
        this.failures.reset();
        this.notify();
      })
      .catch((e: unknown) => {
        if (generation !== this.generation) return;
        this.pending.delete(key);
        this.fail(e);
        if (isDefinitive(e)) {
          this.parked.add(key);
          return;
        }
        const waiting = this.retrying.size > 0;
        this.retrying.add(key);
        if (!waiting) void this.retryTiles(generation);
      });
  }

  // Waits out tileBreaker for the tiles that fail during the wait. The notify at its
  // end makes the next value() call request them again.
  private async retryTiles(generation: number): Promise<void> {
    if (!(await this.tileBreaker.wait()) || generation !== this.generation) return;
    this.retrying.clear();
    this.notify();
  }

  /**
   * Never rejects. A connectivity failure retries under the breaker, and a definitive
   * rejection parks the source. Every distinct failure reaches onStatusChange.
   */
  private async resolve(): Promise<void> {
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
        this.breaker.reset();
        this.failures.reset();
        return;
      } catch (e) {
        if (generation !== this.generation) return;
        this.fail(e);
        if (isDefinitive(e) || !(await this.breaker.wait())) return;
      }
  }

  private async attempt(generation: number, client: Client): Promise<void> {
    const { channel, useIndexOfChannel, xChannel, yChannel } = this.props;
    const c = await client.channels.retrieve(channel);
    const ch = selectChannel(c, useIndexOfChannel);
    const x = primitive.isZero(xChannel)
      ? null
      : await client.channels.retrieve(xChannel);
    if (generation !== this.generation) return;
    this.channel = ch;
    this.xy = x != null && !x.isIndex;
    if (this.live) {
      this.boundary = this.rawFrom();
      await this.window.open(client, ch, pairOf(c, useIndexOfChannel, x, yChannel));
      if (generation !== this.generation) return;
      this.rawLoaded = true;
    }
    this.notify();
  }

  // Declares the home view loaded on the first failure, so retries never hold loading
  // forever.
  private fail(e: unknown): void {
    this.failures.report(e, "Failed to read channel data");
    this.declareLoaded();
  }

  // Notifies so a failure still wakes observers of the status.
  private declareLoaded(): void {
    if (!this.loading) return;
    this.loading = false;
    this.notify();
  }

  // The data type of every series the line holds, or undefined when it holds none.
  private heldDataType(): DataType | undefined {
    for (const tile of this.held.values())
      if (tile.series.series.length > 0) return tile.series.series[0].dataType;
    return this.window.data.series[0]?.dataType;
  }

  private restart(): void {
    this.cleanup();
    this.notify();
  }

  cleanup(): void {
    this.generation++;
    this.retryNotifier.notify();
    this.drawn = undefined;
    this.window.clear();
    this.held.forEach((tile) => tile.series.release());
    this.held.clear();
    this.pending.clear();
    this.retrying.clear();
    this.parked.clear();
    this.valid = false;
    this.channel = null;
    this.loading = true;
    this.rawLoaded = false;
    this.failures.reset();
  }
}

type Constructor = new (
  client: Client | null,
  props: unknown,
  options?: CreateOptions,
) => Telem;

const REGISTRY: Record<string, Constructor> = {
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
