// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type breaker,
  type destructor,
  errors,
  observe,
  type record,
  type state,
  type TimeStamp,
} from "@synnaxlabs/x";
import type z from "zod";

import { AccessDeniedError } from "@/errors";
import type { framer } from "@/framer";
import { bindDerived, type DeriveParams } from "@/query/derived";
import { list } from "@/query/list";
import { isOffline } from "@/query/offline";
import { space } from "@/query/space";
import { createStreamer, type Listener, type Streamer } from "@/query/streamer";
import { type Keyed, type ListenerSpec, Table, type TableParams } from "@/query/table";
import type { Data, Params } from "@/query/types";

export interface CacheParams {
  /**
   * Opens the raw frame streamer used to receive change signals; reconnect hardening is
   * applied internally. Null constructs a detached cache: purely local tables and no
   * change stream. Used for clients constructed with `cache: false`.
   */
  openStreamer: framer.StreamOpener | null;
  /** Retry behavior for change-stream reconnect attempts. */
  breaker?: breaker.Config;
  /** Called whenever the change stream goes live, before reconciliation begins. */
  onStreamLive?: () => void;
  /** Called when the change stream fails and reconnection begins. */
  onStreamDrop?: (error: Error) => void;
  /**
   * Called when the change stream is refused because the caller lacks access to
   * the mirrored channels. Only a policy change lifts a denial.
   */
  onStreamDenied?: (error: Error) => void;
  /**
   * Receives each failed change-stream reconnect attempt. Defaults to console.warn.
   */
  onStreamRetry?: (error: Error) => void;
  /**
   * Receives errors that have no caller to throw to: listener fan-out, streamer frame
   * handling, and background reconciliation. Defaults to console logging.
   */
  onError?: (error: Error) => void;
  /**
   * The most records a table holds while the change stream is live. Past it, the table
   * drops the least recently read records that no answer, list, or keyed read needs.
   * Defaults to {@link DEFAULT_RECORD_CAP}.
   */
  recordCap?: number;
  /**
   * The clock that the teardown grace of answers and lists reads. Defaults to the
   * system clock.
   */
  clock?: () => TimeStamp;
}

/** The most records a table holds by default. */
const DEFAULT_RECORD_CAP = 10_000;

// A trim drops records down to this share of the cap, so the next few writes do not
// trim again.
const TRIM_FLOOR = 0.9;

/** Configuration for a derived table owned by a {@link Cache}. */
export interface DerivedConfig<
  Key extends record.Key,
  Value extends Keyed<Key>,
  Composed extends Keyed<Key>,
> extends DeriveParams<Key, Value, Composed> {
  /** Names the table in diagnostics. */
  name: string;
  /** Overrides the deep-equality default used to silence redundant sets. */
  equal?: (a: Composed, b: Composed, key: Key) => boolean;
}

/** Configuration for a table owned by a {@link Cache}. */
export interface TableConfig<
  Key extends record.Key = record.Key,
  Value extends state.State = state.State,
> extends Omit<TableParams<Key, Value>, "onError"> {
  /** Names the table in diagnostics (reconciliation errors). */
  name: string;
  /** Mirror listeners keeping this table current from stream channels. */
  listen?: Array<ListenerSpec<Key, Value>>;
}

interface RegisteredTable {
  name: string;
  /** Drops the records past cap. A derived table drops what its source drops. */
  trim?: (cap: number) => void;
  listeners: Listener[];
  reconcile?: () => Promise<void>;
  reset: () => void;
  clearPendingDeletes: () => void;
  close: () => void;
}

/** The cache-facing lifecycle of an answer space or of lists, generics erased. */
interface SpaceLifecycle {
  close: () => void;
  reset: () => void;
}

/**
 * The client's local mirror of cluster state: keyed tables, the change-stream loop,
 * connection epochs, and reconciliation. Holds zero domain knowledge — domain clients
 * create their tables here and expose typed query spaces. Always present on a client; a
 * null {@link CacheParams.openStreamer} stands in for "disabled" or "not yet connected"
 * rather than a null cache. The change stream opens lazily: nothing touches the network
 * until the first {@link ensureStreaming} call. All tables and listeners must be
 * declared before streaming starts.
 */
