// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, deep, destructor, type record, state, TimeStamp } from "@synnaxlabs/x";
import type z from "zod";

/** A deleted row's last known value, kept for restore and deletion UX. */
export interface Tombstone<Value> {
  /** The row's value at the moment it was deleted. */
  corpse: Value;
  /** When the deletion was applied to this table. */
  deletedAt: TimeStamp;
}

/**
 * Presence of a key in a table: "present" when a live row exists, "tombstoned"
 * when the row was deleted and its corpse is retained, "unknown" when the
 * table has never seen the key or the corpse was cleared.
 */
export type RowStatus = "present" | "tombstoned" | "unknown";

/** A change to a single row of a {@link Table}. */
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
   * returning only the rows that still exist. Powers {@link Table.retrieve},
   * key-announce listeners, and reconciliation. Omit for tables with no
   * server backing (they are skipped by all three).
   */
  fetch?: (keys: Key[]) => Promise<Array<Value & record.Keyed<Key>>>;
  /**
   * How fetched records hydrate the table: "set" (default) overwrites rows,
   * "if-absent" preserves existing rows. Dispatch-backed document tables use
   * "if-absent" so fetches never clobber locally replayed edits.
   */
  hydrate?: "set" | "if-absent";
}

/**
 * The sole owner of one resource's record content and tombstones. Everything
 * else in the cache holds keys into it: answers store key lists, dispatch
 * bookkeeping rides row deletion, and consumers assemble content at read time.
 *
 * An equal-value set announces nothing: writers echoing state the table
 * already holds (server echoes of optimistic writes, idempotent backfills)
 * are silenced by the equality check rather than by writer identity.
 */
export class Table<
  Key extends record.Key = record.Key,
  Value extends state.State = state.State,
