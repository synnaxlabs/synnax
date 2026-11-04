// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { EOF, Unreachable } from "@synnaxlabs/freighter";
import {
  type Aggregation,
  array,
  type CrudeRate,
  DataType,
  Rate,
  type Reduction,
  Series,
  sleep,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";

import { channel } from "@/channel";
import { AccessDeniedError, NotFoundError, ValidationError } from "@/errors";
import { framer } from "@/framer";

// Samples per series in an unreduced read, matching an iterator response.
const CHUNK = 1e5;
// An alignment domain holds at most 2^32 samples. Each restart starts a new one.
const DOMAIN = 1n << 32n;
const INDEX_KEY = 1;
const HISTORY_CHANGE_KEY = 1 << 20;
// The node that stores every channel of the Core.
const NODE = 1;
// Moves a misaligned data series past every index sample a join remembers.
const MISALIGNMENT = 1n << 24n;
// The Core streams a stored channel in leading alignment domains, apart from the ones
// its reads return.
const LEADING = (1n << 31n) << 32n;

export interface SyntheticCoreProps {
  /** The rate of the index channel. */
  rate: CrudeRate;
  /** The time of the first sample. */
  start: TimeStamp;
  /** The time on the Core's clock. Every sample before it is written. */
  now: TimeStamp;
  /**
   * The number of float32 channels sampled on the index.
   * @default 1
   */
  channels?: number;
  /** Adds a calculated channel with its own index, and a virtual channel. */
  derived?: boolean;
  /** How long the Core takes to ack a stream update after it applies the keys. */
  ackDelay?: TimeSpan;
  /** Streams the data samples in a frame after the frame of their index samples. */
  dataFramedApart?: boolean;
  /** Streams data series with no time range. */
  dataUntimed?: boolean;
  /** Streams data series at alignments that none of their index samples have. */
  dataMisaligned?: boolean;
  /**
   * The text of sample i of a channel retyped to string.
   * @default the sample's index as text
   */
  text?: (i: number) => string;
  /**
   * How far the newest written sample trails the Core's clock, as for a writer that
   * stamps its samples late.
   */
  lag?: TimeSpan;
}

// A span of a channel's samples rewritten with values moved by delta.
interface Rewrite {
  key: channel.Key;
  tr: TimeRange;
  delta: number;
}

/**
 * A Core in memory that serves one index channel and float32 channels sampled on it at
 * a fixed rate. Reads follow the Core's group sizes, group anchors, alignments, and
 * time ranges, but reduced points of a data channel are not reductions of its raw
 * samples. A read costs time in proportion to the points it returns, so years of
 * samples are cheap to hold.
 */
export class SyntheticCore {
  readonly index: channel.Channel;
  readonly data: channel.Channel[];
  /** A calculated channel of data_0, when derived channels were asked for. */
  readonly calculated?: channel.Channel;
  /** The index the calculation writes the calculated channel's timestamps to. */
  readonly calculatedIndex?: channel.Channel;
  /** A virtual channel with no index, when derived channels were asked for. */
  readonly virtual?: channel.Channel;
  /** The channel history changes stream on. */
  readonly historyChange: channel.Channel;
  /** The number of reads the Core has served, reduced or not. */
  reads = 0;
  /** The node the feeds of the Core connect to, or undefined before it is known. */
  node: number | undefined = NODE;
  readonly channels = {
    retrieve: async (key: channel.Key | channel.Name): Promise<channel.Channel> => {
      const ch = this.all().find((c) => c.key === key || c.name === key);
      if (ch == null) throw new NotFoundError(`channel ${key} not found`);
      return ch;
    },
  };

  private readonly start: bigint;
  private readonly period: bigint;
  private clock: TimeStamp;
  private readonly streamers = new Set<SyntheticStreamer>();
  // restarts holds the first sample of each domain after the first.
  private readonly restarts: number[] = [];
  private readonly rewrites: Rewrite[] = [];
  private readonly dataTypes = new Map<channel.Key, DataType>();
  private disconnected = false;
  private readonly ackDelay: TimeSpan;
  private readonly dataFramedApart: boolean;
  private readonly dataUntimed: boolean;
  private readonly dataMisaligned: boolean;
  private readonly text: (i: number) => string;
  private readonly lag: TimeSpan;
  private readonly denied = new Set<channel.Key>();

  constructor({
    rate,
    start,
    now,
    channels = 1,
    derived = false,
    ackDelay = TimeSpan.ZERO,
    dataFramedApart = false,
    dataUntimed = false,
    dataMisaligned = false,
    text = (i) => String(i),
    lag = TimeSpan.ZERO,
  }: SyntheticCoreProps) {
    this.start = start.valueOf();
    this.period = new Rate(rate).period.valueOf();
    this.clock = now;
    this.ackDelay = ackDelay;
    this.dataFramedApart = dataFramedApart;
    this.dataUntimed = dataUntimed;
    this.dataMisaligned = dataMisaligned;
    this.text = text;
    this.lag = lag;
    this.index = new channel.Channel({
      key: INDEX_KEY,
      name: "time",
      dataType: DataType.TIMESTAMP,
      isIndex: true,
      leaseholder: NODE,
    });
    this.data = Array.from(
      { length: channels },
      (_, i) =>
        new channel.Channel({
          key: INDEX_KEY + 1 + i,
          name: `data_${i}`,
          dataType: DataType.FLOAT32,
          index: INDEX_KEY,
          leaseholder: NODE,
        }),
    );
    if (derived) {
      this.calculatedIndex = new channel.Channel({
        key: INDEX_KEY + 3 + channels,
        name: "calculated_time",
        dataType: DataType.TIMESTAMP,
        isIndex: true,
        virtual: true,
      });
      this.calculated = new channel.Channel({
        key: INDEX_KEY + 1 + channels,
        name: "calculated",
        dataType: DataType.FLOAT32,
        index: this.calculatedIndex.key,
        virtual: true,
        expression: "return data_0 * 2",
      });
      this.virtual = new channel.Channel({
        key: INDEX_KEY + 2 + channels,
        name: "virtual",
        dataType: DataType.FLOAT32,
        virtual: true,
      });
    }
    this.historyChange = new channel.Channel({
      key: HISTORY_CHANGE_KEY,
      name: "sy_history_change",
      dataType: DataType.JSON,
      virtual: true,
    });
    this.checkDomain();
  }

  /** @returns the time on the Core's clock. */
  now = (): TimeStamp => this.clock;

  /** Retrieves the payloads of the Core's channels with the given keys or names. */
  retrieveChannels: framer.ChannelRetriever = async (params) => {
    const keys = array.toArray(params) as unknown[];
    return this.all()
      .filter((c) => keys.includes(c.key) || keys.includes(c.name))
      .map((c) => c.payload);
  };

  /** Changes the data type of every sample of a channel, as a recalculation does. */
  retype(key: channel.Key, dataType: DataType): void {
    this.dataTypes.set(key, dataType);
  }

  /** Rejects each stream of the channel, as the Core does for a user without access. */
  deny(key: channel.Key): void {
    this.denied.add(key);
  }

  /**
   * Moves the values of a channel's samples in the time range by delta, and streams the
   * history change.
   */
  rewrite(key: channel.Key, tr: TimeRange, delta: number): void {
    this.rewrites.push({ key, tr, delta });
    const signal = new Series({
      data: [[{ key, timeRange: tr }]],
      dataType: DataType.JSON,
    });
    this.stream(new framer.Frame(this.historyChange.key, signal));
  }

  /** Streams the frame to every open streamer. Reads never return its samples. */
  stream(frame: framer.Frame): void {
    this.streamers.forEach((s) => s.push(frame));
  }

  /**
   * Restarts the Core: open streams end, and samples written from now on start a new
   * alignment domain.
   */
  restart(): void {
    this.restarts.push(this.written());
    this.drop();
  }

  /** Ends every open stream, and fails every read and stream open until reconnect. */
  disconnect(): void {
    this.disconnected = true;
    this.drop();
  }

  /** Lets reads and stream opens through again. */
  reconnect(): void {
    this.disconnected = false;
  }

  /** Opens a feed that reads and streams from the Core, unless props replace them. */
  openFeed(props: Partial<framer.FeedProps> = {}): framer.Feed {
    return new framer.Feed({
      now: this.now,
      readRemote: async (tr, keys, reduction) => this.read(tr, keys, reduction),
      retrieveChannels: this.retrieveChannels,
      openStreamer: async (config) => this.openStreamer(config),
      readLatest: async (keys, count, tr) => this.readLatest(keys, count, tr),
      node: () => this.node,
      ...props,
    });
  }

  /**
   * Advances the clock by the span, and streams the samples written within it to every
   * open streamer.
   * @param streamed - False drops the samples from the stream, as a relay that falls
   * behind does.
   */
  advance(span: TimeSpan, streamed = true): void {
    const from = this.written();
    this.clock = this.clock.add(span);
    this.checkDomain();
    const to = this.written();
    if (to === from || !streamed) return;
    const index = new framer.Frame();
    this.raw(this.index.key, from, to).forEach((s) =>
      index.push(
        this.index.key,
        new Series({ ...seriesProps(s), alignment: s.alignment + LEADING }),
      ),
    );
    const data = this.dataFramedApart ? new framer.Frame() : index;
    this.data.forEach(({ key }) =>
      this.raw(key, from, to).forEach((s) =>
        data.push(
          key,
          new Series({
            ...seriesProps(s),
            timeRange: this.dataUntimed ? TimeRange.ZERO : s.timeRange,
            alignment:
              (this.dataMisaligned ? s.alignment + MISALIGNMENT : s.alignment) +
              LEADING,
          }),
        ),
      ),
    );
    if (this.calculated != null && this.calculatedIndex != null)
      for (const { key } of [this.calculatedIndex, this.calculated])
        this.raw(key, from, to).forEach((s) => index.push(key, s));

    if (this.virtual != null) {
      const { key } = this.virtual;
      // The Core streams each frame of a free channel with no index at alignment 0.
      this.raw(key, from, to).forEach((s) =>
        index.push(
          key,
          new Series({ ...seriesProps(s), timeRange: TimeRange.ZERO, alignment: 0n }),
        ),
      );
    }
    this.streamers.forEach((s) => {
      s.push(index);
      if (data !== index) s.push(data);
    });
  }

  /** Reads the channels over the time range, as the Core's read with a reduction. */
  read(tr: TimeRange, keys: channel.Key[], reduction?: Reduction): framer.Frame {
    this.reads++;
    this.checkConnected();
    if (this.virtual != null && keys.includes(this.virtual.key))
      throw new ValidationError("cannot read from a virtual channel");
    const first = this.samplesBefore(tr.start);
    const last = this.samplesBefore(tr.end);
    if (last <= first) return new framer.Frame();
    if (reduction?.variant === "stride")
      throw new Error("synthetic core does not serve stride reductions");
    const size =
      reduction == null
        ? 1
        : groupSize(last - first, reduction.pointLimit, reduction.aggregation);
    const frame = new framer.Frame();
    for (const key of keys) {
      if (size < 2) {
        for (let c = first; c < last; c += CHUNK) {
          const hi = Math.min(c + CHUNK, last);
          // A chunk ends at the first sample it leaves behind.
          const view = new TimeRange(
            c === first ? tr.start : this.time(c),
            hi === last ? tr.end : this.time(hi),
          );
          this.raw(key, c, hi, view).forEach((s) => frame.push(key, s));
        }
        continue;
      }
      frame.push(
        key,
        this.reduced(key, tr, first, last, size, reduction?.aggregation ?? "min_max"),
      );
    }
    return frame;
  }

  /** Reads the newest count samples of each channel in the time range. */
  readLatest(
    keys: channel.Key[],
    count: number,
    tr: TimeRange = TimeRange.MAX,
  ): framer.Frame {
    this.reads++;
    this.checkConnected();
    const last = this.samplesBefore(tr.end);
    const first = Math.max(this.samplesBefore(tr.start), last - count);
    const frame = new framer.Frame();
    if (last > first)
      keys.forEach((key) =>
        this.raw(key, first, last).forEach((s) => frame.push(key, s)),
      );
    return frame;
  }

  /** @returns true when an open stream carries the channel. */
  streams(key: channel.Key): boolean {
    return [...this.streamers].some((s) => s.keys.includes(key));
  }

  /** Opens a stream of the channels in the config, as the Core's streamer. */
  openStreamer(config: framer.StreamerConfig): framer.Streamer {
    this.checkConnected();
    const streamer = new SyntheticStreamer(
      config,
      this.ackDelay,
      (keys) => this.checkAccess(keys),
      (s) => this.streamers.delete(s),
    );
    this.streamers.add(streamer);
    return streamer;
  }

  // Ends every open stream, as a dropped connection does.
  private drop(): void {
    [...this.streamers].forEach((s) => s.close());
  }

  private time(i: number): bigint {
    return this.start + BigInt(i) * this.period;
  }

  // samplesBefore returns the number of written samples stamped before the time.
  private samplesBefore(t: TimeStamp): number {
    const elapsed = t.valueOf() - this.start;
    if (elapsed <= 0n) return 0;
    const n = Number((elapsed + this.period - 1n) / this.period);
    return Math.min(n, this.written());
  }

  private written(): number {
    const elapsed = this.clock.valueOf() - this.lag.valueOf() - this.start;
    if (elapsed <= 0n) return 0;
    return Number((elapsed + this.period - 1n) / this.period);
  }

  private checkDomain(): void {
    if (this.written() >= Number(DOMAIN))
      throw new Error("synthetic core holds at most 2^32 samples");
  }

  private checkAccess(keys: channel.Key[]): void {
    const denied = keys.find((k) => this.denied.has(k));
    if (denied != null)
      throw new AccessDeniedError(`access to channel ${denied} denied`);
  }

  private checkConnected(): void {
    if (this.disconnected) throw new Unreachable({ message: "synthetic core is down" });
  }

  private all(): channel.Channel[] {
    return [
      this.index,
      ...this.data,
      this.calculatedIndex,
      this.calculated,
      this.virtual,
      this.historyChange,
    ].filter((c) => c != null);
  }

  private isIndex(key: channel.Key): boolean {
    return key === this.index.key || key === this.calculatedIndex?.key;
  }

  private value(key: channel.Key, i: number): number {
    const t = new TimeStamp(this.time(i));
    return this.rewrites.reduce(
      (v, r) => (r.key === key && r.tr.contains(t) ? v + r.delta : v),
      Math.sin(i / 1000 + key),
    );
  }

  private range(from: number, to: number): TimeRange {
    return new TimeRange(this.time(from), this.time(to - 1) + 1n);
  }

  // Returns the samples [from, to) split at the first sample of each domain. A series
  // read through a view spans its domain inside the view, as a Cesium read does.
  private raw(key: channel.Key, from: number, to: number, view?: TimeRange): Series[] {
    const out: Series[] = [];
    for (let lo = from; lo < to;) {
      const domain = this.restarts.filter((r) => r <= lo).length;
      const next = this.restarts.find((r) => r > lo);
      const hi = next == null ? to : Math.min(next, to);
      const first = domain === 0 ? 0 : this.restarts[domain - 1];
      let alignment = DOMAIN * BigInt(domain + 1) + BigInt(lo - first);
      // A calculation sums the alignments of its two inputs' series, and gives its
      // index the same alignments.
      if (key === this.calculated?.key || key === this.calculatedIndex?.key)
        alignment *= 2n;
      const timeRange =
        view == null
          ? this.range(lo, hi)
          : bound(this.range(first, next ?? this.written()), view);
      out.push(this.series(key, lo, hi, alignment, timeRange));
      lo = hi;
    }
    return out;
  }

  private series(
    key: channel.Key,
    from: number,
    to: number,
    alignment: bigint,
    timeRange: TimeRange,
  ): Series {
    const n = to - from;
    if (this.dataTypes.get(key)?.equals(DataType.STRING))
      return new Series({
        data: Array.from({ length: n }, (_, i) => this.text(from + i)),
        dataType: DataType.STRING,
        timeRange,
        alignment,
      });
    let data:
      | BigInt64Array<ArrayBuffer>
      | Float32Array<ArrayBuffer>
      | Float64Array<ArrayBuffer>;
    const dataType = this.isIndex(key)
      ? DataType.TIMESTAMP
      : (this.dataTypes.get(key) ?? DataType.FLOAT32);
    if (this.isIndex(key)) {
      const times = new BigInt64Array(n);
      for (let i = 0; i < n; i++) times[i] = this.time(from + i);
      data = times;
    } else {
      const values = dataType.equals(DataType.FLOAT64)
        ? new Float64Array(n)
        : new Float32Array(n);
      for (let i = 0; i < n; i++) values[i] = this.value(key, from + i);
      data = values;
    }
    return new Series({ data, dataType, timeRange, alignment });
  }

  // reduced returns the samples in [first, last) in groups of size samples anchored at
  // sample zero. A group cut by the read reduces the samples inside it. The index
  // reduces exactly: min_max keeps each group's first and last times, decimate its first,
  // and average its midpoint.
  private reduced(
    key: channel.Key,
    tr: TimeRange,
    first: number,
    last: number,
    size: number,
    aggregation: Aggregation,
  ): Series {
    const minMax = aggregation === "min_max";
    const perGroup = minMax ? 2 : 1;
    const head = Math.floor(first / size);
    const groups = Math.floor((last - 1) / size) - head + 1;
    const isIndex = this.isIndex(key);
    const times = isIndex ? new BigInt64Array(groups * perGroup) : null;
    const values = isIndex ? null : new Float32Array(groups * perGroup);
    for (let g = 0; g < groups; g++) {
      const lo = Math.max((head + g) * size, first);
      const hi = Math.min((head + g + 1) * size, last) - 1;
      const at = g * perGroup;
      if (times != null) {
        const [start, end] = [this.time(lo), this.time(hi)];
        if (minMax) [times[at], times[at + 1]] = [start, end];
        else times[at] = aggregation === "decimate" ? start : (start + end) / 2n;
      } else if (values != null) {
        const v = this.value(key, lo);
        if (minMax) [values[at], values[at + 1]] = [v - 1, v + 1];
        else values[at] = v;
      }
    }
    return new Series({
      data: times ?? values!,
      dataType: isIndex ? DataType.TIMESTAMP : DataType.FLOAT32,
      timeRange: bound(this.range(0, this.written()), tr),
      alignment: DOMAIN + BigInt(head * size),
      alignmentMultiple: BigInt(minMax ? size / 2 : size),
    });
  }
}

// Returns the part of tr inside view.
const bound = (tr: TimeRange, view: TimeRange): TimeRange =>
  new TimeRange(TimeStamp.max(tr.start, view.start), TimeStamp.min(tr.end, view.end));

/**
 * @returns the number of samples per group that brings count samples under the point
 * limit, or 1 when they already fit. Mirrors the Core's group size rule.
 */
const groupSize = (count: number, limit: number, aggregation: Aggregation): number => {
  if (limit === 0 || count <= limit) return 1;
  const decimated = Math.floor((count - 1) / limit) + 1;
  if (decimated <= 1) return 1;
  return aggregation === "min_max" ? decimated * 2 : decimated;
};

class SyntheticStreamer implements framer.Streamer {
  keys: channel.Key[];
  private readonly queue: framer.Frame[] = [];
  // waiting resolves the pending pull with the next frame, or with null on close.
  private waiting: ((frame: framer.Frame | null) => void) | null = null;
  private closed = false;

  constructor(
    config: framer.StreamerConfig,
    private readonly ackDelay: TimeSpan,
    private readonly checkAccess: (keys: channel.Key[]) => void,
    private readonly onClose: (s: SyntheticStreamer) => void,
  ) {
    this.keys = toKeys(framer.streamerConfigZ.parse(config).channels);
    checkAccess(this.keys);
  }

  push(frame: framer.Frame): void {
    if (this.closed) return;
    const filtered = frame.get(this.keys);
    if (filtered.series.length === 0) return;
    if (this.waiting == null) return void this.queue.push(filtered);
    this.waiting(filtered);
    this.waiting = null;
  }

  // The keys apply at once, so frames of new keys can arrive before the ack.
  async update(
    channels: channel.Params,
    _?: framer.StreamerUpdateOptions,
  ): Promise<void> {
    const keys = toKeys(channels);
    this.checkAccess(keys);
    this.keys = keys;
    if (!this.ackDelay.isZero) await sleep.sleep(this.ackDelay);
  }

  async read(): Promise<framer.Frame> {
    const frame = await this.pull();
    if (frame == null) throw new EOF();
    return frame;
  }

  async next(): Promise<IteratorResult<framer.Frame>> {
    const frame = await this.pull();
    if (frame == null) return { done: true, value: undefined };
    return { done: false, value: frame };
  }

  close(): void {
    this.closed = true;
    this.waiting?.(null);
    this.waiting = null;
    this.onClose(this);
  }

  [Symbol.asyncIterator](): AsyncIterator<framer.Frame> {
    return this;
  }

  private async pull(): Promise<framer.Frame | null> {
    if (this.closed) return null;
    const next = this.queue.shift();
    if (next != null) return next;
    return await new Promise((resolve) => (this.waiting = resolve));
  }
}

const seriesProps = (s: Series) => ({
  data: s.data,
  dataType: s.dataType,
  timeRange: s.timeRange,
  alignment: s.alignment,
});

const toKeys = (channels: channel.Params): channel.Key[] =>
  array.toArray(channels).map((c) => {
    if (typeof c === "number") return c;
    throw new Error("synthetic core streams channels by key only");
  });
