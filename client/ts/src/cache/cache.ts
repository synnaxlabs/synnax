// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor, observe, type record, type state } from "@synnaxlabs/x";

import { Answers, type AnswersParams } from "@/cache/answers";
import { createStreamer, type Streamer, type StreamOpener } from "@/cache/streamer";
import {
  type ChannelListener,
  Table,
  type TableConfig,
  type TableConfigs,
  type Tables,
} from "@/cache/table";
import { type Data, type Query } from "@/cache/types";

export interface CacheParams {
  /**
   * Opens the frame streamer used to receive change signals. Null constructs
   * a detached cache: purely local tables, no change stream ever opens, and
   * unknown table keys auto-register empty configs on first access. Used for
   * UI layers that need working tables before a cluster connection exists,
   * and for clients constructed with `cache: false`.
   */
  openStreamer: StreamOpener | null;
  /**
   * Receives errors that have no caller to throw to: listener fan-out,
   * streamer frame handling, and background reconciliation. Defaults to
   * console logging.
   */
  onInternalError?: (error: Error) => void;
}

/**
 * The client's local mirror of cluster state: keyed tables, the change-stream
 * loop, connection epochs, and reconciliation. Holds zero domain knowledge —
 * domain clients register their table configs and expose typed query spaces.
 * Always present on a client; detached (see {@link CacheParams.openStreamer})
 * stands in for "disabled" or "not yet connected" rather than a null cache.
 *
 * The change stream opens lazily: nothing touches the network until the
 * first {@link ensureStreaming} call. All tables must be registered before
 * streaming starts.
 */
export class Cache {
  private readonly configs: TableConfigs<any> = {};
  private readonly tables: Tables = {};
  private readonly epochObserver = new observe.Observer<number>();
  private readonly onInternalError: (error: Error) => void;
  private readonly openStreamer: StreamOpener | null;
  private streamer: Streamer | null = null;
  private epochCount = 0;

  constructor({ openStreamer, onInternalError }: CacheParams) {
    this.openStreamer = openStreamer;
    this.onInternalError = onInternalError ?? ((error) => console.error(error));
  }

  /**
   * Stable error sink for machinery built on this cache (dispatch
   * controllers, answer spaces).
   */
  get onError(): (error: Error) => void {
    return (error) => this.onInternalError(error);
  }

  /**
   * Registers a table under the given key. Must be called before streaming
   * starts; registering after {@link ensureStreaming} throws, as the new
   * table's channels would never be streamed.
   */
  registerTable<Key extends record.Key, Value extends state.State>(
    key: string,
    config: TableConfig<any, Key, Value>,
  ): void {
    if (this.streamer != null)
      throw new Error(`cannot register table ${key} after streaming has started`);
    if (key in this.configs) throw new Error(`table ${key} is already registered`);
    this.configs[key] = config;
    this.tables[key] = new Table<Key, Value>(this.onError, config.equal);
  }

  /**
   * Registers additional channel listeners on an already-registered table.
   * Same pre-streaming constraint as {@link registerTable}. Used by machinery
   * (dispatch controllers) that needs the table to exist before it can build
   * its listener.
   */
  addListeners(key: string, ...listeners: ChannelListener<any>[]): void {
    if (this.streamer != null)
      throw new Error(`cannot add listeners to ${key} after streaming has started`);
    const config = this.configs[key];
    if (config == null) throw new Error(`table ${key} is not registered`);
    config.listeners.push(...listeners);
  }

  /** True when the cache has no stream source and tables are purely local. */
  get detached(): boolean {
    return this.openStreamer == null;
  }

  // Detached caches have no domain clients to register configs, so unknown
  // keys become empty local tables on first access.
  private resolve(key: string): Table<any, any> | null {
    const table = this.tables[key];
    if (table != null || !this.detached) return table ?? null;
    this.registerTable(key, { listeners: [] });
    return this.tables[key];
  }

  /**
   * Returns the table registered under the given key. Throws when the key was
   * never registered.
   */
  table<Key extends record.Key, Value extends state.State>(
    key: string,
  ): Table<Key, Value> {
    const table = this.resolve(key);
    if (table == null) throw new Error(`table ${key} is not registered`);
    return table as Table<Key, Value>;
  }

  /**
   * Constructs an answer space wired to this cache: reads open the change
   * stream, maintained answers refetch on epoch bumps, and maintenance errors
   * report to the cache's error sink.
   */
  answers<
    Q extends Query,
    D extends Data,
    K extends record.Key = record.Key,
    V extends state.State = state.State,
  >(params: AnswersParams<Q, D, K, V>): Answers<Q, D, K, V> {
    return new Answers(params, {
      ensureStreaming: async () => await this.ensureStreaming(),
      onEpoch: (callback) => this.onEpoch(callback),
      onError: this.onError,
    });
  }

  /**
   * Ensures the change stream is open, opening it on first call. Callers that
   * populate or read tables await this first so no change is missed between a
   * fetch and the stream opening.
   */
  async ensureStreaming(): Promise<void> {
    const opener = this.openStreamer;
    if (opener == null) return;
    this.streamer ??= createStreamer({
      openStreamer: opener,
      configs: this.configs,
      onError: this.onError,
      tables: this.tables,
      onOpen: () => {
        this.epochCount = 1;
        this.epochObserver.notify(this.epochCount);
      },
      onReopen: () => this.bumpEpoch(),
    });
    await this.streamer.demand();
  }

  /**
   * The connection epoch: 0 before streaming starts, 1 once live, incremented
   * on every reconnect. Changes may have been missed between epochs.
   */
  get epoch(): number {
    return this.epochCount;
  }

  /** Subscribes to epoch changes. Returns a destructor that unsubscribes. */
  onEpoch(callback: (epoch: number) => void): destructor.Destructor {
    return this.epochObserver.onChange(callback);
  }

  /**
   * Re-checks every cached row against the cluster: refreshes rows that
   * still exist and tombstones rows that vanished. Runs automatically
   * after every reconnect for tables whose config provides a refetch.
   */
  async reconcile(): Promise<void> {
    await Promise.all(
      Object.entries(this.configs).map(async ([key, config]) => {
        const { refetch } = config;
        if (refetch == null) return;
        const table = this.tables[key];
        const keys = table.list().map((value: record.Keyed<record.Key>) => value.key);
        if (keys.length === 0) return;
        try {
          const values = await refetch(keys);
          const present = new Set(values.map(({ key }) => key));
          const vanished = keys.filter((k) => !present.has(k));
          if (vanished.length > 0) table.delete(vanished);
          if (values.length > 0) table.setMany(values);
        } catch (exc) {
          this.onInternalError(
            new Error(`failed to reconcile ${key} cache`, { cause: exc }),
          );
        }
      }),
    );
  }

  private bumpEpoch(): void {
    this.epochCount++;
    this.epochObserver.notify(this.epochCount);
    this.reconcile().catch((exc: unknown) =>
      this.onInternalError(
        new Error("failed to reconcile caches after reconnect", { cause: exc }),
      ),
    );
  }

  /** Closes the change stream. A no-op when streaming never started. */
  async close(): Promise<void> {
    if (this.streamer == null) return;
    await this.streamer.close();
  }
}
