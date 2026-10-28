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
  breaker,
  type CrudeTimeSpan,
  type destructor,
  errors,
  Rate,
  scheduler,
  type Series,
  sync,
  TimeSpan,
} from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { UnexpectedError } from "@/errors";
import { type Joins } from "@/framer/cache/join";
import { type Store } from "@/framer/cache/store";
import { isDefinitive } from "@/framer/retry";
import { type Streamer, type StreamerConfig } from "@/framer/streamer";
import { status } from "@/status";

/** Stream lifecycle hooks the streamer uses to track connection health. */
export interface StreamHooks {
  /** Called after every successful reconnect of the underlying stream. */
  onReopen: () => void;
  /** Called when the underlying stream fails and reconnection begins. */
  onDrop: (error: Error) => void;
  /** Aborts when the streamer closes while this stream is current. A stream still
   * opening or retrying must stop. */
  signal: AbortSignal;
}

/** Opens the underlying frame stream, hardened against connection loss. */
export interface HardenedOpener {
  (config: StreamerConfig, hooks: StreamHooks): Promise<Streamer>;
}

/**
 * Receives each frame of the shared stream: the series of each channel as the Core
 * streamed them.
 */
export type FrameHandler = (frame: Map<channel.Key, Series[]>) => void;

/** Receives per-key data-plane status transitions for demanded channels. */
export type StatusHandler = (key: channel.Key, status: status.Status) => void;

/**
 * A durable handle on streaming demand. Closing it releases the demand; until then
 * the streamer owns converging the underlying stream to it.
 */
export interface Demand {
  /** Releases the demand. */
  close: () => void;
  /** @returns the current data-plane status for the given key. */
  status: (key: channel.Key) => status.Status;
  /** Registers a handler for status transitions on the demand's keys. */
  onStatusChange: (handler: StatusHandler) => destructor.Destructor;
  /** Registers a handler that runs when the stream drops, since frames may be lost. */
  onDrop: (handler: () => void) => destructor.Destructor;
}

interface Entry {
  handler: FrameHandler;
  keys: Set<channel.Key>;
  statusHandlers: Set<StatusHandler>;
  dropHandlers: Set<() => void>;
  removed: boolean;
}

// Waits for the stream to ack a set of keys.
interface AckWaiter {
  keys: channel.Key[];
  resolve: () => void;
  reject: (e: Error) => void;
}

export interface MultiplexedStreamerProps {
  store: Store;
  /** Joins the stream's frames to the raw blocks of joined channels. */
  joins: Joins;
  /** Receives the index channels of the joins that wait for a catch-up read. */
  onWait: (indexes: Set<channel.Key>) => void;
  openStreamer: HardenedOpener;
  instrumentation?: alamos.Instrumentation;
  /**
   * How long the keys of a released demand stay on the stream, damping churn from rapid
   * release and demand cycles.
   * @default TimeSpan.seconds(5)
   */
  removalDelay?: CrudeTimeSpan;
  /** Retry behavior for stream open and update failures. */
  breaker?: breaker.Config;
}

// Limits stream updates to one per interval while demand keeps changing.
const RECONCILE_INTERVAL = TimeSpan.milliseconds(100);

const THROTTLE_RATE = Rate.hz(60).valueOf();

const LOADING: status.Crude = { variant: "loading", message: "subscribing" };
const STREAMING: status.Crude = { variant: "success", message: "streaming" };
const RECONNECTING: status.Crude = { variant: "warning", message: "reconnecting" };

/**
 * Multiplexes durable stream demands onto one shared frame stream. Demand registration
 * is synchronous and cannot fail per key; all convergence work surfaces as per-key
 * statuses instead of thrown errors.
 */
export class MultiplexedStreamer {
  private readonly props: Required<
    Omit<MultiplexedStreamerProps, "removalDelay" | "breaker">
  > & {
    removalDelay: TimeSpan;
  };