export class Cache {
  private readonly tables: RegisteredTable[] = [];
  private readonly reactions: Listener[] = [];
  private readonly spaces: SpaceLifecycle[] = [];
  private readonly detachers: destructor.Destructor[] = [];
  private readonly epochObserver = new observe.Observer<number>();
  private readonly downObserver = new observe.Observer<void>();
  private readonly missedObserver = new observe.Observer<void>();
  private readonly params: CacheParams;
  private streamer: Streamer | null = null;
  private epochCount = 0;
  private reconciling: Promise<void> | null = null;
  private reconcileQueued = false;
  private unmaintained = false;
  /** Whether the change stream is live, so tables may drop records. */
  private _live = false;
  private trimScheduled = false;

  /**
   * Stable error sink for machinery built on this cache (dispatch
   * controllers, answer spaces).
   */
  readonly onError: (error: Error) => void;

  constructor(params: CacheParams) {
    this.params = params;
    this.onError = params.onError ?? console.error;
  }

  /**
   * Creates a table owned by this cache and returns it, binding its declared mirror
   * listeners. Must be called before streaming starts; creating after {@link
   * ensureStreaming} throws, as the new table's channels would never be streamed.
   */
  createTable<Key extends record.Key, Value extends state.State>(
    config: TableConfig<Key, Value>,
  ): Table<Key, Value> {
    if (this.streamer != null)
      throw new Error(`cannot create table ${config.name} after streaming has started`);
    const { name, listen, ...params } = config;
    const table = new Table<Key, Value>({ ...params, onError: this.onError });
    this.register(table);
    this.tables.push({
      name,
      trim: (cap) => {
        if (table.size > cap) table.trim(Math.floor(cap * TRIM_FLOOR));
      },
      listeners: (listen ?? []).map((spec) => spec.bind(table)),
      reconcile: config.fetch == null ? undefined : async () => await table.reconcile(),
      reset: () => table.reset(),
      clearPendingDeletes: () => table.clearPendingDeletes(),
      close: () => table.close(),
    });
    return table;
  }

  /**
   * Creates a table materialized from another table owned by this cache. Composed
   * entries are replaced on source and trigger events, so entry identity changes
   * exactly when composition changes; misses on read fetch through the source. It holds
   * an entry for each record that its source keeps through a trim. Resets with the
   * cache. No stream listeners of its own, so it may be created at any time.
   */
  derive<Key extends record.Key, Value extends Keyed<Key>, Composed extends Keyed<Key>>(
    config: DerivedConfig<Key, Value, Composed>,
  ): Table<Key, Composed> {
    const { name, source, compose, triggers, equal } = config;
    const table = new Table<Key, Composed>({
      onError: this.onError,
      equal,
      fetch: async (keys) => (await source.retrieve(keys)).map(compose),
    });
    this.detachers.push(bindDerived(table, { source, compose, triggers }));
    this.tables.push({
      name,
      listeners: [],
      reset: () => table.reset(),
      clearPendingDeletes: () => table.clearPendingDeletes(),
      close: () => table.close(),
    });
    return table;
  }

  /** Trims the cache after each write that grows the table past the cap. */
  private register<Key extends record.Key, Value extends state.State>(
    table: Table<Key, Value>,
  ): void {
    const cap = this.params.recordCap ?? DEFAULT_RECORD_CAP;
    this.detachers.push(
      table.subscribeBatch(() => {
        if (table.size > cap) this.scheduleTrim();
      }),
    );
  }

  private scheduleTrim(): void {
    if (this.trimScheduled) return;
    this.trimScheduled = true;
    queueMicrotask(() => {
      this.trimScheduled = false;
      this.trim();
    });
  }

  /**
   * Drops records past the cap from every table. Nothing drops while the change stream
   * is down, so a view reopened offline still shows what it showed.
   */
  private trim(): void {
    if (!this._live) return;
    const cap = this.params.recordCap ?? DEFAULT_RECORD_CAP;
    this.tables.forEach(({ trim }) => trim?.(cap));
  }

