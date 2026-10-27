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
  type DataType,
  MultiSeries,
  scheduler,
  Size,
  sync,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { UnexpectedError } from "@/errors";
import { DEFAULT_BATCH_DEBOUNCE, type RemoteReader } from "@/framer/cache/reader";
import { DEFAULT_STATIC_PROPS } from "@/framer/cache/static";
import { IDENTITY_TRANSFORM, type Transform } from "@/framer/cache/transform";

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
  readRemote: RemoteReader;
  /** Applied to every series before it enters the cache. Defaults to identity. */
  transform?: Transform;
  /**
   * Window in which tile reads with equal bounds and options merge into one request.
   * @default TimeSpan.milliseconds(50)
   */
  batchDebounce?: TimeSpan;
  /**
   * Byte size the cache evicts down to, least recently used first. Each tile counts its
   * samples and a small fixed overhead. A tile whose series a caller holds is never
   * evicted, so the cache can exceed the budget.
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

interface CacheEntry {
  channel: channel.Key;
  series: MultiSeries;
  bytes: number;
  fetchedAt: TimeStamp;
}

const DEFAULT_BUDGET = Size.megabytes(128);

// ENTRY_OVERHEAD estimates the memory of a cache entry apart from its samples, so that
// tiles with no samples count against the budget too.
const ENTRY_OVERHEAD = 256;

const cacheKey = ({ key, level, index, pointLimit, aggregation }: TileSpec): string =>
  `${key}/${level}/${index}/${pointLimit}/${aggregation}`;

/**
 * @returns a key for the time range and reduction of the tile's read. Specs with equal
 * keys differ at most in channel, so their reads share one request.
 */
export const tileRequestKey = (spec: TileSpec): string => {
  const { level, index, pointLimit, aggregation, start, end } = spec;
  return [level, index, pointLimit, aggregation, start?.valueOf(), end?.valueOf()].join(
    "/",
  );
};

const held = (series: MultiSeries): boolean =>
  series.series.some((s) => s.refCount > 0);

/**
 * Reads reduced tiles of channel data, caching each finished tile under a byte budget.
 */
