// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { UnexpectedError } from "@/errors";
import { type DynamicProps } from "@/telem/cache/dynamic";
import {
  DEFAULT_STATIC_PROPS,
  type StaticProps,
  zeroGCMetrics,
} from "@/telem/cache/static";
import { Unary } from "@/telem/cache/unary";

export const CACHE_BUFFER_SIZE: TimeSpan = TimeSpan.seconds(60);

/** Props for instantiating a {@link Cache} */
export interface CacheProps
  extends StaticProps, Partial<Pick<DynamicProps, "dynamicBufferSize">> {
  /**
   * Sets the interval at which the cache will garbage collect, removing stale data
   * that is not in use by the rest of the program.
   * @default TimeSpan.seconds(30)
   */
  gcInterval?: TimeSpan;
}

/**
 * Maintains an in-memory cache of channel data. Entries are created on demand and
 * carry no channel metadata; buffer types derive from the data written to them.
 */
export class Cache {
  private readonly props: Required<CacheProps>;
  private readonly cache = new Map<channel.Key, Unary>();
  private readonly gcInterval: ReturnType<typeof setInterval>;
  private closed = false;

  constructor(props: CacheProps = {}) {
    const {
      dynamicBufferSize = CACHE_BUFFER_SIZE,
      gcInterval = TimeSpan.seconds(30),
      instrumentation = DEFAULT_STATIC_PROPS.instrumentation,
      transform = DEFAULT_STATIC_PROPS.transform,
      staleEntryThreshold = DEFAULT_STATIC_PROPS.staleEntryThreshold,
    } = props;
    this.props = {
      dynamicBufferSize,
      gcInterval,
      instrumentation,
      transform,
      staleEntryThreshold,
    };
    this.gcInterval = setInterval(() => this.gc(), gcInterval.milliseconds);
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
    const { dynamicBufferSize, transform, staleEntryThreshold } = this.props;
    const unary = new Unary({
      dynamicBufferSize,
      transform,
      staleEntryThreshold,
      instrumentation: this.props.instrumentation.child(`cache-${key}`),
    });
    this.cache.set(key, unary);
    return unary;
  }

  /** Garbage collects the cache */
  private gc(): void {
    const {
      instrumentation: { L },
    } = this.props;
    L.info("starting garbage collection");
    const total = zeroGCMetrics();
    this.cache.forEach((c) => {
      const res = c.gc();
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