  /**
   * Registers a channel reaction: a listener that is not a table mirror
   * (dispatch wires, cross-table side effects). Same pre-streaming
   * constraint as {@link createTable}.
   */
  listen<Z extends z.ZodType>(listener: Listener<Z>): void {
    if (this.streamer != null)
      throw new Error(
        `cannot listen to ${listener.channel} after streaming has started`,
      );
    this.reactions.push(listener);
  }

  /**
   * Constructs an answer space wired to this cache: reads open the change
   * stream, maintained answers refetch on epoch bumps, maintenance errors
   * report to the cache's error sink, and the space closes with the cache.
   */
  queries<
    Q extends Params,
    D extends Data,
    K extends record.Key = record.Key,
    V extends state.State = state.State,
  >(config: space.Config<Q, D, K, V>): space.Space<Q, D, K, V> {
    const answers = new space.Space(config, this.spaceHooks());
    this.spaces.push(answers);
    return answers;
  }

  /**
   * Constructs the lists of a table, wired to this cache as {@link queries} wires an
   * answer space.
   */
  lists<
    Q extends list.Query<O, C> & { readonly [key: string]: Params },
    D extends Data,
    K extends record.Key,
    V extends Keyed<K>,
    C,
    O,
  >(config: list.Config<Q, D, K, V, C, O>): list.Lists<Q, D, K, V, C, O> {
    const lists = new list.Lists(config, this.spaceHooks());
    this.spaces.push(lists);
    return lists;
  }

  private spaceHooks(): space.Hooks {
    return {
      ensureStreaming: async () => await this.ensureStreaming(),
      onEpoch: (callback) => this.onEpoch(callback),
      onMissed: (callback) => this.onMissed(callback),
      epoch: () => this.epochCount,
      live: () => this._live,
      maintained: () => !this.unmaintained && this.params.openStreamer != null,
      onError: this.onError,
      clock: this.params.clock,
    };
  }

  /**
   * Ensures the change stream is open, opening it on first call. Each open reconciles
   * the records cached before it, so a caller need not await this before a fetch.
   * @throws {AccessDeniedError} if the caller cannot stream the mirrored
   * channels. Answers stop being maintained until a later call succeeds.
   */
  async ensureStreaming(): Promise<void> {
    const {
      openStreamer,
      breaker,
      onStreamLive,
      onStreamDrop,
      onStreamRetry,
      onStreamDenied,
    } = this.params;
    if (openStreamer == null) return;
    if (this.streamer == null) {
      // A reset can retire the streamer while its open is in flight; a
      // retired streamer's lifecycle callbacks must not touch the epoch.
      const streamer = createStreamer({
        openStreamer,
        breaker,
        listeners: [
          ...this.tables.flatMap(({ listeners }) => listeners),
          ...this.reactions,
        ],
        onError: this.onError,
        onLive: onStreamLive,
        onDrop: (error) => {
          if (this.streamer === streamer) this.goDown();
          onStreamDrop?.(error);
        },
        onRetry: onStreamRetry,
        // Both run before the stream's first frame.
        onOpen: () => {
          if (this.streamer !== streamer) return;
          this.unmaintained = false;
          this.goLive();
        },
        onReopen: () => {
          if (this.streamer === streamer) this.goLive();
        },
        // The stream stays open, so the epoch holds: changes sent after the drop
        // still arrive.
        onMissed: () => {
          if (this.streamer !== streamer) return;
          this.scheduleReconcile();
          this.missedObserver.notify();
        },
        onDead: (error) => {
          if (this.streamer !== streamer) return;
          this.unmaintained = true;
          this.goDown();
          if (AccessDeniedError.matches(error)) onStreamDenied?.(error);
        },
      });
      this.streamer = streamer;
    }
    try {
      await this.streamer.demand();
    } catch (exc) {
      this.unmaintained = true;
      const error = errors.fromUnknown(exc);
      // The first open never reaches onDead: the stream dies before there is
      // one to observe.
      if (AccessDeniedError.matches(error)) onStreamDenied?.(error);
      throw error;
    }
  }

  /**
   * The connection epoch: 0 before streaming starts, incremented each time the stream
   * goes live. Changes may have been missed between epochs.
   */
  get epoch(): number {
    return this.epochCount;
  }

