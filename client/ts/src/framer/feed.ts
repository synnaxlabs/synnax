// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { debounce, MultiSeries, type TimeRange } from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { UnexpectedError } from "@/errors";
import { Cache, type CacheProps } from "@/framer/cache/cache";
import {
  DEFAULT_BATCH_DEBOUNCE,
  Reader,
  type ReaderProps,
} from "@/framer/cache/reader";
import { DEFAULT_STATIC_PROPS } from "@/framer/cache/static";
import {
  MultiplexedStreamer,
  type MultiplexedStreamerProps,
  type StreamHandler,
  type Subscription,
} from "@/framer/cache/streamer";
import { type Frame } from "@/framer/frame";
import { HardenedStreamer } from "@/framer/hardened";
import { type StreamOpener } from "@/framer/streamer";

/** A function that reads the latest stored sample of each channel from the cluster. */
export interface LatestReader {
  (keys: channel.Key[]): Promise<Frame>;
}

type LatestBatch = Array<debounce.Entry<channel.Key, MultiSeries>>;

/** Props for {@link Feed}, including the transport it reads and streams through. */
export interface FeedProps
  extends
    CacheProps,
    Omit<ReaderProps, "cache">,
    Omit<MultiplexedStreamerProps, "cache" | "openStreamer"> {
  openStreamer: StreamOpener;
  readLatest: LatestReader;
}

/** The part of {@link FeedProps} a caller sets. The client supplies the rest. */
export interface FeedOptions extends Omit<
  FeedProps,
  "readRemote" | "openStreamer" | "readLatest"
> {}

/**
 * Cached, demand-managed access to channel telemetry: durable streaming
 * subscriptions multiplexed onto one frame stream, and batched historical reads
 * served through per-channel rolling buffers.
 */
export class Feed {
  private readonly cache: Cache;
  private readonly reader: Reader;
  private readonly streamer: MultiplexedStreamer;
  private readonly latest: debounce.Batcher<channel.Key, MultiSeries>;
  private readonly inFlight = new Set<LatestBatch>();
  private closed = false;

  constructor(props: FeedProps) {
    const {
      readRemote,
      openStreamer,
      readLatest,
      transform = DEFAULT_STATIC_PROPS.transform,
      instrumentation,
      dynamicBufferSize,
      gcInterval,
      staleEntryThreshold,
      staleCoverageThreshold,
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
      staleCoverageThreshold,
      instrumentation: instrumentation?.child("cache"),
    });
    this.reader = new Reader({
      readRemote,
      cache: this.cache,
      batchDebounce,
      overlapThreshold,
    });
    this.streamer = new MultiplexedStreamer({
      cache: this.cache,
      openStreamer: async (config, { onReopen, onDrop }) =>
        await HardenedStreamer.open(openStreamer, config, breaker, onReopen, onDrop),
      removalDelay,
      breaker,
      instrumentation: instrumentation?.child("streamer"),
    });
    this.latest = new debounce.Batcher({
      interval: batchDebounce ?? DEFAULT_BATCH_DEBOUNCE,
      exec: async (entries) => {
        if (this.closed) throw new UnexpectedError("telemetry feed is closed");
        this.inFlight.add(entries);
        try {
          const frame = await readLatest([...new Set(entries.map(({ req }) => req))]);
          entries.forEach(({ req, resolve }) => {
            const series = frame.get(req).series.map((s) => transform.convert(s));
            resolve(new MultiSeries(series));
          });
        } finally {
          this.inFlight.delete(entries);
        }
      },
    });
  }

  /**
   * Registers durable streaming demand for the given keys. Returns synchronously;
   * convergence failures surface as per-key statuses, never as rejections. The
   * handler's first call precedes registration, so a throw from it leaves no
   * subscription behind.
   * @throws {UnexpectedError} if the feed has been closed.
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

  /**
   * Reads the latest stored sample of the given channel, in the representation the
   * cache serves. Reads within one batch window share a single request.
   * @throws {UnexpectedError} if the feed is closed while the read is pending.
   */
  async readLatest(key: channel.Key): Promise<MultiSeries> {
    if (this.closed) throw new UnexpectedError("telemetry feed is closed");
    const entry = this.cache.get(key);
    const stored = entry.latest;
    if (stored != null) return stored;
    const version = entry.version;
    const latest = await this.latest.enqueue(key);
    if (this.streamer.live(key)) entry.storeLatest(latest, version);
    return latest;
  }

  /** Closes the feed, releasing the stream and all cached buffers. */
  async close(): Promise<void> {
    this.closed = true;
    const err = new UnexpectedError("telemetry feed is closed");
    this.latest.close(err);
    this.inFlight.forEach((batch) => batch.forEach(({ reject }) => reject(err)));
    this.inFlight.clear();
    await this.streamer.close();
    await this.reader.close();
    this.cache.close();
  }
}
