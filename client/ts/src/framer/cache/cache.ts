// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type bounds, TimeSpan, type TimeStamp } from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { UnexpectedError } from "@/errors";
import { type ChannelRetriever } from "@/framer/adapter";
import { type DynamicProps } from "@/framer/cache/dynamic";
import { Stamps } from "@/framer/cache/stamps";
import {
  DEFAULT_STATIC_PROPS,
  type StaticProps,
  zeroGCMetrics,
} from "@/framer/cache/static";
import { Unary } from "@/framer/cache/unary";

const DEFAULT_DYNAMIC_BUFFER_SIZE: TimeSpan = TimeSpan.seconds(60);
const DEFAULT_WRITE_STAMPS = 1024;

/** Props for instantiating a {@link Cache} */
export interface CacheProps
  extends StaticProps, Partial<Pick<DynamicProps, "dynamicBufferSize">> {
  /**
   * Sets the interval at which the cache will garbage collect, removing stale data
   * that is not in use by the rest of the program.
   * @default TimeSpan.seconds(30)
   */
  gcInterval?: TimeSpan;
  /**
   * Fetches channel payloads, so entries learn their index and timeAt can pair samples
   * with stamps. Defaults to a retriever that finds no channel.
   */
  retrieveChannels?: ChannelRetriever;
  /**
   * How many write stamps a channel with no index keeps, so timeAt can stamp its
   * samples from the writes that carried them.
   * @default 1024
   */
  writeStamps?: number;
}

/**
 * Maintains an in-memory cache of channel data. Entries are created on demand and
 * buffer types derive from the data written to them. An entry learns its index channel
 * through resolve.
 */
export class Cache {
  private readonly props: Required<CacheProps>;
  private readonly cache = new Map<channel.Key, Unary>();
  // Resolved keys. Null marks a channel with no index.
  private readonly indexes = new Map<channel.Key, channel.Key | null>();
  private readonly gcInterval: ReturnType<typeof setInterval>;
  private closed = false;

  constructor(props: CacheProps = {}) {
    const {
      dynamicBufferSize = DEFAULT_DYNAMIC_BUFFER_SIZE,
      gcInterval = TimeSpan.seconds(30),
      instrumentation = DEFAULT_STATIC_PROPS.instrumentation,
      transform = DEFAULT_STATIC_PROPS.transform,
      staleEntryThreshold = DEFAULT_STATIC_PROPS.staleEntryThreshold,
      staleCoverageThreshold = DEFAULT_STATIC_PROPS.staleCoverageThreshold,
      retrieveChannels = async () => [],
      writeStamps = DEFAULT_WRITE_STAMPS,
    } = props;
    this.props = {
      dynamicBufferSize,
      gcInterval,
      instrumentation,
      transform,
      staleEntryThreshold,
      staleCoverageThreshold,
      retrieveChannels,
      writeStamps,
    };
    this.gcInterval = setInterval(() => this.gc(), gcInterval.milliseconds);
  }

  /**
   * Resolves the index of each key not yet resolved, so timeAt can pair its samples
   * with their stamps. A key the retriever omits resolves to no index. An index
   * resolves with the first channel that names it.
   * @throws {UnexpectedError} if the cache has been closed.
   */
  async resolve(keys: channel.Key[]): Promise<void> {
    if (this.closed)
      throw new UnexpectedError("resolve called on a closed telemetry cache");
    const pending = keys.filter((k) => !this.indexes.has(k));
    if (pending.length === 0) return;
    const channels = await this.props.retrieveChannels(pending);
    if (this.closed) return;
    const found = new Map(channels.map((ch) => [ch.key, ch]));
    pending.forEach((key) => {
      // Recorded since the lookup began, as an earlier key's index or by another call.
      if (this.indexes.has(key)) return;
      const ch = found.get(key);
      const index = ch == null || ch.index === 0 ? null : ch.index;
      this.indexes.set(key, index);
      if (index == null) {
        this.get(key).stamps = new Stamps(this.props.writeStamps);
        return;
      }
      const entry = this.get(index);
      entry.index = entry;
      this.indexes.set(index, index);
      this.get(key).index = entry;
    });
  }

  /** @returns the key of the channel's index, or null when unresolved or none. */
  indexOf(key: channel.Key): channel.Key | null {
    return this.indexes.get(key) ?? null;
  }

  /** @returns the stamp of the channel's sample at alignment, else null. */
  timeAt(key: channel.Key, alignment: bigint): TimeStamp | null {
    return this.get(key).timeAt(alignment);
  }

  /** @returns the stamp of the channel's last live write, else null. */
  lastWrite(key: channel.Key): TimeStamp | null {
    return this.get(key).lastWrite;
  }

  /**
   * Returns the cache entry for the given channel, creating an empty one if it does
   * not exist.
   * @throws {UnexpectedError} if the cache has been closed.
   */
  get(key: channel.Key): Unary {
    if (this.closed)
      throw new UnexpectedError(`get(${key}) called on a closed telemetry cache`);
    const existing = this.cache.get(key);
    if (existing != null) return existing;
    const {
      dynamicBufferSize,
      transform,
      staleEntryThreshold,
      staleCoverageThreshold,
    } = this.props;
    const unary = new Unary({
      dynamicBufferSize,
      transform,
      staleEntryThreshold,
      staleCoverageThreshold,
      instrumentation: this.props.instrumentation.child(`cache-${key}`),
    });
    this.cache.set(key, unary);
    return unary;
  }

  private gc(): void {
    const {
      instrumentation: { L },
    } = this.props;
    L.info("starting garbage collection");
    // Stamps stay while a consumer holds the samples they belong to.
    const held = new Map<channel.Key, bounds.Bounds<bigint>[]>();
    this.cache.forEach((c, key) => {
      const index = this.indexes.get(key);
      if (index == null) return;
      const spans = c.heldAlignments();
      if (spans.length === 0) return;
      const prev = held.get(index);
      if (prev == null) held.set(index, spans);
      else prev.push(...spans);
    });
    const total = zeroGCMetrics();
    this.cache.forEach((c, key) => {
      const res = c.gc(held.get(key) ?? []);
      total.purgedSeries += res.purgedSeries;
      total.purgedBytes = total.purgedBytes.add(res.purgedBytes);
    });
    L.info(
      "garbage collection complete",
      {
        purgedSeries: total.purgedSeries,
        purgedBytes: total.purgedBytes.toString(),
      },
      true,
    );
  }

  /**
   * Closes the cache, clearing all entries and stopping garbage collection. All
   * methods throw after close.
   */
  close(): void {
    this.closed = true;
    clearInterval(this.gcInterval);
    this.cache.forEach((c) => c.close());
    this.cache.clear();
  }
}
