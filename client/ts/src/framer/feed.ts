// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type MultiSeries,
  scheduler,
  type Size,
  type TimeRange,
  TimeStamp,
} from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { isConnectionError, UnexpectedError } from "@/errors";
import { Joins } from "@/framer/cache/join";
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
} from "@/framer/cache/streamer";
import { Handle } from "@/framer/cache/subscription";
import { TileReader } from "@/framer/cache/tile";
import { type Frame } from "@/framer/frame";
import { HardenedStreamer } from "@/framer/hardened";
import { type StreamOpener } from "@/framer/streamer";
import { type SubscribeRequest, type Subscription } from "@/framer/subscription";
import { historyChangeZ } from "@/framer/types.gen";

/** Reads the newest count stored samples of each channel in the time range. */
export interface RemoteLatestReader {
  (keys: channel.Key[], count: number, tr: TimeRange): Promise<Frame>;
}

// The channel the Core publishes history changes on.
const HISTORY_CHANGE_CHANNEL = "sy_history_change";

interface LatestQuery {
  key: channel.Key;
  count: number;
  tr: TimeRange;
}

type LatestEntry = scheduler.BatcherEntry<LatestQuery, MultiSeries>;

/** Props for {@link Feed}, including the transport it reads and streams through. */
export interface FeedProps
  extends
    Pick<StoreProps, "instrumentation">,
    Omit<ReaderProps, "store">,
    Omit<MultiplexedStreamerProps, "store" | "joins" | "onWait" | "openStreamer"> {
  openStreamer: StreamOpener;
  readLatest: RemoteLatestReader;
  /**
   * The key of the Core node the client connects to, once known. History signals keep
   * the blocks of only the channels stored on it.
   * @default a node that is never known
   */
  node?: () => number | undefined;
  /**
   * Byte size the block cache evicts down to. Blocks a subscription serves and blocks
   * the stream writes into are never evicted.
   * @default Size.megabytes(128)
   */
  budget?: Size;
  /** Receives each failed reconnect attempt of the stream. Defaults to console.warn. */
  onRetry?: (error: Error) => void;
}

/** The part of {@link FeedProps} a caller sets. The client supplies the rest. */
export interface FeedOptions extends Omit<
  FeedProps,
  "readRemote" | "retrieveChannels" | "openStreamer" | "readLatest" | "onRetry" | "node"
> {}

/**
 * Cached access to channel telemetry through subscriptions. Subscriptions share one
 * frame stream and batched historical reads, cached in raw and reduced blocks under one
 * byte budget.
 */
export class Feed {
  private readonly reader: Reader;
  private readonly streamer: MultiplexedStreamer;
  private readonly tiles: TileReader;
  private readonly store: Store;
  private readonly joins: Joins;
  private readonly joiner: Joiner;
  private readonly latest: scheduler.Batcher<LatestQuery, MultiSeries>;
  private readonly props: FeedProps;
  private readonly handles = new Set<Handle>();
  // listening is true once a subscription asked for the history changes, and signals
  // carries them once their channel is found.
  private listening = false;
  private signals: Demand | null = null;
  private closed = false;

