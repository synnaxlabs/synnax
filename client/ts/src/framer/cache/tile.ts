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
  type MultiSeries,
  scheduler,
  sync,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { UnexpectedError } from "@/errors";
import { type Block } from "@/framer/cache/block";
import { type Bounds, tileRange } from "@/framer/cache/grid";
import { DEFAULT_BATCH_DEBOUNCE, type RemoteReader } from "@/framer/cache/reader";
import { type Store } from "@/framer/cache/store";

/** Identifies one tile of one channel, reduced under a point limit. */
export interface TileSpec extends Bounds {
  key: channel.Key;
  pointLimit: number;
  aggregation: Aggregation;
}

export interface TileReaderProps {
  /** Reads reduced tiles from the Core. */
  readRemote: RemoteReader;
  /** Caches finished tiles. */
  store: Store;
  /**
   * Window in which tile reads with equal bounds and options merge into one request.
   * @default TimeSpan.milliseconds(50)
   */
  batchDebounce?: TimeSpan;
  /**
   * Deadline for a single remote fetch.
   * @default TimeSpan.seconds(30)
   */
  fetchTimeout?: TimeSpan;
  /**
   * The current time, which stamps each tile when it is fetched.
   * @default TimeStamp.now
   */
  now?: () => TimeStamp;
}

/** A cached tile: one channel's reduced samples. */
class Tile implements Block {
  readonly bytes: number;

  constructor(
    readonly channel: channel.Key,
    readonly series: MultiSeries,
    readonly fetchedAt: TimeStamp,
  ) {
    this.bytes = series.series.reduce((sum, s) => sum + s.byteCapacity.valueOf(), 0);
  }

  get held(): boolean {
    return this.series.series.some((s) => s.refCount > 0);
  }
}

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

/** Reads reduced tiles of channel data, caching each finished tile in the store. */
export class TileReader {
  private readonly props: Required<Omit<TileReaderProps, "batchDebounce">>;
  private readonly batcher: scheduler.Batcher<TileSpec, MultiSeries>;
  // pending holds each tile's fetching reads by channel, keyed by tile request key.
  private readonly pending = new Map<string, Map<channel.Key, Promise<MultiSeries>>>();
  private closed = false;

  constructor(props: TileReaderProps) {
    const {
      readRemote,
      store,
      batchDebounce = DEFAULT_BATCH_DEBOUNCE,
      fetchTimeout = TimeSpan.seconds(30),
      now = () => TimeStamp.now(),
    } = props;
    this.props = { readRemote, store, fetchTimeout, now };
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
    const { store } = this.props;
    const key = cacheKey(spec);
    const tile = tileRequestKey(spec);
    const cached = store.get(key);
    if (cached != null && !(cached instanceof Tile))
      throw new UnexpectedError(`block ${key} is not a tile`);
    if (cached != null)
      if (
        store.expired(cached) ||
        (dataType != null &&
          !cached.series.series.every((s) => s.dataType.equals(dataType)))
      )
        store.delete(key);
      else {
        // Reads made later in this task enter pending first.
        await Promise.resolve();
        if (this.closed) throw new UnexpectedError("tile reader is closed");
        // Samples written into the tile since it was cached can change its group size.
        if (!this.pending.has(tile)) return cached.series;
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
      if (!this.closed) store.set(key, new Tile(spec.key, series, this.props.now()));
      return series;
    } finally {
      reads.delete(spec.key);
      if (reads.size === 0) this.pending.delete(tile);
    }
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
    const { readRemote, fetchTimeout } = this.props;
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
          const series = frame.get(key);
          if (series.series.length > 0)
            this.props.store.retainDataType(key, series.series[0].dataType);
          return [key, series];
        }),
      );
      group.forEach(({ req, resolve }) => resolve(results.get(req.key)!));
    } catch (err) {
      group.forEach(({ reject }) => reject(err));
    }
  }

  /**
   * Closes the reader, rejecting every pending read with an UnexpectedError. Fetch
   * results that arrive later are discarded.
   */
  close(): void {
    this.closed = true;
    this.batcher.close(new UnexpectedError("tile reader is closed"));
  }
}
