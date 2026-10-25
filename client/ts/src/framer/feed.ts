// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type DataType,
  debounce,
  type MultiSeries,
  type Size,
  TimeRange,
  type TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { UnexpectedError } from "@/errors";
import { Joiner } from "@/framer/cache/joiner";
import {
  DEFAULT_BATCH_DEBOUNCE,
  Reader,
  type ReaderProps,
} from "@/framer/cache/reader";
import { Store, type StoreProps } from "@/framer/cache/store";
import {
  type Demand,
  MultiplexedStreamer,
  type MultiplexedStreamerProps,
  type StreamHandler,
} from "@/framer/cache/streamer";
import { Handle } from "@/framer/cache/subscription";
import { TileReader, type TileSpec } from "@/framer/cache/tile";
import { type Frame } from "@/framer/frame";
import { HardenedStreamer } from "@/framer/hardened";
import { type StreamOpener } from "@/framer/streamer";
import { type SubscribeRequest, type Subscription } from "@/framer/subscription";
import { historyChangeZ } from "@/framer/types.gen";

/** Reads the newest count stored samples of each channel in the time range. */
export interface LatestReader {
  (keys: channel.Key[], count: number, tr: TimeRange): Promise<Frame>;
}

// The channel the Core publishes history changes on.
const HISTORY_CHANGE_CHANNEL = "sy_history_change";

interface LatestRequest {
  key: channel.Key;
  count: number;
  tr: TimeRange;
}

/** Props for {@link Feed}, including the transport it reads and streams through. */
export interface FeedProps
  extends
    Pick<StoreProps, "dynamicBufferSize" | "instrumentation">,
    Omit<ReaderProps, "store">,
    Omit<MultiplexedStreamerProps, "store" | "openStreamer"> {
  openStreamer: StreamOpener;
  readLatest: LatestReader;
  /**
   * Byte size the block cache evicts down to. Blocks a caller holds and blocks the
   * stream writes into are never evicted.
   * @default Size.megabytes(128)
   */
  budget?: Size;
  /**
   * How long a fetched block answers reads before it is fetched again. Bounds the
   * refetch rate of spans that stay empty. A block a caller holds keeps answering.
   * @default TimeSpan.minutes(10)
   */
  staleCoverageThreshold?: TimeSpan;
  /** Receives each failed reconnect attempt of the stream. Defaults to console.warn. */
  onRetry?: (error: Error) => void;
}

/** The part of {@link FeedProps} a caller sets. The client supplies the rest. */
export interface FeedOptions extends Omit<
  FeedProps,
  "readRemote" | "retrieveChannels" | "openStreamer" | "readLatest" | "onRetry"
> {}

/**
 * Cached, demand-managed access to channel telemetry: durable streaming subscriptions
 * multiplexed onto one frame stream, and batched historical reads, all cached in raw
 * and reduced blocks under one byte budget.
 */
export class Feed {
  private readonly reader: Reader;
  private readonly streamer: MultiplexedStreamer;
  private readonly tiles: TileReader;
  private readonly store: Store;
  private readonly joiner: Joiner;
  private readonly latest: debounce.Batcher<LatestRequest, MultiSeries>;
  private readonly props: FeedProps;
  // signals carries the history changes once a subscription asks for them.
  private signals: Promise<Demand | null> | null = null;
  private closed = false;

  constructor(props: FeedProps) {
    const {
      readRemote,
      retrieveChannels,
      openStreamer,
      readLatest,
      budget,
      instrumentation,
      dynamicBufferSize,
      staleCoverageThreshold,
      removalDelay,
      breaker,
      batchDebounce,
      fetchTimeout,
      now,
      onRetry,
    } = props;
    this.props = props;
    this.store = new Store({
      budget,
      staleThreshold: staleCoverageThreshold,
      dynamicBufferSize,
      instrumentation: instrumentation?.child("store"),
      now,
    });
    this.reader = new Reader({
      readRemote,
      retrieveChannels,
      store: this.store,
      batchDebounce,
      fetchTimeout,
      now,
    });
    this.tiles = new TileReader({
      readRemote,
      store: this.store,
      batchDebounce,
      fetchTimeout,
      now,
    });
    this.streamer = new MultiplexedStreamer({
      store: this.store,
      openStreamer: async (config, { onReopen, onDrop, signal }) => {
        const streamer = new HardenedStreamer(openStreamer, config, {
          breaker,
          onReopen,
          onDrop,
          onRetry,
        });
        signal.addEventListener("abort", () => streamer.close());
        await streamer.start();
        return streamer;
      },
      removalDelay,
      breaker,
      instrumentation: instrumentation?.child("streamer"),
    });
    this.joiner = new Joiner({
      store: this.store,
      reader: this.reader,
      streamer: this.streamer,
      now: this.now,
      breaker,
    });
    this.latest = new debounce.Batcher({
      interval: batchDebounce ?? DEFAULT_BATCH_DEBOUNCE,
      exec: async (entries) => {
        // Requests of one count and range share one read.
        const groups = new Map<string, typeof entries>();
        entries.forEach((e) => {
          const key = `${e.req.count}/${e.req.tr.toString()}`;
          const group = groups.get(key);
          if (group == null) groups.set(key, [e]);
          else group.push(e);
        });
        await Promise.all(
          [...groups.values()].map(async (group) => {
            const { count, tr } = group[0].req;
            try {
              const keys = [...new Set(group.map(({ req }) => req.key))];
              const frame = await readLatest(keys, count, tr);
              group.forEach(({ req, resolve }) => resolve(frame.get(req.key)));
            } catch (e) {
              group.forEach(({ reject }) => reject(e));
            }
          }),
        );
      },
    });
  }