  /**
   * Whether the change stream is live. While it is, each change written to a streamed
   * channel reaches the listeners, and {@link onStreamDown} fires once it stops.
   */
  get live(): boolean {
    return this._live;
  }

  /** Subscribes to epoch changes. Returns a destructor that unsubscribes. */
  onEpoch(callback: (epoch: number) => void): destructor.Destructor {
    return this.epochObserver.onChange(callback);
  }

  /**
   * Subscribes to misses: the Core dropped changes for the stream because it fell
   * behind. The stream stays open and the epoch holds, so changes sent after the drop
   * still arrive. Returns a destructor that unsubscribes.
   */
  onMissed(callback: () => void): destructor.Destructor {
    return this.missedObserver.onChange(callback);
  }

  /**
   * Subscribes to the change stream going down: a drop that starts a reconnect, a
   * failure that ends it, a reset, or a close. No change made while it is down ever
   * arrives. Returns a destructor that unsubscribes.
   */
  onStreamDown(callback: () => void): destructor.Destructor {
    return this.downObserver.onChange(callback);
  }

  private goDown(): void {
    this._live = false;
    this.downObserver.notify();
  }

  /**
   * Re-checks every cached entry against the cluster: refreshes entries that
   * still exist and tombstones entries that vanished. Runs each time the stream goes
   * live and after every miss, for tables that declare a fetch.
   */
  private async reconcile(): Promise<void> {
    await Promise.all(
      this.tables.map(async ({ name, reconcile }) => {
        if (reconcile == null) return;
        try {
          await reconcile();
        } catch (exc) {
          if (isOffline(exc)) return;
          this.onError(new Error(`failed to reconcile ${name} cache`, { cause: exc }));
        }
      }),
    );
  }

  /**
   * Discards every cached entry, tombstone, and answer, closes the change
   * stream so it reopens fresh, and returns the epoch to 0. Called when the
   * cluster behind the connection is replaced: no cached record, corpse, or
   * answer remains meaningful, so nothing is tombstoned or diffed.
   */
  async reset(): Promise<void> {
    const { streamer } = this;
    this.streamer = null;
    this.goDown();
    // Drain the retired stream first so a late frame cannot repopulate the
    // tables cleared below. A close failure must not block the reset: the
    // stream is retired either way.
    try {
      await streamer?.close();
    } catch (exc) {
      console.error("failed to close retired stream", exc);
    }
    this.tables.forEach(({ reset }) => reset());
    this.spaces.forEach((lifecycle) => lifecycle.reset());
    this.unmaintained = false;
    this.epochCount = 0;
    this.epochObserver.notify(0);
  }

  private clearPendingDeletes(): void {
    this.tables.forEach(({ clearPendingDeletes }) => clearPendingDeletes());
  }

  /**
   * Bumps the epoch and reconciles every table. Records fetched before the stream went
   * live, or while it was down, may have missed a change.
   */
  private goLive(): void {
    this.clearPendingDeletes();
    this._live = true;
    this.epochCount++;
    this.epochObserver.notify(this.epochCount);
    this.scheduleReconcile();
    this.trim();
  }

  // Runs one reconcile at a time. A request during a reconcile runs one more after it,
  // since the running one can have fetched before the change that the request reports.
  private scheduleReconcile(): void {
    if (this.reconciling != null) {
      this.reconcileQueued = true;
      return;
    }
    this.reconciling = this.reconcile()
      .catch((exc: unknown) =>
        this.onError(new Error("failed to reconcile caches", { cause: exc })),
      )
      .finally(() => {
        this.reconciling = null;
        if (!this.reconcileQueued) return;
        this.reconcileQueued = false;
        this.scheduleReconcile();
      });
  }

  /**
   * Closes the change stream and every answer space this cache constructed, and rejects
   * table fetches in progress. A no-op for the stream when streaming never started.
   */
  async close(): Promise<void> {
    this.detachers.forEach((detach) => detach());
    this.spaces.forEach((lifecycle) => lifecycle.close());
    this.tables.forEach(({ close }) => close());
    if (this.streamer == null) return;
    this.goDown();
    await this.streamer.close();
  }
}
