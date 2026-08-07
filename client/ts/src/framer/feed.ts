// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type MultiSeries, type TimeRange } from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { Cache, type CacheProps } from "@/framer/cache/cache";
import { Reader, type ReaderProps } from "@/framer/cache/reader";
import {
  Streamer,
  type StreamerProps,
  type StreamHandler,
  type Subscription,
} from "@/framer/cache/streamer";
import { HardenedStreamer } from "@/framer/hardened";
import { type StreamOpener } from "@/framer/streamer";

export interface FeedProps
  extends
    CacheProps,
    Omit<ReaderProps, "cache">,
    Omit<StreamerProps, "cache" | "openStreamer"> {
  openStreamer: StreamOpener;
}

export interface FeedOptions extends Omit<FeedProps, "readRemote" | "openStreamer"> {}

/**
 * Cached, demand-managed access to channel telemetry: durable streaming
 * subscriptions multiplexed onto one frame stream, and batched historical reads
 * served through per-channel rolling buffers.
 */
export class Feed {
  private readonly cache: Cache;
  private readonly reader: Reader;
  private readonly streamer: Streamer;

  constructor(props: FeedProps) {
    const {
      readRemote,
      openStreamer,
      transform,
      instrumentation,
      dynamicBufferSize,
      gcInterval,
      staleEntryThreshold,
      removalDelay,
      breaker,
      batchDebounce,
      overlapThreshold,
    } = props;
    this.cache = new Cache({
      transform,
      dynamicBufferSize,
      gcInterval,
      staleEntryThreshold,
      instrumentation: instrumentation?.child("cache"),
    });
    this.reader = new Reader({
      readRemote,
      cache: this.cache,
      batchDebounce,
      overlapThreshold,
    });
    this.streamer = new Streamer({
      cache: this.cache,
      openStreamer: async (config, { onReopen, onDrop }) =>
        await HardenedStreamer.open(openStreamer, config, breaker, onReopen, onDrop),
      removalDelay,
      breaker,
      instrumentation: instrumentation?.child("streamer"),
    });
  }

  /**
   * Registers durable streaming demand for the given keys. Returns synchronously;
   * failures surface as per-key statuses on the subscription, never as rejections.
   */
  stream(handler: StreamHandler, keys: channel.Key[]): Subscription {
    return this.streamer.stream(handler, keys);
  }

  /**
   * Reads the given time range for the given channel, serving from the cache and
   * gap-filling from the cluster.
   */
  async read(tr: TimeRange, key: channel.Key): Promise<MultiSeries> {
    return await this.reader.read(tr, key);
  }

  /** Closes the feed, releasing the stream and all cached buffers. */
  async close(): Promise<void> {
    await this.streamer.close();
    await this.reader.close();
    this.cache.close();
  }
}
