// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  array,
  type CrudeTimeSpan,
  debounce,
  deep,
  destructor,
  errors,
  type record,
  state,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import type z from "zod";

import { NotFoundError } from "@/errors";
import { Deleted } from "@/query/deleted";
import { type LookupIndex } from "@/query/indexes";
import { type Listener } from "@/query/streamer";

/**
 * Presence of a key in a table: "present" when a live entry exists, "tombstoned"
 * when the entry was deleted and its corpse is retained, "unknown" when the
 * table has never seen the key or the corpse was cleared.
 */
export type EntryStatus = "present" | "tombstoned" | "unknown";

/** A change to a single entry of a {@link Table}. */
export type TableEvent<
  Key extends record.Key = record.Key,
  Value extends state.State = state.State,
> = { variant: "set"; key: Key; value: Value } | { variant: "delete"; key: Key };

/** Receives {@link TableEvent}s. Must be synchronous; thrown errors are
 *  reported to the table's error sink and do not stop delivery. */
export interface TableSubscriber<
  Key extends record.Key = record.Key,
  Value extends state.State = state.State,
> {
  (event: TableEvent<Key, Value>): void;
}

/**
 * Receives every event of a batched write as one call, in event order. A single write
 * delivers a batch of one. Same synchrony and error contract as
 * {@link TableSubscriber}.
 */
export interface BatchSubscriber<
  Key extends record.Key = record.Key,
  Value extends state.State = state.State,
> {
  (events: TableEvent<Key, Value>[]): void;
}

/**
 * How fetched records hydrate a table: "set" overwrites entries, "if-absent"
 * preserves existing entries.
 */
export type HydrateMode = "set" | "if-absent";

/** A table value carrying its entry key, so batch writes and fetches can derive
 *  each entry's key from the record itself. */
export type Keyed<
  Key extends record.Key = record.Key,
  Value extends state.State = state.State,
> = Value & record.Keyed<Key>;

/** Construction arguments for a {@link Table}. */
export interface TableParams<
  Key extends record.Key = record.Key,
  Value extends state.State = state.State,
> {
  /** Receives subscriber and fetch errors that have no caller to throw to. */
  onError: (error: Error) => void;
  /** Overrides the deep-equality default used to silence redundant sets. */
  equal?: (a: Value, b: Value, key: Key) => boolean;
  /**
   * Fetches the current records for the given keys from the cluster,
   * returning only the entries that still exist. Powers {@link Table.retrieve},
   * key-announce listeners, and reconciliation. Omit for tables with no
   * server backing (they are skipped by all three).
   */
  fetch?: (keys: Key[]) => Promise<Array<Keyed<Key, Value>>>;
  /**
   * Hydration mode, "set" by default. Dispatch-backed document tables use
   * "if-absent" so fetches never clobber locally replayed edits.
   */
  hydrate?: HydrateMode;
  /**
   * Window for coalescing concurrent miss fetches into one fetch call.
   * @default TimeSpan.milliseconds(10)
   */
  fetchDebounce?: CrudeTimeSpan;
  /**
   * Secondary indexes over the table's live entries. The table keeps each current
   * across every mutation, rollbacks and resets included.
   */
  indexes?: Array<LookupIndex<Key, Value>>;
}

/**
 * Retrieves by key are strict: a batch containing any vanished key rejects
 * with NotFound instead of returning the survivors. Falls back to probing
 * keys one at a time so survivors are still distinguishable.
 */
