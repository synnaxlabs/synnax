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
  sync,
  TimeSpan,
} from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { UnexpectedError } from "@/errors";
import { type Cache } from "@/framer/cache/cache";
import { type Streamer, type StreamerConfig } from "@/framer/streamer";
import { status } from "@/status";

/** Stream lifecycle hooks the streamer uses to track connection health. */
export interface StreamHooks {
  /** Called after every successful reconnect of the underlying stream. */
  onReopen: () => void;
  /** Called when the underlying stream fails and reconnection begins. */
  onDrop: (error: Error) => void;
}

/** Opens the underlying frame stream, hardened against connection loss. */
export interface HardenedOpener {
  (config: StreamerConfig, hooks: StreamHooks): Promise<Streamer>;
}

/**
 * Receives every channel that changed on the shared stream, which may include
 * keys the handler did not subscribe to.
 */
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

export interface MultiplexedStreamerProps {
  cache: Cache;
  openStreamer: HardenedOpener;
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
export class MultiplexedStreamer {
  private readonly props: Required<
    Omit<MultiplexedStreamerProps, "removalDelay" | "breaker">
  > & {
    removalDelay: TimeSpan;
  };

  // Serializes stream open, update, and close against each other. Never held while
  // waiting on retry backoff so close() stays prompt.
  private readonly connMu = sync.newMutex({ closed: false });
  private readonly breaker: breaker.Breaker;
  private readonly entries = new Set<Entry>();
  private readonly statuses = new Map<channel.Key, status.Status>();
  private reconcileTimer: ReturnType<typeof setTimeout> | null = null;
  private readonly removalTimers = new Set<ReturnType<typeof setTimeout>>();
  private runLoop: Promise<void> | null = null;
  private streamer: Streamer | null = null;
  // The key set last accepted by the stream. Kept separately from the streamer so a
  // mid-reconnect streamer cannot be consulted for it.
  private sentKeys = new Set<channel.Key>();

  constructor(props: MultiplexedStreamerProps) {
    const {
      cache,
      openStreamer,
      instrumentation = alamos.NOOP,
      removalDelay = TimeSpan.seconds(5),
    } = props;
    this.props = {
      instrumentation,
      cache,
      openStreamer,
      removalDelay: new TimeSpan(removalDelay),
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
   * is called with any existing leading buffers, and the stream converges to the
   * new demand in the background.
   * @throws {UnexpectedError} if the streamer has been closed.
   */
  stream(handler: StreamHandler, keys: channel.Key[]): Subscription {
    if (this.connMu.closed)
      throw new UnexpectedError("stream() called on a closed telemetry streamer");
    const { cache, instrumentation: ins } = this.props;
    ins.L.debug("adding stream handler", { keys });
    // Deliver existing buffers before registration so a throwing handler leaves no
    // demand behind.
    const initial: Map<channel.Key, MultiSeries> = new Map(
      keys.map((key) => [
        key,
        new MultiSeries(array.toArray<Series>(cache.get(key).leadingBuffer)),
      ]),
    );
    handler(initial);
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
    this.removalTimers.forEach(clearTimeout);
    this.removalTimers.clear();
    await this.connMu.runExclusive(async () => {
      this.connMu.closed = true;
      try {
        await this.releaseStream();
      } catch (e) {
        ins.L.error("failed to close streamer", { error: e });
      }
    });
  }

  private async releaseStream(): Promise<void> {
    const prev = this.streamer;
    const prevLoop = this.runLoop;
    this.streamer = null;
    this.runLoop = null;
    prev?.close();
    if (prevLoop != null) await prevLoop;
  }

  private remove(entry: Entry): void {
    entry.removed = true;
    const timer = setTimeout(() => {
      this.removalTimers.delete(timer);
      this.entries.delete(entry);
      this.scheduleReconcile();
    }, this.props.removalDelay.milliseconds);
    this.removalTimers.add(timer);
  }

  private scheduleReconcile(delay: TimeSpan = RECONCILE_DELAY): void {
    if (this.connMu.closed || this.reconcileTimer != null) return;
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
      e.statusHandlers.forEach((handler) => {
        try {
          handler(key, next);
        } catch (err) {
          this.props.instrumentation.L.error(
            "status handler failed",
            { error: err },
            true,
          );
        }
      });
    });
  }

  private async reconcile(): Promise<void> {
    const { instrumentation: ins } = this.props;
    let retry = false;
    await this.connMu.runExclusive(async () => {
      if (this.connMu.closed) return;
      const desired = this.desiredKeys();
      try {
        if (desired.size === 0) {
          this.sentKeys = new Set();
          this.statuses.clear();
          await this.releaseStream();
          this.breaker.reset();
          ins.L.info("streamer closed, no keys to stream");
          return;
        }
        if (
          this.streamer != null &&
          desired.size === this.sentKeys.size &&
          desired.isSubsetOf(this.sentKeys)
        )
          return;
        const arrKeys = Array.from(desired);
        const fresh = arrKeys.filter((k) => !this.sentKeys.has(k));
        if (this.streamer == null) {
          ins.L.info("creating new streamer", { keys: arrKeys });
          const streamer = await this.props.openStreamer(
            { channels: arrKeys, throttleRate: THROTTLE_RATE },
            {
              onReopen: () =>
                this.sentKeys.forEach((key) => this.setStatus(key, STREAMING)),
              onDrop: () =>
                this.sentKeys.forEach((key) => this.setStatus(key, RECONNECTING)),
            },
          );
          this.streamer = streamer;
          this.runLoop = this.run(streamer);
        } else {
          ins.L.debug("updating streamer", () => ({
            prev: Array.from(this.sentKeys),
            next: arrKeys,
          }));
          await this.streamer.update(arrKeys);
        }
        this.sentKeys = desired;
        // Existing keys keep their status so a demand change cannot clear a per-key
        // error.
        fresh.forEach((key) => this.setStatus(key, STREAMING));
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
    if (!retry || this.connMu.closed) return;
    if (await this.breaker.wait()) this.scheduleReconcile(TimeSpan.ZERO);
  }

  private async run(streamer: Streamer): Promise<void> {
    const { cache, instrumentation: ins } = this.props;
    try {
      for await (const frame of streamer) {
        // Group series by key in one pass; per-key get() scans the whole frame.
        const grouped: Map<channel.Key, Series[]> = new Map();
        frame.forEach((k, s) => {
          const key = k as channel.Key;
          const existing = grouped.get(key);
          if (existing == null) grouped.set(key, [s]);
          else existing.push(s);
        });
        const changed: Map<channel.Key, MultiSeries> = new Map();
        grouped.forEach((series, k) => {
          try {
            changed.set(k, cache.get(k).writeDynamic(new MultiSeries(series)));
            this.setStatus(k, STREAMING);
          } catch (e) {
            this.setStatus(k, status.fromException(e, "failed to buffer channel"));
          }
        });
        if (changed.size !== 0)
          this.entries.forEach((e) => {
            if (e.removed) return;
            try {
              e.handler(changed);
            } catch (err) {
              ins.L.error("stream handler failed", { error: err }, true);
            }
          });
      }
    } catch (e) {
      ins.L.error("streamer run loop failed", { error: e }, true);
    } finally {
      // A loop that ends while demand remains is an outage, not a shutdown.
      if (!this.connMu.closed && this.streamer === streamer) {
        this.streamer = null;
        this.runLoop = null;
        this.sentKeys = new Set();
        this.scheduleReconcile(TimeSpan.ZERO);
      }
    }
  }
}
