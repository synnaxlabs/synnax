// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import {
  array,
  breaker,
  type CrudeTimeSpan,
  type destructor,
  MultiSeries,
  Rate,
  type Series,
  TimeSpan,
} from "@synnaxlabs/x";
import { Mutex } from "async-mutex";

import { type channel } from "@/channel";
import { UnexpectedError } from "@/errors";
import { framer } from "@/framer";
import { status } from "@/status";
import { type Cache } from "@/telem/cache/cache";

/** Receives buffers allocated for the handler's subscribed channels. */
export type StreamHandler = (data: Map<channel.Key, MultiSeries>) => void;

/** Receives per-key data-plane status transitions for subscribed channels. */
export type StatusHandler = (key: channel.Key, status: status.Status) => void;

/**
 * A durable handle on streaming demand. Closing it releases the demand; until then
 * the streamer owns converging the underlying stream to it.
 */
export interface Subscription {
  /** Releases the subscription's demand. */
  close: () => void;
  /** @returns the current data-plane status for the given key. */
  status: (key: channel.Key) => status.Status;
  /** Registers a handler for status transitions on the subscription's keys. */
  onStatusChange: (handler: StatusHandler) => destructor.Destructor;
}

interface Entry {
  handler: StreamHandler;
  keys: Set<channel.Key>;
  statusHandlers: Set<StatusHandler>;
  removed: boolean;
}

export interface StreamerProps {
  cache: Cache;
  openStreamer: framer.StreamOpener;
  instrumentation?: alamos.Instrumentation;
  /**
   * How long a closed subscription's keys stay on the stream, damping churn from
   * rapid unsubscribe and resubscribe cycles.
   * @default TimeSpan.seconds(5)
   */
  removalDelay?: CrudeTimeSpan;
  /** Retry behavior for stream open and update failures. */
  breaker?: breaker.Config;
}

// Coalesces rapid demand changes into one stream update. The timer is not reset by
// later calls, so sustained subscribing cannot starve the update.
const RECONCILE_DELAY = TimeSpan.milliseconds(100);

const THROTTLE_RATE = Rate.hz(60).valueOf();

const LOADING: status.Crude = { variant: "loading", message: "subscribing" };
const STREAMING: status.Crude = { variant: "success", message: "streaming" };
const RECONNECTING: status.Crude = { variant: "warning", message: "reconnecting" };

/**
 * Multiplexes durable subscriptions onto one shared frame stream. Demand
 * registration is synchronous and cannot fail per key; all convergence work
 * surfaces as per-key statuses instead of thrown errors.
 */
export class Streamer {
  private readonly props: Required<Omit<StreamerProps, "removalDelay" | "breaker">> & {
    removalDelay: TimeSpan;
  };

  // Serializes stream open, update, and close against each other. Never held while
  // waiting on retry backoff so close() stays prompt.
  private readonly connMu = new Mutex();
  private readonly breaker: breaker.Breaker;
  private readonly entries = new Set<Entry>();
  private readonly statuses = new Map<channel.Key, status.Status>();
  private reconcileTimer: ReturnType<typeof setTimeout> | null = null;
  private runLoop: Promise<void> | null = null;
  private streamer: framer.HardenedStreamer | null = null;
  // The key set last accepted by the stream. Kept separately from the streamer so a
  // mid-reconnect streamer cannot be consulted for it.
  private sentKeys = new Set<channel.Key>();
  private closed = false;

  constructor(props: StreamerProps) {
    this.props = {
      instrumentation: alamos.NOOP,
      cache: props.cache,
      openStreamer: props.openStreamer,
      removalDelay: new TimeSpan(props.removalDelay ?? TimeSpan.seconds(5)),
    };
    const {
      maxRetries = Infinity,
      baseInterval = TimeSpan.milliseconds(250),
      maxInterval = TimeSpan.seconds(5),
      scale = 2,
      jitter = 0.25,
    } = props.breaker ?? {};
    this.breaker = new breaker.Breaker({
      maxRetries,
      baseInterval,
      maxInterval,
      scale,
      jitter,
    });
  }

  /**
   * Registers demand for the given keys. Registration is synchronous: the handler
   * is seeded with any existing leading buffers, and the stream converges to the
   * new demand in the background.
   * @throws {UnexpectedError} if the streamer has been closed.
   */
  stream(handler: StreamHandler, keys: channel.Key[]): Subscription {
    if (this.closed)
      throw new UnexpectedError("stream() called on a closed telemetry streamer");
    const { cache, instrumentation: ins } = this.props;
    ins.L.debug("adding stream handler", { keys });
    const entry: Entry = {
      handler,
      keys: new Set(keys),
      statusHandlers: new Set(),
      removed: false,
    };
    this.entries.add(entry);
    keys.forEach((key) => {
      if (!this.statuses.has(key)) this.statuses.set(key, status.create(LOADING));
    });
    // Seed the handler with existing dynamic buffers so late subscribers render
    // buffered data immediately.
    const seed: Map<channel.Key, MultiSeries> = new Map(
      keys.map((key) => [
        key,
        new MultiSeries(array.toArray<Series>(cache.get(key).leadingBuffer)),
      ]),
    );
    handler(seed);
    this.scheduleReconcile();
    return {
      close: () => this.remove(entry),
      status: (key) => this.statuses.get(key) ?? status.create(LOADING),
      onStatusChange: (statusHandler) => {
        entry.statusHandlers.add(statusHandler);
        return () => entry.statusHandlers.delete(statusHandler);
      },
    };
  }

