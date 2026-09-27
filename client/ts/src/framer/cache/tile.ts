// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Unreachable } from "@synnaxlabs/freighter";
import {
  type Aggregation,
  debounce,
  errors,
  MultiSeries,
  type Reduction,
  type Series,
  Size,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { UnexpectedError } from "@/errors";
import { DEFAULT_BATCH_DEBOUNCE } from "@/framer/cache/reader";
import { DEFAULT_STATIC_PROPS } from "@/framer/cache/static";
import { IDENTITY_TRANSFORM, type Transform } from "@/framer/cache/transform";
import { type Frame } from "@/framer/frame";

/**
 * Identifies one tile: the samples of one channel inside a fixed span of time, reduced
 * under a point limit. Tiles at level L span 2^L milliseconds, and tile i of a level
 * starts i spans after the Unix epoch.
 */
export interface TileSpec {
  key: channel.Key;
  level: number;
  index: number;
  pointLimit: number;
  aggregation: Aggregation;
  /** Starts the read after the tile does. A read with a start is never cached. */
  start?: TimeStamp;
  /**
   * Ends the read before the tile does, for a tile that is still being written. A read
   * with an end is never cached.
   */
  end?: TimeStamp;
}

/** Reads the given channels over a time range with the given reduction. */
export interface TileRemoteReader {
  (tr: TimeRange, keys: channel.Key[], reduction: Reduction): Promise<Frame>;
}

/** @returns the span of every tile at the given level. */
export const tileSpan = (level: number): TimeSpan =>
  new TimeSpan(TimeSpan.MILLISECOND.valueOf() << BigInt(level));

/** @returns the time range the tile covers, cut to the spec's start and end. */
export const tileRange = ({ level, index, start, end }: TileSpec): TimeRange => {
  const span = tileSpan(level).valueOf();
  let lower = BigInt(index) * span;
  let upper = lower + span;
  if (start != null && start.valueOf() > lower) lower = start.valueOf();
  if (end != null && end.valueOf() < upper) upper = end.valueOf();
  return new TimeRange(lower, upper);
};

export interface TileReaderProps {
  /** Reads reduced tiles from the Core. */
  readRemote: TileRemoteReader;
  /** Applied to every series before it enters the cache. Defaults to identity. */
  transform?: Transform;
  /**
   * Window in which tile reads with equal bounds and options merge into one request.
   * @default TimeSpan.milliseconds(50)
   */
  batchDebounce?: TimeSpan;
  /**
   * Byte size the cache evicts down to, least recently used first. A tile whose series
   * a caller holds is never evicted, so the cache can exceed the budget.
   * @default Size.megabytes(128)
   */
  budget?: Size;
  /**
   * How long a cached tile answers reads before it is fetched again, so that writes
   * into the past show up.
   * @default TimeSpan.minutes(10)
   */
  staleThreshold?: TimeSpan;
  /**
   * Deadline for a single remote fetch.
   * @default TimeSpan.seconds(30)
   */
  fetchTimeout?: TimeSpan;
}

interface Entry {
  series: MultiSeries;
  bytes: number;
  fetchedAt: TimeStamp;
}

const DEFAULT_BUDGET = Size.megabytes(128);

const cacheKey = ({ key, level, index, pointLimit, aggregation }: TileSpec): string =>
  `${key}/${level}/${index}/${pointLimit}/${aggregation}`;

// Reads that share bounds and options share one request.
const batchKey = (spec: TileSpec): string => {
  const { level, index, pointLimit, aggregation, start, end } = spec;
  return [level, index, pointLimit, aggregation, start?.valueOf(), end?.valueOf()].join(
    "/",
  );
};

const held = (series: MultiSeries): boolean =>
  series.series.some((s) => s.refCount > 0);

/**
 * Reads reduced tiles of channel data, caching each finished tile under a byte budget.
 * Tiles never merge with each other or with the alignment-keyed cache.
 */
export class TileReader {
  private readonly props: Required<Omit<TileReaderProps, "batchDebounce">>;
  private readonly batcher: debounce.Batcher<TileSpec, MultiSeries>;
  // Map order is recency order: a hit moves its entry to the end.
  private readonly entries = new Map<string, Entry>();
  private readonly pending = new Map<string, Promise<MultiSeries>>();
  private readonly inFlight = new Set<Array<debounce.Entry<TileSpec, MultiSeries>>>();
  private bytes = 0;
  private closed = false;

  constructor(props: TileReaderProps) {
    const {
      readRemote,
      transform = IDENTITY_TRANSFORM,
      batchDebounce = DEFAULT_BATCH_DEBOUNCE,
      budget = DEFAULT_BUDGET,
      staleThreshold = DEFAULT_STATIC_PROPS.staleCoverageThreshold,
      fetchTimeout = TimeSpan.seconds(30),
    } = props;
    this.props = { readRemote, transform, budget, staleThreshold, fetchTimeout };
    this.batcher = new debounce.Batcher({
      interval: batchDebounce,
      exec: async (entries) => await this.batchRead(entries),
    });
  }

  /** @returns the total byte size of the cached tiles. */
  get size(): Size {
    return Size.bytes(this.bytes);
  }

  /**
   * Reads the tile, serving it from the cache when a fresh copy is there.
   * @throws {UnexpectedError} if the reader is closed while the read is pending.
   * @throws {Unreachable} if the fetch serving the read exceeds the fetch timeout.
   */
  async read(spec: TileSpec): Promise<MultiSeries> {
    if (this.closed) throw new UnexpectedError("tile reader is closed");
    if (spec.start != null || spec.end != null) return await this.batcher.enqueue(spec);
    const key = cacheKey(spec);
    const entry = this.entries.get(key);
    if (entry != null) {
      this.remove(key, entry);
      if (TimeStamp.since(entry.fetchedAt).lessThan(this.props.staleThreshold)) {
        this.entries.set(key, entry);
        this.bytes += entry.bytes;
        return entry.series;
      }
    }
    const pending = this.pending.get(key);
    if (pending != null) return await pending;
    const fetched = this.batcher.enqueue(spec);
    this.pending.set(key, fetched);
    try {
      const series = await fetched;
      if (!this.closed) this.insert(key, series);
      return series;
    } finally {
      this.pending.delete(key);
    }
  }

  private insert(key: string, series: MultiSeries): void {
    const bytes = series.series.reduce((sum, s) => sum + s.byteLength.valueOf(), 0);
    this.entries.set(key, { series, bytes, fetchedAt: TimeStamp.now() });
    this.bytes += bytes;
    const budget = this.props.budget.valueOf();
    for (const [k, e] of this.entries) {
      if (this.bytes <= budget) break;
      if (!held(e.series)) this.remove(k, e);
    }
  }

  private remove(key: string, entry: Entry): void {
    this.entries.delete(key);
    this.bytes -= entry.bytes;
  }

  private async batchRead(
    entries: Array<debounce.Entry<TileSpec, MultiSeries>>,
  ): Promise<void> {
    const groups = new Map<string, Array<debounce.Entry<TileSpec, MultiSeries>>>();
    entries.forEach((entry) => {
      const key = batchKey(entry.req);
      const group = groups.get(key);
      if (group == null) groups.set(key, [entry]);
      else group.push(entry);
    });
    await Promise.all([...groups.values()].map(async (g) => await this.fetch(g)));
  }

  // Never throws: every outcome settles the group's entries.
  private async fetch(
    group: Array<debounce.Entry<TileSpec, MultiSeries>>,
  ): Promise<void> {
    const { transform } = this.props;
    const spec = group[0].req;
    const keys = [...new Set(group.map(({ req }) => req.key))];
    this.inFlight.add(group);
    try {
      const frame = await this.fetchWithDeadline(tileRange(spec), keys, {
        variant: "limit",
        aggregation: spec.aggregation,
        pointLimit: spec.pointLimit,
      });
      if (!this.inFlight.has(group)) return;
      const grouped = new Map<channel.Key, Series[]>();
      frame.forEach((k, s) => {
        const key = k as channel.Key;
        const converted = transform.convert(s);
        const existing = grouped.get(key);
        if (existing == null) grouped.set(key, [converted]);
        else existing.push(converted);
      });
      const results = new Map(
        keys.map((key) => [key, new MultiSeries(grouped.get(key) ?? [])]),
      );
      group.forEach(({ req, resolve }) => resolve(results.get(req.key)!));
    } catch (err) {
      group.forEach(({ reject }) => reject(err));
    } finally {
      this.inFlight.delete(group);
    }
  }

  private async fetchWithDeadline(
    tr: TimeRange,
    keys: channel.Key[],
    reduction: Reduction,
  ): Promise<Frame> {
    const { readRemote, fetchTimeout } = this.props;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const deadline = new Promise<never>((_, reject) => {
      timer = setTimeout(() => {
        const message = `tile read for ${tr.toString()} timed out after ${fetchTimeout.toString()}`;
        reject(new Unreachable({ message }));
      }, fetchTimeout.milliseconds);
    });
    const fetched = readRemote(tr, keys, reduction);
    try {
      return await Promise.race([fetched, deadline]);
    } catch (err) {
      // The read already failed for its callers; a late settle of the losing fetch must
      // not surface as an unhandled rejection.
      fetched.catch(() => {});
      throw errors.fromUnknown(err);
    } finally {
      clearTimeout(timer);
    }
  }

  /**
   * Closes the reader, rejecting every pending read with an UnexpectedError and
   * dropping every cached tile. Fetch results that arrive later are discarded.
   */
  close(): void {
    this.closed = true;
    const err = new UnexpectedError("tile reader is closed");
    this.batcher.close(err);
    this.inFlight.forEach((group) => group.forEach(({ reject }) => reject(err)));
    this.inFlight.clear();
    this.entries.clear();
    this.bytes = 0;
  }
}