  // Serializes stream open, update, and close against each other. An open or update
  // retries while holding it, so close() interrupts them before acquiring it.
  private readonly connMu = sync.newMutex({ closed: false });
  // Belongs to the current stream, so a retired stream's abort listener is collected
  // with it.
  private streamAbort: AbortController | null = null;
  private readonly breaker: breaker.Breaker;
  private readonly entries = new Set<Entry>();
  // Indexes entries by key, so a status change visits only the entries of its key.
  private readonly entriesByKey = new Map<channel.Key, Set<Entry>>();
  private readonly statuses = new Map<channel.Key, status.Status>();
  private readonly scheduleReconcile = scheduler.throttle(() => {
    void this.reconcile();
  }, RECONCILE_INTERVAL);
  private readonly removalTimers = new Set<ReturnType<typeof setTimeout>>();
  private runLoop: Promise<void> | null = null;
  private streamer: Streamer | null = null;
  // The key set last accepted by the stream. Kept separately from the streamer so a
  // mid-reconnect streamer cannot be consulted for it.
  private sentKeys = new Set<channel.Key>();
  // outdated is true from a stream drop until a stream opens again.
  private outdated = false;
  // Keys the Core rejected on their own, with the rejection. They stay off the stream
  // so the rest of it keeps flowing.
  private readonly rejected = new Map<channel.Key, Error>();
  // Keys held back while a rejection of several new keys is narrowed down. One returns
  // to the stream per successful reconcile.
  private deferred: channel.Key[] = [];
  // The keys the current stream acked. A drop clears them until the stream reopens.
  private acked = new Set<channel.Key>();
  private readonly ackWaiters = new Set<AckWaiter>();

