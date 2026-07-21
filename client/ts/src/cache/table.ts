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
  deep,
  type destructor,
  type record,
  state,
  TimeStamp,
} from "@synnaxlabs/x";
import type z from "zod";

/// Reorders the given items to match the order of keys, dropping any key that
/// has no corresponding item and deduplicating repeated keys. Used by
/// multi-retrieve query implementations to preserve the caller's input key order
/// across cached + freshly-fetched items.
export const orderByKeys = <K extends record.Key, V>(
  keys: K[],
  items: V[],
  getKey: (v: V) => K,
): V[] => {
  const byKey = new Map<K, V>();
  for (const item of items) byKey.set(getKey(item), item);
  const ordered: V[] = [];
  const seen = new Set<K>();
  for (const key of keys) {
    if (seen.has(key)) continue;
    seen.add(key);
    const item = byKey.get(key);
    if (item != null) ordered.push(item);
  }
  return ordered;
};

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

  constructor(
    onError: (error: Error) => void,
    equal: (a: Value, b: Value, key: Key) => boolean = (a, b) => deep.equal(a, b),
  ) {
    this.onError = onError;
    this.equal = equal;
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
  set(key: Key, value: state.SetArg<Value | undefined>): destructor.Destructor {
    return this.setOne(key, value) ?? (() => {});
  }

  /**
   * Sets every given keyed value.
   * @returns A rollback that undoes the inserted values in reverse order.
   */
  setMany(
    values: (Value & record.Keyed<Key>) | Array<Value & record.Keyed<Key>>,
  ): destructor.Destructor {
    const rollbacks: destructor.Destructor[] = [];
    array.toArray(values).forEach((val) => {
      const rollback = this.setOne(val.key, val);
      if (rollback != null) rollbacks.push(rollback);
    });
    return () => rollbacks.reverse().forEach((r) => r());
  }

  /**
   * Sets the given value(s) only for keys not already present, leaving
   * existing rows untouched. Use this to hydrate from a bulk fetch without
   * clobbering rows that already hold live state (e.g. a document with
   * unsynced local edits).
   * @returns A rollback that removes the rows this call inserted.
   */
  setIfAbsent(
    values: (Value & record.Keyed<Key>) | Array<Value & record.Keyed<Key>>,
  ): destructor.Destructor {
    const rollbacks: destructor.Destructor[] = [];
    array.toArray(values).forEach((val) => {
      if (this.rows.has(val.key)) return;
      const rollback = this.setOne(val.key, val);
      if (rollback != null) rollbacks.push(rollback);
    });
    return () => rollbacks.reverse().forEach((r) => r());
  }

  get(key: Key): Value | undefined;
  get(keys: Key[] | ((value: Value) => boolean)): Value[];
  get(keys: Key | Key[] | ((value: Value) => boolean)): Value | Value[] | undefined {
    if (typeof keys === "function") return Array.from(this.rows.values()).filter(keys);
    if (Array.isArray(keys))
      return keys.map((key) => this.rows.get(key)).filter((e) => e != null) as Value[];
    return this.rows.get(keys);
  }

  list(): Value[] {
    return Array.from(this.rows.values());
  }

  has(key: Key | Key[]): boolean {
    if (Array.isArray(key)) return key.every((k) => this.rows.has(k));
    return this.rows.has(key);
  }

  /** Returns the presence of the given key: present, tombstoned, or unknown. */
  status(key: Key): RowStatus {
    if (this.rows.has(key)) return "present";
    if (this.tombstones.has(key)) return "tombstoned";
    return "unknown";
  }

  /** Returns the tombstone for the given key, or undefined if none exists. */
  getTombstone(key: Key): Tombstone<Value> | undefined {
    return this.tombstones.get(key);
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

  clear() {
    this.rows.clear();
    this.tombstones.clear();
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
 * Configuration for listening to changes on a specific Synnax channel.
 *
 * @template Tbls - The tables available to the listener
 * @template Z - Zod schema type for validating channel data
 */
export interface ChannelListener<
  Tbls extends Tables = {},
  Z extends z.ZodType = z.ZodType,
> {
  /** The name of the Synnax channel to listen to */
  channel: string;
  /** Zod schema for parsing and validating channel data */
  schema: Z;
  /** Callback function invoked when the channel data changes */
  onChange: (params: ChannelListenerParams<Tbls, Z>) => Promise<unknown> | unknown;
}

/**
 * Arguments passed to a channel listener's onChange callback.
 *
 * @template Tbls - The tables available to the listener
 * @template Z - Zod schema type for validating channel data
 */
export type ChannelListenerParams<
  Tbls extends Tables = {},
  Z extends z.ZodType = z.ZodType,
> = {
  /** The parsed and validated data that changed */
  changed: z.output<Z>;
  /** The tables available to the listener */
  store: Tbls;
};

/**
 * Configuration for a single {@link Table} including its channel listeners.
 */
export interface TableConfig<
  Tbls extends Tables = {},
  Key extends record.Key = record.Key,
  Value extends state.State = state.State,
> {
  /** Overrides the deep-equality default used to silence redundant sets. */
  equal?: (a: Value, b: Value, key: Key) => boolean;
  /** Array of channel listeners to register for this table */
  listeners: ChannelListener<Tbls>[];
  /**
   * Fetches the current value of the given keys, returning only the rows
   * that still exist. Used by epoch reconciliation: keys missing from the
   * result are tombstoned, returned values refresh the table. Tables without
   * a refetch are skipped during reconciliation.
   */
  refetch?: (keys: Key[]) => Promise<Array<Value & record.Keyed<Key>>>;
}

/**
 * Configuration map for a cache's tables. Keys are table names and values are
 * their configurations.
 */
export interface TableConfigs<Tbls extends Tables = {}> {
  [key: string]: TableConfig<Tbls, any, any>;
}

/**
 * A collection of tables keyed by name, as handed to channel listeners.
 */
export interface Tables {
  [key: string]: Table<any, any>;
}

export const partialUpdate = <Key extends record.Key, Value extends Record<any, any>>(
  table: Table<Key, Value>,
  key: Key,
  value: Partial<Value>,
): destructor.Destructor =>
  table.set(key, (p) => (p == null ? undefined : { ...p, ...value }));