export class TileReader {
  private readonly props: Required<Omit<TileReaderProps, "batchDebounce">>;
  private readonly batcher: scheduler.Batcher<TileSpec, MultiSeries>;
  // Map order is recency order: a hit moves its entry to the end.
  private readonly tiles = new Map<string, CacheEntry>();
  // dataTypes holds the data type of each channel's newest fetched tile, which every
  // cached tile of the channel shares.
  private readonly dataTypes = new Map<channel.Key, DataType>();
  // pending holds each tile's fetching reads by channel, keyed by tile request key.
  private readonly pending = new Map<string, Map<channel.Key, Promise<MultiSeries>>>();
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
    this.batcher = new scheduler.Batcher({
      interval: batchDebounce,
      exec: async (entries) => await this.batchRead(entries),
    });
  }

  /**
   * Reads the tile, serving it from the cache when a fresh copy is there. When another
   * channel of the tile is being fetched, the read fetches the tile again with it, so
   * channels drawn as a pair share one group size.
   * @param dataType - The data type of the series the caller holds. A cached tile of
   * another data type is read again.
   * @throws {UnexpectedError} if the reader is closed while the read is pending.
   * @throws {Unreachable} if the fetch serving the read exceeds the fetch timeout.
   */
  async read(spec: TileSpec, dataType?: DataType): Promise<MultiSeries> {
    if (this.closed) throw new UnexpectedError("tile reader is closed");
    if (spec.start != null || spec.end != null) return await this.batcher.enqueue(spec);
    const key = cacheKey(spec);
    const tile = tileRequestKey(spec);
    const cached = this.tiles.get(key);
    if (cached != null) {
      this.remove(key, cached);
      const fresh = TimeStamp.since(cached.fetchedAt).lessThan(
        this.props.staleThreshold,
      );
      if (
        fresh &&
        (dataType == null ||
          cached.series.series.every((s) => s.dataType.equals(dataType)))
      ) {
        this.tiles.set(key, cached);
        this.bytes += cached.bytes;
        // Reads made later in this task enter pending first.
        await Promise.resolve();
        if (this.closed) throw new UnexpectedError("tile reader is closed");
        // Samples written into the tile since it was cached can change its group size.
        if (!this.pending.has(tile)) return cached.series;
      }
    }
    let reads = this.pending.get(tile);
    const pending = reads?.get(spec.key);
    if (pending != null) return await pending;
    if (reads == null) {
      reads = new Map();
      this.pending.set(tile, reads);
    }
    const fetched = this.batcher.enqueue(spec);
    reads.set(spec.key, fetched);
    try {
      const series = await fetched;
      if (!this.closed) this.insert(key, spec.key, series);
      return series;
    } finally {
      reads.delete(spec.key);
      if (reads.size === 0) this.pending.delete(tile);
    }
  }

  private insert(key: string, ch: channel.Key, series: MultiSeries): void {
    const replaced = this.tiles.get(key);
    if (replaced != null) this.remove(key, replaced);
    const bytes = series.series.reduce(
      (sum, s) => sum + s.byteLength.valueOf(),
      ENTRY_OVERHEAD,
    );
    this.tiles.set(key, { channel: ch, series, bytes, fetchedAt: TimeStamp.now() });
    this.bytes += bytes;
    const budget = this.props.budget.valueOf();
    for (const [k, t] of this.tiles) {
      if (this.bytes <= budget) break;
      if (!held(t.series)) this.remove(k, t);
    }
  }

  private remove(key: string, tile: CacheEntry): void {
    this.tiles.delete(key);
    this.bytes -= tile.bytes;
  }

  // A channel's data type can change, and tiles of two data types cannot be drawn
  // together, so a fetched tile of a new data type drops the channel's cached tiles.
  private retainDataType(ch: channel.Key, dataType: DataType): void {
    const held = this.dataTypes.get(ch);
    this.dataTypes.set(ch, dataType);
    if (held == null || held.equals(dataType)) return;
    for (const [key, tile] of this.tiles)
      if (tile.channel === ch) this.remove(key, tile);
  }

  private async batchRead(
    entries: Array<scheduler.BatcherEntry<TileSpec, MultiSeries>>,
  ): Promise<void> {
    const groups = new Map<
      string,
      Array<scheduler.BatcherEntry<TileSpec, MultiSeries>>
    >();
    entries.forEach((entry) => {
      const key = tileRequestKey(entry.req);
      const group = groups.get(key);
      if (group == null) groups.set(key, [entry]);
      else group.push(entry);
    });
    await Promise.all([...groups.values()].map(async (g) => await this.fetch(g)));
  }

  // Never throws: every outcome settles the group's entries.
  private async fetch(
    group: Array<scheduler.BatcherEntry<TileSpec, MultiSeries>>,
  ): Promise<void> {
    const { readRemote, transform, fetchTimeout } = this.props;
    const spec = group[0].req;
    const tr = tileRange(spec);
    const keys = [...new Set(group.map(({ req }) => req.key))];
    try {
      const frame = await sync.withTimeout(
        readRemote(tr, keys, {
          variant: "limit",
          aggregation: spec.aggregation,
          pointLimit: spec.pointLimit,
        }),
        fetchTimeout,
        () =>
          new Unreachable({
            message: `tile read for ${tr.toString()} timed out after ${fetchTimeout.toString()}`,
          }),
      );
      if (this.closed) return;
      const results = new Map(
        keys.map((key) => {
          const series = frame.get(key).series.map((s) => transform.convert(s));
          if (series.length > 0) this.retainDataType(key, series[0].dataType);
          return [key, new MultiSeries(series)];
        }),
      );
      group.forEach(({ req, resolve }) => resolve(results.get(req.key)!));
    } catch (err) {
      group.forEach(({ reject }) => reject(err));
    }
  }

  /**
   * Closes the reader, rejecting every pending read with an UnexpectedError and
   * dropping every cached tile. Fetch results that arrive later are discarded.
   */
  close(): void {
    this.closed = true;
    this.batcher.close(new UnexpectedError("tile reader is closed"));
    this.tiles.clear();
    this.dataTypes.clear();
    this.bytes = 0;
  }
}