  constructor(props: MultiplexedStreamerProps) {
    const {
      store,
      joins,
      onWait,
      openStreamer,
      instrumentation = alamos.NOOP,
      removalDelay = TimeSpan.seconds(5),
    } = props;
    this.props = {
      instrumentation,
      store,
      joins,
      onWait,
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
   * Registers demand for the given keys, and passes each frame of the stream to the
   * handler. The stream converges to the new demand in the background.
   * @throws {UnexpectedError} if the streamer has been closed.
   */
  demand(keys: channel.Key[], handler: FrameHandler): Demand {
    if (this.connMu.closed)
      throw new UnexpectedError("demand() called on a closed telemetry streamer");
    this.props.instrumentation.L.debug("adding stream demand", { keys });
    const entry: Entry = {
      handler,
      keys: new Set(keys),
      statusHandlers: new Set(),
      dropHandlers: new Set(),
      removed: false,
    };
    this.entries.add(entry);
    keys.forEach((key) => {
      if (!this.statuses.has(key)) this.statuses.set(key, status.create(LOADING));
      const entries = this.entriesByKey.get(key);
      if (entries == null) this.entriesByKey.set(key, new Set([entry]));
      else entries.add(entry);
    });
    this.scheduleReconcile();
    return {
      close: () => this.remove(entry),
      status: (key) => this.statuses.get(key) ?? status.create(LOADING),
      onStatusChange: (statusHandler) => {
        entry.statusHandlers.add(statusHandler);
        return () => entry.statusHandlers.delete(statusHandler);
      },
      onDrop: (dropHandler) => {
        entry.dropHandlers.add(dropHandler);
        return () => entry.dropHandlers.delete(dropHandler);
      },
    };
  }

  /**
   * Resolves when the stream carries the keys and the Core has acked them, so the
   * stream delivers every sample written after it resolves.
   * @param signal - Rejects the wait with its reason when it aborts.
   * @throws {UnexpectedError} if the streamer closes first.
   * @throws the Core's rejection of a key, if the stream leaves the key off.
   */
  async waitAcked(keys: channel.Key[], signal?: AbortSignal): Promise<void> {
    if (this.connMu.closed) throw new UnexpectedError("telemetry streamer is closed");
    signal?.throwIfAborted();
    await new Promise<void>((resolve, reject) => {
      const abort = (): void => {
        if (this.ackWaiters.delete(waiter)) reject(errors.fromUnknown(signal?.reason));
      };
      const waiter: AckWaiter = {
        keys,
        resolve: () => {
          signal?.removeEventListener("abort", abort);
          resolve();
        },
        reject: (e) => {
          signal?.removeEventListener("abort", abort);
          reject(e);
        },
      };
      signal?.addEventListener("abort", abort);
      this.ackWaiters.add(waiter);
      this.settleAckWaiters();
    });
  }

  private setAcked(keys: Set<channel.Key>): void {
    this.acked = keys;
    this.props.store.setAcked(keys);
    this.settleAckWaiters();
  }

  private settleAckWaiters(): void {
    this.ackWaiters.forEach((w) => {
      const rejected = w.keys.find((k) => this.rejected.has(k));
      if (rejected != null) {
        this.ackWaiters.delete(w);
        return w.reject(this.rejected.get(rejected)!);
      }
      if (!w.keys.every((k) => this.acked.has(k))) return;
      this.ackWaiters.delete(w);
      w.resolve();
    });
  }

  async close(): Promise<void> {
    const { instrumentation: ins } = this.props;
    const closed = new UnexpectedError("telemetry streamer is closed");
    this.ackWaiters.forEach((w) => w.reject(closed));
    this.ackWaiters.clear();
    this.scheduleReconcile.cancel();
    this.removalTimers.forEach(clearTimeout);
    this.removalTimers.clear();
    this.connMu.closed = true;
    this.streamAbort?.abort();
    await this.connMu.runExclusive(async () => {
      try {
        await this.releaseStream();
      } catch (e) {
        ins.L.error("failed to close streamer", { error: e });
      }
    });
  }

  private async releaseStream(): Promise<void> {
    this.setAcked(new Set());
    const prev = this.streamer;
    const prevLoop = this.runLoop;
    this.streamer = null;
    this.runLoop = null;
    this.streamAbort = null;
    prev?.close();
    if (prevLoop != null) await prevLoop;
  }

  private remove(entry: Entry): void {
    if (entry.removed) return;
    entry.removed = true;
    const timer = setTimeout(() => {
      this.removalTimers.delete(timer);
      this.entries.delete(entry);
      entry.keys.forEach((key) => {
        const entries = this.entriesByKey.get(key)!;
        entries.delete(entry);
        if (entries.size === 0) this.entriesByKey.delete(key);
      });
      this.scheduleReconcile();
    }, this.props.removalDelay.milliseconds);
    this.removalTimers.add(timer);
  }

  // Retries skip the update interval, so narrowing a rejected key runs at the pace of
  // the Core.
  private reconcileNow(): void {
    this.scheduleReconcile();
    this.scheduleReconcile.flush();
  }

  /** @returns the union of keys across all entries, including those pending
   * removal, so the stream lags demand only by the removal delay. */
  private demandedKeys(): Set<channel.Key> {
    return new Set(this.entriesByKey.keys());
  }

  /** @returns the demanded keys the stream should carry now. */
  private desiredKeys(): Set<channel.Key> {
    const demanded = this.demandedKeys();
    this.rejected.forEach((_, k) => {
      if (!demanded.has(k)) this.rejected.delete(k);
    });
    this.deferred = this.deferred.filter((k) => demanded.has(k));
    return demanded.difference(this.rejected).difference(new Set(this.deferred));
  }

  private setStatus(key: channel.Key, crude: status.Crude): void {
    const prev = this.statuses.get(key);
    if (prev?.variant === crude.variant && prev?.message === crude.message) return;
    const next = status.create(crude);
    this.statuses.set(key, next);
    this.entriesByKey.get(key)?.forEach((e) => {
      if (e.removed) return;
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

  // Closes the open blocks of the keys, so reads fetch the span the stream will not
  // deliver.
  private flushBuffers(keys: Iterable<channel.Key>): void {
    const { joins, onWait } = this.props;
    for (const key of keys) onWait(joins.flush(key));
  }

  // Frames may drop until a stream opens again.
  private drop(): void {
    this.outdated = true;
    this.entries.forEach((e) => {
      if (e.removed) return;
      e.dropHandlers.forEach((handler) => {
        try {
          handler();
        } catch (err) {
          this.props.instrumentation.L.error(
            "drop handler failed",
            { error: err },
            true,
          );
        }
      });
    });
  }

  // Drops every cached block once a stream opens after a drop, since the drop can miss
  // history changes that outdate them.
  private refresh(): void {
    if (!this.outdated) return;
    this.outdated = false;
    this.props.onWait(this.props.joins.invalidate());
  }

  private async reconcile(): Promise<void> {
    const { instrumentation: ins } = this.props;
    let retry = false;
    await this.connMu.runExclusive(async () => {
      if (this.connMu.closed) return;
      const desired = this.desiredKeys();
      const arrKeys = Array.from(desired);
      const fresh = arrKeys.filter((k) => !this.sentKeys.has(k));
      try {
        if (desired.size === 0) {
          const removed = this.sentKeys;
          this.sentKeys = new Set();
          this.pruneStatuses();
          await this.releaseStream();
          this.flushBuffers(removed);
          this.breaker.reset();
          ins.L.info("streamer closed, no keys to stream");
          this.releaseDeferred();
          return;
        }
        if (
          this.streamer != null &&
          desired.size === this.sentKeys.size &&
          desired.isSubsetOf(this.sentKeys)
        )
          return this.releaseDeferred();
        const removed = Array.from(this.sentKeys).filter((k) => !desired.has(k));
        if (this.streamer == null) {
          ins.L.info("creating new streamer", { keys: arrKeys });
          this.streamAbort = new AbortController();
          const streamer = await this.props.openStreamer(
            { channels: arrKeys, throttleRate: THROTTLE_RATE },
            {
              signal: this.streamAbort.signal,
              // The reopened stream acked its keys when it opened.
              onReopen: () => {
                this.refresh();
                this.setAcked(new Set(this.sentKeys));
                this.sentKeys.forEach((key) => this.setStatus(key, STREAMING));
              },
              onDrop: () => {
                this.setAcked(new Set());
                this.flushBuffers(this.sentKeys);
                this.drop();
                this.sentKeys.forEach((key) => this.setStatus(key, RECONNECTING));
              },
            },
          );
          this.streamer = streamer;
          this.refresh();
          this.runLoop = this.run(streamer);
        } else {
          ins.L.debug("updating streamer", () => ({
            prev: Array.from(this.sentKeys),
            next: arrKeys,
          }));
          await this.streamer.update(arrKeys, { waitForAck: true });
        }
        this.sentKeys = desired;
        this.setAcked(new Set(desired));
        this.flushBuffers(removed);
        // Existing keys keep their status so a demand change cannot clear a per-key
        // error.
        fresh.forEach((key) => this.setStatus(key, STREAMING));
        this.pruneStatuses();
        this.breaker.reset();
        this.releaseDeferred();
      } catch (e) {
        ins.L.error("failed to update streamer", { error: e });
        const failure = status.fromException(e, "Failed to stream channel");
        if (isDefinitive(e) && !this.connMu.closed && fresh.length > 0) {
          // One of the new keys caused the rejection. Retry with one of them, and
          // return the rest one at a time, until the culprit fails alone.
          if (fresh.length === 1) {
            this.rejected.set(fresh[0], errors.fromUnknown(e));
            this.setStatus(fresh[0], failure);
            this.settleAckWaiters();
            this.releaseDeferred();
          } else {
            this.deferred.push(...fresh.slice(1));
            this.reconcileNow();
          }
          return;
        }
        desired.forEach((key) => {
          if (!this.sentKeys.has(key)) this.setStatus(key, failure);
        });
        retry = true;
      }
    });
    if (!retry || this.connMu.closed) return;
    if (await this.breaker.wait()) this.reconcileNow();
  }

  // Returns one held-back key to the stream. Every reconcile that does not defer keys
  // calls it, so narrowing a rejection always runs to completion.
  private releaseDeferred(): void {
    if (this.deferred.length === 0) return;
    this.deferred.shift();
    this.reconcileNow();
  }

  // Keeps statuses for demanded keys, so a rejected key keeps its error.
  private pruneStatuses(): void {
    const demanded = this.demandedKeys();
    this.statuses.forEach((_, key) => {
      if (!demanded.has(key)) this.statuses.delete(key);
    });
  }

  private async run(streamer: Streamer): Promise<void> {
    const { joins, onWait, instrumentation: ins } = this.props;
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
        onWait(joins.write(grouped));
        grouped.forEach((_, k) => this.setStatus(k, STREAMING));
        if (grouped.size !== 0)
          this.entries.forEach((e) => {
            if (e.removed) return;
            try {
              e.handler(grouped);
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
        this.streamAbort = null;
        this.setAcked(new Set());
        this.flushBuffers(this.sentKeys);
        this.drop();
        this.sentKeys = new Set();
        this.reconcileNow();
      }
    }
  }
}