> {
  private readonly rows = new Map<Key, Value>();
  private readonly tombstones = new Map<Key, Tombstone<Value>>();
  private readonly subscribers = new Map<
    TableSubscriber<Key, Value>,
    Key | undefined
  >();
  private readonly onError: (error: Error) => void;
  private readonly equal: (a: Value, b: Value, key: Key) => boolean;
  private readonly fetchRows?: (
    keys: Key[],
  ) => Promise<Array<Value & record.Keyed<Key>>>;
  private readonly hydrateMode: "set" | "if-absent";

  constructor({ onError, equal, fetch, hydrate }: TableParams<Key, Value>) {
    this.onError = onError;
    this.equal = equal ?? ((a, b) => deep.equal(a, b));
    this.fetchRows = fetch;
    this.hydrateMode = hydrate ?? "set";
  }

  private setOne(
    key: Key,
    value: state.SetArg<Value | undefined>,
  ): destructor.Destructor | undefined {
    const prev = this.rows.get(key);
    const next = state.executeSetter(value, prev);
    if (next == null || (prev != null && this.equal(next, prev, key))) return undefined;
    const prevTombstone = this.tombstones.get(key);
    this.tombstones.delete(key);
    this.rows.set(key, next);
    this.notify({ variant: "set", key, value: next });

    return () => {
      if (prev === undefined) {
        this.rows.delete(key);
        if (prevTombstone != null) this.tombstones.set(key, prevTombstone);
        this.notify({ variant: "delete", key });
      } else {
        this.rows.set(key, prev);
        this.notify({ variant: "set", key, value: prev });
      }
    };
  }

  /**
   * Sets the value for the given key, or applies a setter to the previous
   * value. A setter producing null/undefined, or a value equal to the current
   * one, is a no-op.
   * @returns A rollback that undoes the set.
   */
  set(key: Key, value: state.SetArg<Value | undefined>): destructor.Destructor;
  /**
   * Sets every given keyed value, keying each row by its key property.
   * @returns A rollback that undoes the inserted values in reverse order.
   */
  set(
    values: (Value & record.Keyed<Key>) | Array<Value & record.Keyed<Key>>,
  ): destructor.Destructor;
  set(
    keyOrValues: Key | (Value & record.Keyed<Key>) | Array<Value & record.Keyed<Key>>,
    value?: state.SetArg<Value | undefined>,
  ): destructor.Destructor {
    if (typeof keyOrValues !== "object")
      return this.setOne(keyOrValues as Key, value) ?? destructor.NOOP;
    const rollbacks: destructor.Destructor[] = [];
    array.toArray(keyOrValues).forEach((val) => {
      const rollback = this.setOne(val.key, val);
      if (rollback != null) rollbacks.push(rollback);
    });
    return () => rollbacks.reverse().forEach((r) => r());
  }

  private setIfAbsent(
    values: Array<Value & record.Keyed<Key>>,
  ): destructor.Destructor {
    const rollbacks: destructor.Destructor[] = [];
    values.forEach((val) => {
      if (this.rows.has(val.key)) return;
      const rollback = this.setOne(val.key, val);
      if (rollback != null) rollbacks.push(rollback);
    });
    return () => rollbacks.reverse().forEach((r) => r());
  }

  /**
   * Writes fetched records into the table under its declared hydrate mode:
   * "set" overwrites rows, "if-absent" leaves existing rows untouched.
   * @returns A rollback that undoes the rows this call wrote.
   */
  ingest(
    values: (Value & record.Keyed<Key>) | Array<Value & record.Keyed<Key>>,
  ): destructor.Destructor {
    const arr = array.toArray(values);
    if (this.hydrateMode === "if-absent") return this.setIfAbsent(arr);
    return this.set(arr);
  }

  /** Returns every row in the table. */
  get(): Value[];
  get(key: Key): Value | undefined;
  get(keys: Key[]): Value[];
  get(filter: (value: Value) => boolean): Value[];
  get(
    keys?: Key | Key[] | ((value: Value) => boolean),
  ): Value | Value[] | undefined {
    if (keys === undefined) return Array.from(this.rows.values());
    if (typeof keys === "function") return Array.from(this.rows.values()).filter(keys);
    if (Array.isArray(keys))
      return keys.map((key) => this.rows.get(key)).filter((e) => e != null) as Value[];
    return this.rows.get(keys);
  }

  has(key: Key): boolean {
    return this.rows.has(key);
  }

  /**
   * Returns the presence of the given key: present, tombstoned, or unknown.
   * Cache-internal surface: consumed by the query machinery, not domain code.
   */
  status(key: Key): RowStatus {
    if (this.rows.has(key)) return "present";
    if (this.tombstones.has(key)) return "tombstoned";
    return "unknown";
  }

  /**
   * Returns the tombstone for the given key, or undefined if none exists.
   * Cache-internal surface: consumed by the query machinery, not domain code.
   */
  getTombstone(key: Key): Tombstone<Value> | undefined {
    return this.tombstones.get(key);
  }

  /**
   * Resolves the given keys to records: serves cached rows and fetches the
   * misses through the table's fetch, hydrating results under the declared
   * mode. With refresh, every key is fetched regardless of presence. Returns
   * the table's rows for the found keys in input order, deduplicated; keys
   * the cluster no longer has are omitted. Tables without a fetch serve
   * cached rows only.
   */
  async retrieve(keys: Key[], opts: { refresh?: boolean } = {}): Promise<Value[]> {
    const { fetchRows } = this;
    if (fetchRows != null) {
      const misses =
        opts.refresh === true ? keys : keys.filter((key) => !this.rows.has(key));
      if (misses.length > 0) {
        const fetched = await fetchRows(misses);
        if (fetched.length > 0)
          if (opts.refresh === true) this.set(fetched);
          else this.ingest(fetched);
      }
    }
    const seen = new Set<Key>();
    const results: Value[] = [];
    for (const key of keys) {
      if (seen.has(key)) continue;
      seen.add(key);
      const row = this.rows.get(key);
      if (row != null) results.push(row);
    }
    return results;
  }

  /**
   * Deletes rows and notifies subscribers. Deleted values are retained as
   * tombstones until a subsequent set for the key.
   * @param key - The key(s) to delete or a filter function
   * @returns A rollback that restores the deleted rows
   */
  delete(
    key: Key | Key[] | ((value: Value, key: Key) => boolean),
  ): destructor.Destructor {
    const toDelete: Array<{ key: Key; value?: Value }> = [];

    if (typeof key === "function")
      this.rows.forEach((value, k) => {
        if (key(value, k)) toDelete.push({ key: k, value });
      });
    else
      array.toArray(key).forEach((k) => {
        const value = this.rows.get(k);
        toDelete.push({ key: k, value });
      });

    toDelete.forEach(({ key: k, value }) => {
      this.rows.delete(k);
      if (value != null)
        this.tombstones.set(k, { corpse: value, deletedAt: TimeStamp.now() });
      this.notify({ variant: "delete", key: k });
    });

    return () =>
      toDelete.forEach(({ key: k, value }) => {
        if (value == null) return;
        this.tombstones.delete(k);
        this.rows.set(k, value);
        this.notify({ variant: "set", key: k, value });
      });
  }

  /**
   * Subscribes to row changes. With a key, only changes to that row fire.
   * @returns A destructor that unsubscribes.
   */
  subscribe(subscriber: TableSubscriber<Key, Value>, key?: Key): destructor.Destructor {
    this.subscribers.set(subscriber, key);
    return () => this.subscribers.delete(subscriber);
  }

  private notify(event: TableEvent<Key, Value>) {
    this.subscribers.forEach((key, subscriber) => {
      if (key != null && key !== event.key) return;
      try {
        subscriber(event);
      } catch (exc) {
        this.onError(
          new Error("failed to notify table subscriber", {
            cause: exc,
          }),
        );
      }
    });
  }
}

