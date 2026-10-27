// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { EOF } from "@synnaxlabs/freighter";
import {
  type Aggregation,
  array,
  type CrudeRate,
  DataType,
  Rate,
  type Reduction,
  Series,
  TimeRange,
  type TimeSpan,
  type TimeStamp,
} from "@synnaxlabs/x";

import { channel } from "@/channel";
import { NotFoundError } from "@/errors";
import { Feed, type FeedOptions } from "@/framer/feed";
import { Frame } from "@/framer/frame";
import { type Streamer, type StreamerConfig, streamerConfigZ } from "@/framer/streamer";

// Samples per series in an unreduced read, matching an iterator response.
const CHUNK = 1e5;
// Every sample shares one alignment domain, which holds at most 2^32 samples.
const DOMAIN = 1n << 32n;
// Streamed samples carry the Core's provisional leading alignments, which never pair
// with the alignments of the same samples read back.
const LEADING_DOMAIN = (1n << 31n) << 32n;
const INDEX_KEY = 1;

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
}

/**
 * A Core in memory that serves one index channel and float32 channels sampled on it at a
 * fixed rate. Reads follow the Core's group sizes, group anchors, and alignments, but
 * reduced points of a data channel are not reductions of its raw samples. A read costs
 * time in proportion to the points it returns, so years of samples are cheap to hold.
 */
export class SyntheticCore {
  readonly index: channel.Channel;
  readonly data: channel.Channel[];
  /** The number of reads the Core has served, reduced or not. */
  reads = 0;
  readonly channels = {
    retrieve: async (key: channel.Key | channel.Name): Promise<channel.Channel> => {
      const ch = [this.index, ...this.data].find(
        (c) => c.key === key || c.name === key,
      );
      if (ch == null) throw new NotFoundError(`channel ${key} not found`);
      return ch;
    },
  };

  private readonly start: bigint;
  private readonly period: bigint;
  private clock: TimeStamp;
  private readonly streamers = new Set<SyntheticStreamer>();

  constructor({ rate, start, now, channels = 1 }: SyntheticCoreProps) {
    this.start = start.valueOf();
    this.period = new Rate(rate).period.valueOf();
    this.clock = now;
    this.index = new channel.Channel({
      key: INDEX_KEY,
      name: "time",
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    });
    this.data = Array.from(
      { length: channels },
      (_, i) =>
        new channel.Channel({
          key: INDEX_KEY + 1 + i,
          name: `data_${i}`,
          dataType: DataType.FLOAT32,
          index: INDEX_KEY,
        }),
    );
    this.checkDomain();
  }

  /** @returns the time on the Core's clock. */
  now = (): TimeStamp => this.clock;

  /** Opens a feed that reads and streams from the Core. */
  openFeed(options: FeedOptions = {}): Feed {
    return new Feed({
      now: this.now,
      ...options,
      readRemote: async (tr, keys, reduction) => this.read(tr, keys, reduction),
      openStreamer: async (config) => this.openStreamer(config),
      readLatest: async (keys) => this.readLatest(keys),
    });
  }

  /**
   * Advances the clock by the span, and streams the samples written within it to every
   * open streamer.
   */
  advance(span: TimeSpan): void {
    const from = this.written();
    this.clock = this.clock.add(span);
    this.checkDomain();
    const to = this.written();
    if (to === from) return;
    const keys = [this.index.key, ...this.data.map((c) => c.key)];
    const frame = new Frame(
      keys,
      keys.map((key) => this.raw(key, from, to, LEADING_DOMAIN)),
    );
    this.streamers.forEach((s) => s.push(frame));
  }

  /** Reads the channels over the time range, as the Core's read with a reduction. */
  read(tr: TimeRange, keys: channel.Key[], reduction?: Reduction): Frame {
    this.reads++;
    const first = this.samplesBefore(tr.start);
    const last = this.samplesBefore(tr.end);
    if (last <= first) return new Frame();
    if (reduction?.variant === "stride")
      throw new Error("synthetic core does not serve stride reductions");
    const size =
      reduction == null
        ? 1
        : groupSize(last - first, reduction.pointLimit, reduction.aggregation);
    const frame = new Frame();
    for (const key of keys) {
      if (size < 2) {
        for (let c = first; c < last; c += CHUNK)
          frame.push(key, this.raw(key, c, Math.min(c + CHUNK, last)));
        continue;
      }
      frame.push(
        key,
        this.reduced(key, first, last, size, reduction?.aggregation ?? "min_max"),
      );
    }
    return frame;
  }

