// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type destructor,
  errors,
  observe,
  type record,
  type state,
} from "@synnaxlabs/x";
import type z from "zod";

import { NotFoundError } from "@/errors";
import { Queries, type QueriesParams, type Retrieves } from "@/query/query";
import {
  createStreamer,
  type Listener,
  type Streamer,
  type StreamOpener,
} from "@/query/streamer";
import { type ListenerSpec, Table, type TableParams } from "@/query/table";
import { type Data, type Params } from "@/query/types";

export interface CacheParams {
  /**
   * Opens the frame streamer used to receive change signals. Null constructs
   * a detached cache: purely local tables and no change stream. Used for
   * clients constructed with `cache: false`.
   */
  openStreamer: StreamOpener | null;
  /**
   * Receives errors that have no caller to throw to: listener fan-out,
   * streamer frame handling, and background reconciliation. Defaults to
   * console logging.
   */
  onError?: (error: Error) => void;
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

interface TableEntry {
  name: string;
  listeners: Listener[];
  reconcile?: () => Promise<void>;
  reset: () => void;
}

/** The cache-facing lifecycle of an answer space, generics erased. */
interface Space {
  close: () => void;
  reset: () => void;
}

/**
 * Retrieves by key are strict: a batch containing any vanished key rejects
 * with NotFound instead of returning the survivors. Falls back to probing
 * keys one at a time so survivors are still distinguishable.
 */
const fetchSurvivors = async <Key extends record.Key, Value extends state.State>(
  fetch: NonNullable<TableParams<Key, Value>["fetch"]>,
  keys: Key[],
): Promise<Value[]> => {
  try {
    return await fetch(keys);
  } catch (exc) {
    if (!NotFoundError.matches(exc)) throw errors.fromUnknown(exc);
  }
  if (keys.length === 1) return [];
  const probed = await Promise.all(
    keys.map(async (key) => {
      try {
        return await fetch([key]);
      } catch (exc) {
        if (NotFoundError.matches(exc)) return [];
        throw errors.fromUnknown(exc);
      }
    }),
  );
  return probed.flat();
};

/**
 * Binds a table's reconcile pass: refreshes rows that still exist on the
 * cluster and tombstones rows that vanished.
 */
const bindReconcile =
  <Key extends record.Key, Value extends state.State>(
    table: Table<Key, Value>,
    fetch: NonNullable<TableParams<Key, Value>["fetch"]>,
  ): (() => Promise<void>) =>
  async () => {
    const keys = table.keys();
    if (keys.length === 0) return;
    const values = await fetchSurvivors(fetch, keys);
    const present = new Set<Key>(values.map(({ key }) => key));
    const vanished = keys.filter((k) => !present.has(k));
    if (vanished.length > 0) table.delete(vanished);
    if (values.length > 0) table.set(values);
  };

/**
 * The client's local mirror of cluster state: keyed tables, the change-stream
 * loop, connection epochs, and reconciliation. Holds zero domain knowledge —
 * domain clients create their tables here and expose typed query spaces.
 * Always present on a client; a null {@link CacheParams.openStreamer} stands
 * in for "disabled" or "not yet connected" rather than a null cache.
 *
 * The change stream opens lazily: nothing touches the network until the
 * first {@link ensureStreaming} call. All tables and listeners must be
 * declared before streaming starts.
 */
export class Cache {
  private readonly entries: TableEntry[] = [];
  private readonly reactions: Listener[] = [];
  private readonly spaces: Space[] = [];
  private readonly epochObserver = new observe.Observer<number>();
  private readonly openStreamer: StreamOpener | null;
  private streamer: Streamer | null = null;
  private epochCount = 0;

  /**
   * Stable error sink for machinery built on this cache (dispatch
   * controllers, answer spaces).
   */
  readonly onError: (error: Error) => void;

  constructor({ openStreamer, onError = console.error }: CacheParams) {
    this.openStreamer = openStreamer;
    this.onError = onError;
  }