/**
 * A mirror listener declaration: keeps the owning table current from one
 * stream channel. Built with {@link createSetListener},
 * {@link createDeleteListener}, or {@link createFetchListener}; bound to its
 * table by the cache.
 */
export type ListenerSpec<
  Key extends record.Key = record.Key,
  Value extends state.State = state.State,
> =
  | {
      kind: "set";
      channel: string;
      schema: z.ZodType;
      key: (changed: unknown) => Key;
      value: (changed: unknown, prev: Value | undefined) => Value | null | undefined;
    }
  | {
      kind: "delete";
      channel: string;
      schema: z.ZodType;
      key: (changed: unknown) => Key;
    }
  | { kind: "fetch"; channel: string; schema: z.ZodType };

/**
 * Declares that the channel broadcasts records to mirror into the table.
 * By default the parsed record keys itself (`changed.key`) and is stored
 * as-is; `key` derives the row key, `value` transforms the record and may
 * merge with the previous row (return null/undefined to skip the write).
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
): ListenerSpec<Key, Value> => ({
  kind: "set",
  channel,
  schema,
  key:
    opts.key != null
      ? (changed) => opts.key!(changed as z.output<Z>)
      : (changed) => (changed as record.Keyed<Key>).key,
  value:
    opts.value != null
      ? (changed, prev) => opts.value!(changed as z.output<Z>, prev)
      : (changed) => changed as Value,
});

/**
 * Declares that the channel broadcasts deletions to mirror into the table.
 * By default the parsed value is the row key; `key` derives it instead.
 */
export const createDeleteListener = <
  Z extends z.ZodType,
  Key extends record.Key = record.Key,
  Value extends state.State = state.State,
>(
  channel: string,
  schema: Z,
  opts: { key?: (changed: z.output<Z>) => Key } = {},
): ListenerSpec<Key, Value> => ({
  kind: "delete",
  channel,
  schema,
  key:
    opts.key != null
      ? (changed) => opts.key!(changed as z.output<Z>)
      : (changed) => changed as Key,
});

/**
 * Declares that the channel announces keys whose records changed. Announced
 * keys are refetched through the table's fetch and overwrite their rows.
 */
export const createFetchListener = <
  Z extends z.ZodType,
  Key extends record.Key = record.Key,
  Value extends state.State = state.State,
>(
  channel: string,
  schema: Z,
): ListenerSpec<Key, Value> => ({ kind: "fetch", channel, schema });

export const partialUpdate = <Key extends record.Key, Value extends object>(
  table: Table<Key, Value>,
  key: Key,
  value: Partial<Value>,
): destructor.Destructor =>
  table.set(key, (p) => (p == null ? undefined : { ...p, ...value }));