  private get now(): () => TimeStamp {
    return this.props.now ?? (() => TimeStamp.now());
  }

  /**
   * Subscribes to the channels of the request. The subscription reads and streams in
   * the background, and reports failures as statuses of its snapshots.
   * @throws {UnexpectedError} if the feed has been closed.
   */
  subscribe(request: SubscribeRequest): Subscription {
    if (this.closed) throw new UnexpectedError("telemetry feed is closed");
    this.signals ??= this.listen();
    return new Handle(
      {
        store: this.store,
        reader: this.reader,
        tiles: this.tiles,
        streamer: this.streamer,
        joiner: this.joiner,
        retrieveChannels: this.props.retrieveChannels,
        readLatest: async (key, count, tr) =>
          await this.latest.enqueue({ key, count, tr }),
        now: this.now,
        breaker: this.props.breaker,
      },
      request,
    );
  }

  // Streams the history changes, which drop the cached samples they name. A failure to
  // find the channel leaves fetched blocks to expire, and the next subscription tries
  // again.
  private async listen(): Promise<Demand | null> {
    const { retrieveChannels, instrumentation } = this.props;
    try {
      const [payload] = await retrieveChannels([HISTORY_CHANGE_CHANNEL]);
      if (payload == null)
        throw new UnexpectedError(`${HISTORY_CHANGE_CHANNEL} not found`);
      if (this.closed) return null;
      this.store.setSignalKey(payload.key);
      return this.streamer.demand([payload.key], (frame) => {
        frame.get(payload.key)?.forEach((s) =>
          s
            .parseJSON(historyChangeZ.array())
            .flat()
            .forEach(({ key, timeRange }) => this.store.invalidate(key, timeRange)),
        );
      });
    } catch (e) {
      instrumentation?.L.warn("failed to stream history changes", { error: e });
      this.signals = null;
      return null;
    }
  }

  /**
   * Registers durable streaming demand for the given keys. Returns synchronously;
   * convergence failures surface as per-key statuses, never as rejections. The
   * handler's first call precedes registration, so a throw from it leaves no
   * subscription behind.
   * @throws {UnexpectedError} if the feed has been closed.
   */
  stream(handler: StreamHandler, keys: channel.Key[]): Demand {
    return this.streamer.stream(handler, keys);
  }

  /**
   * Reads the given time range for the given channel, serving from cached raw blocks
   * and fetching the blocks the cache lacks. Series are not cut to the range, so the
   * result can hold samples outside it.
   * @throws {NotFoundError} if the channel does not exist.
   * @throws {UnexpectedError} if the feed is closed while the read is pending.
   * @throws {Unreachable} if a fetch serving the read exceeds the fetch timeout.
   */
  async read(tr: TimeRange, key: channel.Key): Promise<MultiSeries> {
    return await this.reader.read(tr, key);
  }

  /**
   * Reads one tile of a channel, reduced under the tile's point limit. Finished tiles
   * are cached, so the result may be shared with other callers. Reads of tiles with
   * equal bounds and options within one batch window share a single request, so their
   * series share one group size.
   * @param dataType - The data type of the series the caller holds. A cached tile of
   * another data type is read again.
   * @throws {UnexpectedError} if the feed is closed while the read is pending.
   * @throws {Unreachable} if the fetch serving the read exceeds the fetch timeout.
   */
  async readTile(spec: TileSpec, dataType?: DataType): Promise<MultiSeries> {
    return await this.tiles.read(spec, dataType);
  }

  /**
   * Reads the latest stored sample of the given channel, in the representation the
   * cache serves. Reads within one batch window share a single request.
   * @throws {UnexpectedError} if the feed is closed while the read is pending.
   */
  async readLatest(key: channel.Key): Promise<MultiSeries> {
    if (this.closed) throw new UnexpectedError("telemetry feed is closed");
    return await this.latest.enqueue({ key, count: 1, tr: TimeRange.MAX });
  }

  /** Closes the feed, releasing the stream and all cached buffers. */
  async close(): Promise<void> {
    this.closed = true;
    this.latest.close(new UnexpectedError("telemetry feed is closed"));
    this.joiner.close();
    (await this.signals)?.close();
    await this.streamer.close();
    await this.reader.close();
    this.tiles.close();
    this.store.close();
  }
}
