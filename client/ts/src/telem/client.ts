// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type alamos } from "@synnaxlabs/alamos";
import {
  type breaker,
  type CrudeTimeSpan,
  type MultiSeries,
  type TimeRange,
  type TimeSpan,
} from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { framer } from "@/framer";
import { Cache } from "@/telem/cache/cache";
import { Reader, type ReadRemoteFunc } from "@/telem/reader";
import { Streamer, type StreamHandler, type Subscription } from "@/telem/streamer";
import { type Transform } from "@/telem/transform";

export interface ClientProps {
  /** Reads a frame of historical data from the cluster. */
  readRemote: ReadRemoteFunc;
  /** Opens the underlying frame stream. */
  openStreamer: framer.StreamOpener;
  /** Applied to every series before it enters cache buffers. Defaults to identity. */
  transform?: Transform;
  instrumentation?: alamos.Instrumentation;
  /** See {@link Cache} */
  dynamicBufferSize?: number | TimeSpan;
  /** See {@link Cache} */
  gcInterval?: TimeSpan;
  /** See {@link Cache} */
  staleEntryThreshold?: TimeSpan;
  /** See {@link Streamer} */
  removalDelay?: CrudeTimeSpan;
  /** See {@link Streamer} */
  breaker?: breaker.Config;
  /** See {@link Reader} */
  batchDebounce?: TimeSpan;
  /** See {@link Reader} */
  overlapThreshold?: TimeSpan;
}

/** The subset of {@link ClientProps} an owner may tune from the outside. */
export interface Options extends Pick<ClientProps, "transform"> {}

/**
 * Cached, demand-managed access to channel telemetry: durable streaming
 * subscriptions multiplexed onto one frame stream, and batched historical reads
 * served through per-channel rolling buffers.
 */
export class Client {
  private readonly cache: Cache;
  private readonly reader: Reader;
  private readonly streamer: Streamer;

  constructor(props: ClientProps) {
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
      instrumentation: instrumentation?.child("reader"),
    });
    this.streamer = new Streamer({
      cache: this.cache,
      openStreamer: async (config, { onReopen, onDrop }) =>
        await framer.HardenedStreamer.open(
          openStreamer,
          config,
          breaker,
          onReopen,
          onDrop,
        ),
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

  /** Closes the client, releasing the stream and all cached buffers. */
  async close(): Promise<void> {
    await this.streamer.close();
    await this.reader.close();
    this.cache.close();
  }
}