  private readLatest(keys: channel.Key[]): Frame {
    const n = this.written();
    const frame = new Frame();
    keys.forEach((key) => {
      if (n > 0) frame.push(key, this.raw(key, n - 1, n));
    });
    return frame;
  }

  private openStreamer(config: StreamerConfig): Streamer {
    const streamer = new SyntheticStreamer(config, (s) => this.streamers.delete(s));
    this.streamers.add(streamer);
    return streamer;
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
    const elapsed = this.clock.valueOf() - this.start;
    if (elapsed <= 0n) return 0;
    return Number((elapsed + this.period - 1n) / this.period);
  }

  private checkDomain(): void {
    if (this.written() >= Number(DOMAIN))
      throw new Error("synthetic core holds at most 2^32 samples");
  }

  private value(key: channel.Key, i: number): number {
    return Math.sin(i / 1000 + key);
  }

  private range(from: number, to: number): TimeRange {
    return new TimeRange(this.time(from), this.time(to - 1) + 1n);
  }

  private raw(key: channel.Key, from: number, to: number, domain = DOMAIN): Series {
    const n = to - from;
    let data: BigInt64Array<ArrayBuffer> | Float32Array<ArrayBuffer>;
    if (key === this.index.key) {
      const times = new BigInt64Array(n);
      for (let i = 0; i < n; i++) times[i] = this.time(from + i);
      data = times;
    } else {
      const values = new Float32Array(n);
      for (let i = 0; i < n; i++) values[i] = this.value(key, from + i);
      data = values;
    }
    return new Series({
      data,
      dataType: key === this.index.key ? DataType.TIMESTAMP : DataType.FLOAT32,
      timeRange: this.range(from, to),
      alignment: domain + BigInt(from),
    });
  }

  // reduced returns the samples in [first, last) in groups of size samples anchored at
  // sample zero. A group cut by the read reduces the samples inside it. The index
  // reduces exactly: min_max keeps each group's first and last times, decimate its first,
  // and average its midpoint.
  private reduced(
    key: channel.Key,
    first: number,
    last: number,
    size: number,
    aggregation: Aggregation,
  ): Series {
    const minMax = aggregation === "min_max";
    const perGroup = minMax ? 2 : 1;
    const head = Math.floor(first / size);
    const groups = Math.floor((last - 1) / size) - head + 1;
    const isIndex = key === this.index.key;
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
      timeRange: this.range(first, last),
      alignment: DOMAIN + BigInt(head * size),
      alignmentMultiple: BigInt(minMax ? size / 2 : size),
    });
  }
}

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

class SyntheticStreamer implements Streamer {
  keys: channel.Key[];
  private readonly queue: Frame[] = [];
  // waiting resolves the pending pull with the next frame, or with null on close.
  private waiting: ((frame: Frame | null) => void) | null = null;
  private closed = false;

  constructor(
    config: StreamerConfig,
    private readonly onClose: (s: SyntheticStreamer) => void,
  ) {
    this.keys = toKeys(streamerConfigZ.parse(config).channels);
  }

  push(frame: Frame): void {
    if (this.closed) return;
    const filtered = frame.get(this.keys);
    if (filtered.series.length === 0) return;
    if (this.waiting == null) return void this.queue.push(filtered);
    this.waiting(filtered);
    this.waiting = null;
  }

  async update(channels: channel.Params): Promise<void> {
    this.keys = toKeys(channels);
  }

  async read(): Promise<Frame> {
    const frame = await this.pull();
    if (frame == null) throw new EOF();
    return frame;
  }

  async next(): Promise<IteratorResult<Frame>> {
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

  [Symbol.asyncIterator](): AsyncIterator<Frame> {
    return this;
  }

  private async pull(): Promise<Frame | null> {
    if (this.closed) return null;
    const next = this.queue.shift();
    if (next != null) return next;
    return await new Promise((resolve) => (this.waiting = resolve));
  }
}

const toKeys = (channels: channel.Params): channel.Key[] =>
  array.toArray(channels).map((c) => {
    if (typeof c === "number") return c;
    throw new Error("synthetic core streams channels by key only");
  });