  constructor(props: FeedProps) {
    const {
      readRemote,
      retrieveChannels,
      openStreamer,
      budget,
      instrumentation,
      removalDelay,
      breaker,
      batchDebounce,
      fetchTimeout,
      now,
      onRetry,
      node,
    } = props;
    this.props = props;
    this.store = new Store({
      budget,
      instrumentation: instrumentation?.child("store"),
      now,
      node,
    });
    this.joins = new Joins(this.store);
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
      joins: this.joins,
      onWait: (indexes) => this.joiner.wake(indexes),
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
      joins: this.joins,
      reader: this.reader,
      streamer: this.streamer,
      now: this.now,
      breaker,
    });
    this.latest = new scheduler.Batcher({
      interval: batchDebounce ?? DEFAULT_BATCH_DEBOUNCE,
      exec: async (entries) => {
        // Requests of one count and range share one read.
        const groups = Map.groupBy(
          entries,
          ({ req }) => `${req.count}/${req.tr.toString()}`,
        );
        await Promise.all(
          [...groups.values()].map(async (group) => await this.fetchLatest(group)),
        );
      },
    });
  }

  // Never throws: every outcome settles the entries, which share a count and range.
  private async fetchLatest(entries: LatestEntry[]): Promise<void> {
    const { count, tr } = entries[0].req;
    try {
      const keys = [...new Set(entries.map(({ req }) => req.key))];
      const frame = await this.props.readLatest(keys, count, tr);
      entries.forEach(({ req, resolve }) => resolve(frame.get(req.key)));
    } catch (e) {
      const byChannel = Map.groupBy(entries, ({ req }) => req.key);
      // One channel's error fails a read of several, so each channel reads again alone
      // and gets its own result.
      if (byChannel.size > 1 && !isConnectionError(e))
        await Promise.all(
          [...byChannel.values()].map(async (group) => await this.fetchLatest(group)),
        );
      else entries.forEach(({ reject }) => reject(e));
    }
  }

  private get now(): () => TimeStamp {
    return this.props.now ?? (() => TimeStamp.now());
  }

  /**
   * Subscribes to the channels of the request. The subscription reads and streams in
   * the background, and reports failures as statuses of its snapshots.
   * @throws {UnexpectedError} if the feed has been closed.
   * @throws {ValidationError} if the count of a points limit is not finite or is
   * negative.
   */
  subscribe(request: SubscribeRequest): Subscription {
    if (this.closed) throw new UnexpectedError("telemetry feed is closed");
    if (!this.listening) void this.listen();
    const handle: Handle = new Handle(
      {
        store: this.store,
        joins: this.joins,
        reader: this.reader,
        tiles: this.tiles,
        streamer: this.streamer,
        joiner: this.joiner,
        retrieveChannels: this.props.retrieveChannels,
        readLatest: async (key, count, tr) =>
          await this.latest.enqueue({ key, count, tr }),
        now: this.now,
        breaker: this.props.breaker,
        onClose: () => this.handles.delete(handle),
        instrumentation: this.props.instrumentation?.child("subscription"),
      },
      request,
    );
    this.handles.add(handle);
    return handle;
  }

  // Streams the history changes, which drop the cached samples they name. A failure to
  // find the channel leaves fetched blocks to expire, and the next subscription tries
  // again.
  private async listen(): Promise<void> {
    const { retrieveChannels, instrumentation } = this.props;
    this.listening = true;
    try {
      const [payload] = await retrieveChannels([HISTORY_CHANGE_CHANNEL]);
      if (payload == null)
        throw new UnexpectedError(`${HISTORY_CHANGE_CHANNEL} not found`);
      if (this.closed) return;
      this.store.setSignalKey(payload.key);
      this.signals = this.streamer.demand([payload.key], (frame) => {
        frame.get(payload.key)?.forEach((s) =>
          s
            .parseJSON(historyChangeZ.array())
            .flat()
            .forEach(({ key, timeRange }) =>
              this.joiner.wake(this.joins.invalidate(key, timeRange)),
            ),
        );
      });
    } catch (e) {
      instrumentation?.L.warn("failed to stream history changes", { error: e });
      this.listening = false;
    }
  }

  /**
   * Closes the feed and every open subscription, releasing the stream and all cached
   * buffers.
   */
  async close(): Promise<void> {
    this.closed = true;
    this.handles.forEach((h) => h.close());
    this.latest.close(new UnexpectedError("telemetry feed is closed"));
    this.joiner.close();
    this.signals?.close();
    this.signals = null;
    await this.streamer.close();
    await this.reader.close();
    this.tiles.close();
    this.joins.close();
    this.store.close();
  }
}
