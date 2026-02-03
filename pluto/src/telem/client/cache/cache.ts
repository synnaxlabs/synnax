// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, TimeSpan, UnexpectedError } from "@synnaxlabs/client";

import { type DynamicProps } from "@/telem/client/cache/dynamic";
import {
  DEFAULT_STATIC_PROPS,
  type StaticProps,
  zeroCacheGCMetrics,
} from "@/telem/client/cache/static";
import { Unary } from "@/telem/client/cache/unary";

export const CACHE_BUFFER_SIZE: TimeSpan = TimeSpan.seconds(60);

/** Props for instantiating an @link Cache */
export interface CacheProps
  extends StaticProps, Partial<Pick<DynamicProps, "dynamicBufferSize">> {
  /** Used to populate new cache entries with relevant info about the channel */
  channelRetriever: channel.Retriever;
  /**
   * Sets the interval at which the cache will garbage collect, removing data that
   * currently in use by the rest of hte program.
   * @default TimeSpan.seconds(30)
   */
  gcInterval?: TimeSpan;
}

/**
 * Maintains an in-memory cache of channel data.
 */
export class Cache {
  private readonly props: Required<CacheProps>;
  private readonly cache = new Map<channel.Key, Unary>();
  private readonly gcInterval: ReturnType<typeof setInterval>;

  constructor(props: CacheProps) {
    this.props = {
      dynamicBufferSize: CACHE_BUFFER_SIZE,
      gcInterval: TimeSpan.seconds(30),
      ...DEFAULT_STATIC_PROPS,
      ...props,
    };
    this.gcInterval = setInterval(() => this.gc(), this.props.gcInterval.milliseconds);
  }

  /**
   * Populates cache entries for the given keys, if they are not already present.
   *
   * @param keys - The keys to populate the cache with.
   */
  async populateMissing(keys: channel.Keys): Promise<void> {
    const { instrumentation: ins, channelRetriever, dynamicBufferSize } = this.props;
    const toFetch: channel.Keys = [];
    for (const key of keys) if (!this.cache.has(key)) toFetch.push(key);
    if (toFetch.length === 0) return;
    const channels = await channelRetriever.retrieve(toFetch);
    channels.forEach((channel) => {
      if (this.cache.has(channel.key)) return;
      const unary = new Unary({
        channel,
        dynamicBufferSize,
        instrumentation: ins.child(`cache-${channel.name}-${channel.key}`),
      });
      this.cache.set(channel.key, unary);
    });
  }

  /**
   * Returns the cache entry for the given channel.
   *
   * @param key - The key of the channel to retrieve.
   * @returns The cache entry for the given channel.
   */
  get(key: channel.Key): Unary {
    const c = this.cache.get(key);
    if (c != null) return c;
    throw new UnexpectedError(`cache entry for ${key} not found`);
  }

  /** Garbage collects the cache */
  private gc(): void {
    const {
      instrumentation: { L },
    } = this.props;
    L.info("starting garbage collection");
    const totalGCMetrics = zeroCacheGCMetrics();
    this.cache.forEach((c) => {
      const res = c.gc();
      totalGCMetrics.purgedSeries += res.purgedSeries;
      totalGCMetrics.purgedBytes = totalGCMetrics.purgedBytes.add(res.purgedBytes);
    });
    L.info(
      "garbage collection complete",
      {
        purgedSeries: totalGCMetrics.purgedSeries,
        purgedBytes: totalGCMetrics.purgedBytes.toString(),
      },
      true,
    );
  }

  /**
   * Closes the cache, clearing all entries and stopping garbage collection. After this
   * method is called, the cache and it's entries should not be used.
   */
  close(): void {
    clearInterval(this.gcInterval);
    this.cache.forEach((c) => c.close());
    this.cache.clear();
  }
}