const fetchSurvivors = async <Key extends record.Key, Value extends state.State>(
  fetch: NonNullable<TableParams<Key, Value>["fetch"]>,
  keys: Key[],
): Promise<Array<Keyed<Key, Value>>> => {
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

const DEFAULT_FETCH_DEBOUNCE = TimeSpan.milliseconds(10);

/**
 * The sole owner of one resource's record content and tombstones. Everything else in
 * the cache holds keys into it: answers store key lists, dispatch bookkeeping rides
 * entry deletion, and consumers assemble content at read time. An equal-value set
 * announces nothing: writers echoing state the table already holds (server echoes of
 * optimistic writes, idempotent backfills) are silenced by the equality check rather
 * than by writer identity.
 */
export class Table<
  Key extends record.Key = record.Key,
  Value extends state.State = state.State,
> {
  private readonly entries = new Map<Key, Value>();
  private readonly tombstones = new Map<Key, Deleted<Value>>();
  private readonly broadSubscribers = new Set<TableSubscriber<Key, Value>>();
  private readonly batchSubscribers = new Set<BatchSubscriber<Key, Value>>();
  private readonly keyedSubscribers = new Map<Key, Set<TableSubscriber<Key, Value>>>();
  private batching: TableEvent<Key, Value>[] | null = null;
  private readonly onError: (error: Error) => void;
  private readonly equal: (a: Value, b: Value, key: Key) => boolean;
  private readonly fetchEntries?: (keys: Key[]) => Promise<Array<Keyed<Key, Value>>>;
  private readonly hydrateMode: HydrateMode;
  private readonly fetchBatcher: debounce.Batcher<
    Key[],
    Array<Keyed<Key, Value>>
  > | null;
  private readonly indexes: Array<LookupIndex<Key, Value>>;
  private gen = 0;

  constructor({
    onError,
    equal = deep.equal,
    fetch,
    hydrate = "set",
    fetchDebounce = DEFAULT_FETCH_DEBOUNCE,
    indexes = [],
  }: TableParams<Key, Value>) {
    this.onError = onError;
    this.equal = equal;
    this.fetchEntries = fetch;
    this.hydrateMode = hydrate;
    this.indexes = [...indexes];
    this.fetchBatcher =
      fetch == null
        ? null
        : new debounce.Batcher({
            interval: fetchDebounce,
            exec: async (requests) => {
              const keys = new Set<Key>();
              requests.forEach(({ req }) => req.forEach((key) => keys.add(key)));
              let fetched: Array<Keyed<Key, Value>>;
              try {
                fetched = await fetch(Array.from(keys));
              } catch (exc) {
                if (!NotFoundError.matches(exc) || requests.length === 1)
                  throw errors.fromUnknown(exc);
                // A strict fetch rejects the whole batch when any caller's key
                // has vanished. Refetch per caller so each settles exactly as
                // its own request would have, keeping the batch transparent.
                await Promise.all(
                  requests.map(async ({ req, resolve, reject }) => {
                    try {
                      const mine = new Set(req);
                      resolve((await fetch(req)).filter(({ key }) => mine.has(key)));
                    } catch (exc) {
                      reject(exc);
                    }
                  }),
                );
                return;
              }
              // The window's fetch carries other callers' keys too; each caller
              // hydrates only the entries it asked for.
              requests.forEach(({ req, resolve }) => {
                const mine = new Set(req);
                resolve(fetched.filter(({ key }) => mine.has(key)));
              });
            },
          });
  }

  private applySet(key: Key, value: Value): void {
    this.entries.set(key, value);
    for (const index of this.indexes) index.set(key, value);
  }

  private applyDelete(key: Key): void {
    this.entries.delete(key);
    for (const index of this.indexes) index.delete(key);
  }

  /**
   * Registers a secondary index and backfills it from the live entries, for a
   * domain that owns an index's meaning but not the table it reads.
   * @returns the index.
   */
  index<I extends LookupIndex<Key, Value>>(index: I): I {
    this.entries.forEach((value, key) => index.set(key, value));
    this.indexes.push(index);
    return index;
  }

  private setOne(
    key: Key,
    value: state.SetArg<Value | undefined>,
  ): destructor.Destructor | undefined {
    const prev = this.entries.get(key);
    const next = state.executeSetter(value, prev);
    if (next == null || (prev != null && this.equal(next, prev, key))) return undefined;
    const prevTombstone = this.tombstones.get(key);
    this.tombstones.delete(key);
    this.applySet(key, next);
    this.notify({ variant: "set", key, value: next });

    return () => {
      if (prev === undefined) {
        this.applyDelete(key);
        if (prevTombstone != null) this.tombstones.set(key, prevTombstone);
        this.notify({ variant: "delete", key });
      } else {
        this.applySet(key, prev);
        this.notify({ variant: "set", key, value: prev });
      }
    };
  }

  /**
   * Sets the value for the given key, or applies a setter to the previous value. A
   * setter producing null/undefined, or a value equal to the current one, is a no-op.
   * @returns A rollback that undoes the set.
   */
  set(key: Key, value: state.SetArg<Value | undefined>): destructor.Destructor;
  /**
   * Sets every given keyed value, keying each entry by its key property.
   * @returns A rollback that undoes the inserted values in reverse order.
   */
  set(values: Keyed<Key, Value> | Array<Keyed<Key, Value>>): destructor.Destructor;
  set(
    keyOrValues: Key | Keyed<Key, Value> | Array<Keyed<Key, Value>>,
    value?: state.SetArg<Value | undefined>,
  ): destructor.Destructor {
    if (typeof keyOrValues !== "object")
      return this.setOne(keyOrValues as Key, value) ?? destructor.NOOP;
    const rollbacks: destructor.Destructor[] = [];
    this.batch(() =>
      array.toArray(keyOrValues).forEach((val) => {
        const rollback = this.setOne(val.key, val);
        if (rollback != null) rollbacks.push(rollback);
      }),
    );
    return () => this.batch(() => rollbacks.reverse().forEach((r) => r()));
  }

  private setIfAbsent(values: Array<Keyed<Key, Value>>): destructor.Destructor {
    const rollbacks: destructor.Destructor[] = [];
    this.batch(() =>
      values.forEach((val) => {
        if (this.entries.has(val.key)) return;
        const rollback = this.setOne(val.key, val);
        if (rollback != null) rollbacks.push(rollback);
      }),
    );
    return () => this.batch(() => rollbacks.reverse().forEach((r) => r()));
  }

  /**
   * Writes fetched records into the table under its declared hydrate mode:
   * "set" overwrites entries, "if-absent" leaves existing entries untouched.
   * @returns A rollback that undoes the entries this call wrote.
   */
  ingest(values: Keyed<Key, Value> | Array<Keyed<Key, Value>>): destructor.Destructor {
    const arr = array.toArray(values);
    if (this.hydrateMode === "if-absent") return this.setIfAbsent(arr);
    return this.set(arr);
  }

  /** Returns every entry in the table, or every entry the filter accepts. */
  get(filter?: (value: Value) => boolean): Value[];
  get(key: Key): Value | undefined;
  get(keys: Key[]): Value[];
  get(keys?: Key | Key[] | ((value: Value) => boolean)): Value | Value[] | undefined {
    if (keys === undefined) return Array.from(this.entries.values());
    if (typeof keys === "function")
      return Array.from(this.entries.values()).filter(keys);
    if (Array.isArray(keys))
      return keys
        .map((key) => this.entries.get(key))
        .filter((e): e is Value => e != null);
    return this.entries.get(keys);
  }

  has(key: Key): boolean {
    return this.entries.has(key);
  }

  /** Returns the key of every live entry. */
  keys(): Key[] {
    return Array.from(this.entries.keys());
  }

  /**
   * Returns the presence of the given key: present, tombstoned, or unknown.
   * Cache-internal surface: consumed by the query machinery, not domain code.
   */
  status(key: Key): EntryStatus {
    if (this.entries.has(key)) return "present";
    if (this.tombstones.has(key)) return "tombstoned";
    return "unknown";
  }

  /**
   * Returns the tombstone for the given key, or undefined if none exists.
   * Cache-internal surface: consumed by the query machinery, not domain code.
   */
  getTombstone(key: Key): Deleted<Value> | undefined {
    return this.tombstones.get(key);
  }

  /**
   * Resolves the given keys to records: serves cached entries and fetches the misses
   * through the table's fetch, hydrating results under the declared mode. With refresh,
   * every key is fetched regardless of presence and cached entries the fetch omits are
   * tombstoned. Returns the table's entries for the found keys in input order,
   * deduplicated; keys the cluster no longer has are omitted. Tables without a fetch
   * serve cached entries only.
   */
  async retrieve(keys: Key[], opts: { refresh?: boolean } = {}): Promise<Value[]> {
    if (this.fetchBatcher != null) {
      const misses =
        opts.refresh === true ? keys : keys.filter((key) => !this.entries.has(key));
      if (misses.length > 0) {
        const gen = this.gen;
        const fetched = await this.fetchBatcher.enqueue(misses);
        if (gen === this.gen)
          if (opts.refresh === true) {
            // A refresh is authoritative for its keys: cached entries the
            // fetch omitted vanished from the cluster and are tombstoned.
            const present = new Set<Key>(fetched.map(({ key }) => key));
            const vanished = misses.filter(
              (key) => !present.has(key) && this.entries.has(key),
            );
            this.batch(() => {
              if (vanished.length > 0) this.delete(vanished);
              if (fetched.length > 0) this.set(fetched);
            });
          } else if (fetched.length > 0) this.ingest(fetched);
      }
    }
    const seen = new Set<Key>();
    const results: Value[] = [];
    for (const key of keys) {
      if (seen.has(key)) continue;
      seen.add(key);
      const entry = this.entries.get(key);
      if (entry != null) results.push(entry);
    }
    return results;
  }

  /**
   * Deletes entries and notifies subscribers. Deleted values are retained as
   * tombstones until a subsequent set for the key.
   * @param key - The key(s) to delete or a filter function
   * @returns A rollback that restores the deleted entries
   */
  delete(
    key: Key | Key[] | ((value: Value, key: Key) => boolean),
  ): destructor.Destructor {
    return this.remove(key, true);
  }

  /**
   * Removes entries and notifies subscribers, retaining no corpses: the undo of a write
   * the record never survived, not a deletion. A reader finds the key unknown rather
   * than deleted.
   * @returns A rollback that restores the removed entries
   */
  evict(key: Key | Key[]): destructor.Destructor {
    return this.remove(key, false);
  }

  private remove(
    key: Key | Key[] | ((value: Value, key: Key) => boolean),
    tombstone: boolean,
  ): destructor.Destructor {
    const toDelete: Array<{ key: Key; value?: Value }> = [];

    if (typeof key === "function")
      this.entries.forEach((value, k) => {
        if (key(value, k)) toDelete.push({ key: k, value });
      });
    else
      array.toArray(key).forEach((k) => {
        const value = this.entries.get(k);
        toDelete.push({ key: k, value });
      });

    this.batch(() =>
      toDelete.forEach(({ key: k, value }) => {
        this.applyDelete(k);
        if (tombstone && value != null)
          this.tombstones.set(k, new Deleted(value, TimeStamp.now()));
        this.notify({ variant: "delete", key: k });
      }),
    );

    return () =>
      this.batch(() =>
        toDelete.forEach(({ key: k, value }) => {
          if (value == null) return;
          this.tombstones.delete(k);
          this.applySet(k, value);
          this.notify({ variant: "set", key: k, value });
        }),
      );
  }

  /**
   * Re-checks every cached entry against the cluster through the table's
   * fetch: refreshes entries that still exist and tombstones entries that
   * vanished. A no-op for empty tables and tables without a fetch.
   */
  async reconcile(): Promise<void> {
    const { fetchEntries } = this;
    if (fetchEntries == null) return;
    const keys = this.keys();
    if (keys.length === 0) return;
    const gen = this.gen;
    const values = await fetchSurvivors(fetchEntries, keys);
    // A reset mid-fetch means the cluster was replaced: writing the fetched
    // entries would repopulate the cleared table with old-cluster records.
    if (gen !== this.gen) return;
    const present = new Set<Key>(values.map(({ key }) => key));
    const vanished = keys.filter((k) => !present.has(k));
    this.batch(() => {
      if (vanished.length > 0) this.delete(vanished);
      if (values.length > 0) this.set(values);
    });
  }

  /**
   * Discards every entry and tombstone without notifying subscribers. Called
   * only by the cache's reset, which resets the answer spaces itself:
   * per-entry delete events would masquerade as deletions of live records.
   * Fetches in flight when the reset runs discard their results.
   */
  reset(): void {
    this.gen++;
    this.entries.clear();
    this.tombstones.clear();
    for (const index of this.indexes) index.reset();
  }

  /**
   * Subscribes to entry changes delivered per batched write: one call carrying every
   * event of the batch, in order, after all of the batch's entries have applied.
   * Cache-internal surface: consumed by the query machinery, not domain code;
   * per-record consumers use {@link subscribe}.
   * @returns A destructor that unsubscribes.
   */
  subscribeBatch(subscriber: BatchSubscriber<Key, Value>): destructor.Destructor {
    this.batchSubscribers.add(subscriber);
    return () => this.batchSubscribers.delete(subscriber);
  }

  /**
   * Subscribes to entry changes. With a key, only changes to that entry fire.
   * @returns A destructor that unsubscribes.
   */
  subscribe(subscriber: TableSubscriber<Key, Value>, key?: Key): destructor.Destructor {
    if (key == null) {
      this.broadSubscribers.add(subscriber);
      return () => this.broadSubscribers.delete(subscriber);
    }
    let held = this.keyedSubscribers.get(key);
    if (held == null) {
      held = new Set();
      this.keyedSubscribers.set(key, held);
    }
    held.add(subscriber);
    return () => {
      held.delete(subscriber);
      // Guarded on identity so a stale destructor cannot drop a set a later
      // subscription re-created under the same key.
      if (held.size === 0 && this.keyedSubscribers.get(key) === held)
        this.keyedSubscribers.delete(key);
    };
  }

  /**
   * Runs fn with event delivery deferred: writes apply to entries immediately, and
   * every event they produce is delivered as one batch when the outermost batch
   * exits. Delivery stays synchronous, inside this call. Nested batches join the
   * outermost one. Events flush even when fn throws, since the entries they describe
   * were already applied.
   */
  batch<T>(fn: () => T): T {
    if (this.batching != null) return fn();
    const events: TableEvent<Key, Value>[] = [];
    this.batching = events;
    try {
      return fn();
    } finally {
      this.batching = null;
      if (events.length > 0) this.flush(events);
    }
  }

  private notify(event: TableEvent<Key, Value>) {
    if (this.batching != null) {
      this.batching.push(event);
      return;
    }
    this.flush([event]);
  }

  private flush(events: TableEvent<Key, Value>[]) {
    for (const event of events) {
      this.deliver(this.keyedSubscribers.get(event.key), event);
      this.deliver(this.broadSubscribers, event);
    }
    for (const subscriber of this.batchSubscribers)
      try {
        subscriber(events);
      } catch (exc) {
        this.onError(new Error("failed to notify table subscriber", { cause: exc }));
      }
  }

  private deliver(
    subscribers: Iterable<TableSubscriber<Key, Value>> | undefined,
    event: TableEvent<Key, Value>,
  ) {
    if (subscribers == null) return;
    for (const subscriber of subscribers)
      try {
        subscriber(event);
      } catch (exc) {
        this.onError(new Error("failed to notify table subscriber", { cause: exc }));
      }
  }
}

/**
 * A mirror listener declaration: keeps the owning table current from one stream
 * channel. Built with {@link createSetListener}, {@link createDeleteListener}, or
 * {@link createFetchListener}; bound to its table by the cache.
 */
export interface ListenerSpec<
  Key extends record.Key = record.Key,
  Value extends state.State = state.State,
> {
  /** Binds the declaration to the owning table as a raw channel listener. */
  bind: (table: Table<Key, Value>) => Listener;
}

/**
 * Declares that the channel broadcasts records to mirror into the table.
 * By default the parsed record keys itself (`changed.key`) and is stored
 * as-is; `key` derives the entry key, `value` transforms the record and may
 * merge with the previous entry (return null/undefined to skip the write).
 */
export const createSetListener = <
  Z extends z.ZodType,
  Key extends record.Key = record.Key,
  Value extends state.State = state.State,
>(
  channel: string,
  schema: Z,
  opts: {
    key?: (changed: z.output<Z>) => Key;
    value?: (changed: z.output<Z>, prev: Value | undefined) => Value | null | undefined;
  } = {},
): ListenerSpec<Key, Value> => {
  const {
    key = (changed) => (changed as record.Keyed<Key>).key,
    value = (changed) => changed as Value,
  } = opts;
  return {
    bind: (table): Listener<Z> => ({
      channel,
      schema,
      onChange: (changed) => {
        table.batch(() =>
          changed.forEach((c) =>
            table.set(key(c), (prev) => value(c, prev) ?? undefined),
          ),
        );
      },
    }),
  };
};

/**
 * Declares that the channel broadcasts deletions to mirror into the table.
 * By default the parsed value is the entry key; `key` derives it instead.
 */
export const createDeleteListener = <
  Z extends z.ZodType,
  Key extends record.Key = record.Key,
  Value extends state.State = state.State,
>(
  channel: string,
  schema: Z,
  opts: { key?: (changed: z.output<Z>) => Key } = {},
): ListenerSpec<Key, Value> => {
  const { key = (changed) => changed as Key } = opts;
  return {
    bind: (table): Listener<Z> => ({
      channel,
      schema,
      onChange: (changed) => {
        table.delete(changed.map((c) => key(c)));
      },
    }),
  };
};

/**
 * Declares that the channel announces keys whose records changed. Announced
 * keys are refetched through the table's fetch and overwrite their entries.
 */
export const createFetchListener = <
  Z extends z.ZodType<Key | Key[]>,
  Key extends record.Key = record.Key,
  Value extends state.State = state.State,
>(
  channel: string,
  schema: Z,
): ListenerSpec<Key, Value> => ({
  bind: (table): Listener<Z> => ({
    channel,
    schema,
    onChange: async (changed) => {
      await table.retrieve(
        changed.flatMap((c) => array.toArray(c)),
        { refresh: true },
      );
    },
  }),
});

export const partialUpdate = <Key extends record.Key, Value extends object>(
  table: Table<Key, Value>,
  key: Key,
  value: Partial<Value>,
): destructor.Destructor =>
  table.set(key, (p) => (p == null ? undefined : { ...p, ...value }));