  /**
   * Creates a table owned by this cache and returns it, binding its declared
   * mirror listeners. Must be called before streaming starts; creating after
   * {@link ensureStreaming} throws, as the new table's channels would never
   * be streamed.
   */
  createTable<Key extends record.Key, Value extends state.State>(
    config: TableConfig<Key, Value>,
  ): Table<Key, Value> {
    if (this.streamer != null)
      throw new Error(`cannot create table ${config.name} after streaming has started`);
    const { name, listen, ...params } = config;
    const table = new Table<Key, Value>({ ...params, onError: this.onError });
    this.entries.push({
      name,
      listeners: (listen ?? []).map((spec) => spec.bind(table)),
      reconcile: config.fetch == null ? undefined : bindReconcile(table, config.fetch),
      reset: () => table.reset(),
    });
    return table;
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
  >(params: QueriesParams<Q, D, K, V>): Retrieves<Q, D> {
    const space = new Queries(params, {
      ensureStreaming: async () => await this.ensureStreaming(),
      onEpoch: (callback) => this.onEpoch(callback),
      onError: this.onError,
    });
    this.spaces.push(space);
    return space;
  }

  /**
   * Ensures the change stream is open, opening it on first call. Callers that
   * populate or read tables await this first so no change is missed between a
   * fetch and the stream opening.
   */
  async ensureStreaming(): Promise<void> {
    const { openStreamer } = this;
    if (openStreamer == null) return;
    if (this.streamer == null) {
      // A reset can retire the streamer while its open is in flight; a
      // retired streamer's lifecycle callbacks must not touch the epoch.
      const streamer = createStreamer({
        openStreamer,
        listeners: [
          ...this.entries.flatMap(({ listeners }) => listeners),
          ...this.reactions,
        ],
        onError: this.onError,
        onOpen: () => {
          if (this.streamer !== streamer) return;
          this.epochCount = 1;
          this.epochObserver.notify(this.epochCount);
        },
        onReopen: () => {
          if (this.streamer !== streamer) return;
          this.bumpEpoch();
        },
      });
      this.streamer = streamer;
    }
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
   * still exist and tombstones rows that vanished. Runs after every
   * reconnect for tables that declare a fetch.
   */
  private async reconcile(): Promise<void> {
    await Promise.all(
      this.entries.map(async ({ name, reconcile }) => {
        if (reconcile == null) return;
        try {
          await reconcile();
        } catch (exc) {
          this.onError(new Error(`failed to reconcile ${name} cache`, { cause: exc }));
        }
      }),
    );
  }

  /**
   * Discards every cached row, tombstone, and answer, closes the change
   * stream so it reopens fresh, and returns the epoch to 0. Called when the
   * cluster behind the connection is replaced: no cached record, corpse, or
   * answer remains meaningful, so nothing is tombstoned or diffed.
   */
  async reset(): Promise<void> {
    const { streamer } = this;
    this.streamer = null;
    // Drain the retired stream first so a late frame cannot repopulate the
    // tables cleared below. A close failure must not block the reset: the
    // stream is retired either way.
    try {
      await streamer?.close();
    } catch (exc) {
      console.error("failed to close retired stream", exc);
    }
    this.entries.forEach(({ reset }) => reset());
    this.spaces.forEach((space) => space.reset());
    this.epochCount = 0;
    this.epochObserver.notify(0);
  }

  private bumpEpoch(): void {
    this.epochCount++;
    this.epochObserver.notify(this.epochCount);
    this.reconcile().catch((exc: unknown) =>
      this.onError(
        new Error("failed to reconcile caches after reconnect", { cause: exc }),
      ),
    );
  }

  /**
   * Closes the change stream and every answer space this cache constructed.
   * A no-op for the stream when streaming never started.
   */
  async close(): Promise<void> {
    this.spaces.forEach((space) => space.close());
    if (this.streamer == null) return;
    await this.streamer.close();
  }
}