  async close(): Promise<void> {
    const { instrumentation: ins } = this.props;
    if (this.reconcileTimer != null) clearTimeout(this.reconcileTimer);
    this.reconcileTimer = null;
    await this.connMu.runExclusive(async () => {
      this.closed = true;
      const prev = this.streamer;
      const prevLoop = this.runLoop;
      this.streamer = null;
      this.runLoop = null;
      try {
        prev?.close();
        if (prevLoop != null) await prevLoop;
      } catch (e) {
        ins.L.error("failed to close streamer", { error: e });
      }
    });
  }

  private remove(entry: Entry): void {
    entry.removed = true;
    setTimeout(() => {
      this.entries.delete(entry);
      this.scheduleReconcile();
    }, this.props.removalDelay.milliseconds);
  }

  private scheduleReconcile(delay: TimeSpan = RECONCILE_DELAY): void {
    if (this.closed || this.reconcileTimer != null) return;
    this.reconcileTimer = setTimeout(() => {
      this.reconcileTimer = null;
      void this.reconcile();
    }, delay.milliseconds);
  }

  /** @returns the union of keys across all entries, including those pending
   * removal, so the stream lags demand only by the removal delay. */
  private desiredKeys(): Set<channel.Key> {
    const keys = new Set<channel.Key>();
    this.entries.forEach((e) => e.keys.forEach((k) => keys.add(k)));
    return keys;
  }

  private setStatus(key: channel.Key, crude: status.Crude): void {
    const prev = this.statuses.get(key);
    if (prev?.variant === crude.variant && prev?.message === crude.message) return;
    const next = status.create(crude);
    this.statuses.set(key, next);
    this.entries.forEach((e) => {
      if (e.removed || !e.keys.has(key)) return;
      e.statusHandlers.forEach((handler) => handler(key, next));
    });
  }

  private async reconcile(): Promise<void> {
    const { instrumentation: ins } = this.props;
    let retry = false;
    await this.connMu.runExclusive(async () => {
      if (this.closed) return;
      const desired = this.desiredKeys();
      try {
        if (desired.size === 0) {
          const prev = this.streamer;
          const prevLoop = this.runLoop;
          this.streamer = null;
          this.runLoop = null;
          this.sentKeys = new Set();
          this.statuses.clear();
          prev?.close();
          if (prevLoop != null) await prevLoop;
          this.breaker.reset();
          ins.L.info("streamer closed, no keys to stream");
          return;
        }
        if (
          desired.size === this.sentKeys.size &&
          Array.from(desired).every((k) => this.sentKeys.has(k))
        )
          return;
        const arrKeys = Array.from(desired);
        if (this.streamer == null) {
          ins.L.info("creating new streamer", { keys: arrKeys });
          const streamer = await framer.HardenedStreamer.open(
            this.props.openStreamer,
            { channels: arrKeys, throttleRate: THROTTLE_RATE },
            undefined,
            () => this.handleReopen(),
            () => this.handleDrop(),
          );
          this.streamer = streamer;
          this.runLoop = this.run(streamer);
        } else {
          ins.L.debug("updating streamer", {
            prev: Array.from(this.sentKeys),
            next: arrKeys,
          });
          await this.streamer.update(arrKeys);
        }
        this.sentKeys = desired;
        desired.forEach((key) => this.setStatus(key, STREAMING));
        this.statuses.forEach((_, key) => {
          if (!desired.has(key)) this.statuses.delete(key);
        });
        this.breaker.reset();
      } catch (e) {
        ins.L.error("failed to update streamer", { error: e });
        const failure = status.fromException(e, "failed to stream channel");
        desired.forEach((key) => {
          if (!this.sentKeys.has(key)) this.setStatus(key, failure);
        });
        retry = true;
      }
    });
    if (!retry || this.closed) return;
    if (await this.breaker.wait()) this.scheduleReconcile(TimeSpan.ZERO);
  }

  private handleDrop(): void {
    this.sentKeys.forEach((key) => this.setStatus(key, RECONNECTING));
  }

  private handleReopen(): void {
    this.sentKeys.forEach((key) => this.setStatus(key, STREAMING));
  }

  private async run(streamer: framer.HardenedStreamer): Promise<void> {
    const { cache, instrumentation: ins } = this.props;
    try {
      for await (const frame of streamer) {
        const changed: Map<channel.Key, MultiSeries> = new Map();
        for (const k of frame.keys)
          try {
            changed.set(k, cache.get(k).writeDynamic(frame.get(k)));
          } catch (e) {
            // One channel's write failure must not stop the stream for the rest.
            this.setStatus(k, status.fromException(e, "failed to buffer channel"));
          }
        if (changed.size !== 0)
          this.entries.forEach((e) => {
            if (!e.removed) e.handler(changed);
          });
      }
    } catch (e) {
      ins.L.error("streamer run loop failed", { error: e }, true);
    } finally {
      // A loop that ends while demand remains is an outage, not a shutdown:
      // schedule a repair instead of leaving the stream silently dead.
      if (!this.closed && this.streamer === streamer) {
        this.streamer = null;
        this.runLoop = null;
        this.sentKeys = new Set();
        this.scheduleReconcile(TimeSpan.ZERO);
      }
    }
  }
}
