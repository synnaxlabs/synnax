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
  type destructor,
  type narrow,
  type observe,
  type record,
  TimeStamp,
} from "@synnaxlabs/x";
import type z from "zod";

import { executeSetter, type SetArg, type State } from "@/cache/state";
import { type ErrorHandler } from "@/cache/types";

interface ListenerScope<K extends record.Key> {
  scope: string;
  key?: K;
}

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

export interface SetHandler<Value, SetExtra extends unknown | undefined = undefined> {
  (value: Value, variant: SetExtra): void | Promise<void>;
}

export interface DeleteHandler<K extends record.Key> {
  (key: K): void | Promise<void>;
}

/** A deleted entry's last known value, kept for restore and deletion UX. */
export interface Tombstone<Value> {
  /** The entry's value at the moment it was deleted. */
  corpse: Value;
  /** When the deletion was applied to this store. */
  deletedAt: TimeStamp;
}

/**
 * Presence of a key in a store: "present" when a live entry exists,
 * "tombstoned" when the entry was deleted and its corpse is retained,
 * "unknown" when the store has never seen the key or the corpse was cleared.
 */
export type EntryStatus = "present" | "tombstoned" | "unknown";

export class ScopedUnaryStore<
  Key extends record.Key = record.Key,
  Value extends State = State,
  SetExtra extends unknown | undefined = undefined,
> {
  private readonly entries: Map<Key, Value> = new Map();
  // Parallel to entries: a key is never in both maps. delete moves the value
  // here; set clears it.
  private readonly tombstones: Map<Key, Tombstone<Value>> = new Map();
  private readonly setListeners: Map<SetHandler<Value, SetExtra>, ListenerScope<Key>> =
    new Map();
  private readonly deleteListeners: Map<DeleteHandler<Key>, ListenerScope<Key>> =
    new Map();
  private readonly handleError: ErrorHandler;
  private readonly equal: (a: Value, b: Value, key: Key) => boolean;

  constructor(
    handleError: ErrorHandler,
    equal: (a: Value, b: Value, key: Key) => boolean = () => false,
  ) {
    this.handleError = handleError;
    this.equal = equal;
  }

  private setOne(
    scope: string,
    key: Key,
    value: SetArg<Value | undefined>,
    variant: SetExtra,
  ): destructor.Destructor | undefined {
    const prev = this.entries.get(key);
    const next = executeSetter(value, prev);
    if (next == null || (prev != null && this.equal(next, prev, key))) return undefined;
    const prevTombstone = this.tombstones.get(key);
    this.tombstones.delete(key);
    this.entries.set(key, next);
    this.notifySet(scope, key, next, variant);

    return () => {
      if (prev === undefined) {
        this.entries.delete(key);
        if (prevTombstone != null) this.tombstones.set(key, prevTombstone);
        this.notifyDelete(scope, key);
      } else {
        this.entries.set(key, prev);
        this.notifySet(scope, key, prev, variant);
      }
    };
  }

  /**
   * Sets a value for the given key in the store.
   *
   * @param key - The key to set
   * @param value - The value to set, or a function to compute the value from the previous state
   * @param opts - Options for the set operation
   * @returns A rollback function that undoes the set operation
   */
  set(
    scope: string,
    key: Key | Array<Value & record.Keyed<Key>> | (Value & record.Keyed<Key>),
    value?: SetArg<Value | undefined> | SetExtra,
    extra?: SetExtra,
  ): () => void {
    const rollbacks: destructor.Destructor[] = [];

    // Case 1: Array of values with keys
    if (Array.isArray(key))
      key.forEach((val) => {
        const rollback = this.setOne(scope, val.key, val, value as SetExtra);
        if (rollback != null) rollbacks.push(rollback);
      });
    // Case 2: Single value with key property
    else if (typeof key === "object" && "key" in key) {
      const val = key;
      const rollback = this.setOne(scope, val.key, val, value as SetExtra);
      if (rollback != null) rollbacks.push(rollback);
    }
    // Case 3: Key with separate value
    else {
      const rollback = this.setOne(
        scope,
        key as Key,
        value as SetArg<Value | undefined>,
        extra as SetExtra,
      );
      if (rollback != null) rollbacks.push(rollback);
    }

    return () => rollbacks.reverse().forEach((r) => r());
  }

  /**
   * Sets the given value(s) only for keys that are not already present in the
   * store, leaving existing entries untouched. Use this to hydrate the store
   * from a bulk fetch without clobbering entries that already hold live state
   * (e.g. a document with unsynced local edits).
   *
   * @returns A rollback function that removes the entries this call inserted.
   */
  setIfAbsent(
    scope: string,
    value: (Value & record.Keyed<Key>) | Array<Value & record.Keyed<Key>>,
    variant?: SetExtra,
  ): () => void {
    const rollbacks: destructor.Destructor[] = [];
    array.toArray(value).forEach((val) => {
      if (this.entries.has(val.key)) return;
      const rollback = this.setOne(scope, val.key, val, variant as SetExtra);
      if (rollback != null) rollbacks.push(rollback);
    });
    return () => rollbacks.reverse().forEach((r) => r());
  }

  get(key: Key): Value | undefined;
  get(keys: Key[] | ((value: Value) => boolean)): Value[];
  get(keys: Key | Key[] | ((value: Value) => boolean)): Value | Value[] | undefined {
    if (typeof keys === "function")
      return Array.from(this.entries.values()).filter(keys);
    if (Array.isArray(keys))
      return keys
        .map((key) => this.entries.get(key))
        .filter((e) => e != null) as Value[];
    return this.entries.get(keys);
  }

  list(): Value[] {
    return Array.from(this.entries.values());
  }

  has(key: Key | Key[]): boolean {
    if (Array.isArray(key)) return key.every((k) => this.entries.has(k));
    return this.entries.has(key);
  }

  /** Returns the presence of the given key: present, tombstoned, or unknown. */
  status(key: Key): EntryStatus {
    if (this.entries.has(key)) return "present";
    if (this.tombstones.has(key)) return "tombstoned";
    return "unknown";
  }

  /** Returns the tombstone for the given key, or undefined if none exists. */
  getTombstone(key: Key): Tombstone<Value> | undefined {
    return this.tombstones.get(key);
  }

  /**
   * Deletes entries from the store and notifies delete listeners. Deleted
   * values are retained as tombstones until a subsequent set for the key.
   * @param key - The key(s) to delete or a filter function
   * @param variant - Optional variant data for set operations during rollback
   * @returns A rollback function that restores the deleted entries
   */
  delete(
    scope: string,
    key: Key | Key[] | ((value: Value, key: Key) => boolean),
    variant?: SetExtra,
  ): () => void {
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

    toDelete.forEach(({ key: k, value }) => {
      this.entries.delete(k);
      if (value != null)
        this.tombstones.set(k, { corpse: value, deletedAt: TimeStamp.now() });
      this.notifyDelete(scope, k);
    });

    return () =>
      toDelete.forEach(({ key: k, value }) => {
        if (value == null) return;
        this.tombstones.delete(k);
        this.entries.set(k, value);
        this.notifySet(scope, k, value, variant as SetExtra);
      });
  }

  clear() {
    this.entries.clear();
    this.tombstones.clear();
  }

  /**
   * Registers a listener for set operations.
   *
   * @param callback - Function to call when a value is set
   * @param key - Optional key to filter notifications (if provided, only changes to this
   * key trigger the callback)
   * @returns A destructor function to remove the listener
   */
  onSet(
    scope: string,
    callback: SetHandler<Value, SetExtra>,
    key?: Key,
  ): destructor.Destructor {
    this.setListeners.set(callback, { scope, key });
    return () => this.setListeners.delete(callback);
  }

  /**
   * Registers a listener for delete operations.
   *
   * @param callback - Function to call when a value is deleted
   * @param key - Optional key to filter notifications (if provided, only deletion
   * of this key triggers the callback)
   * @returns A destructor function to remove the listener
   */
  onDelete(
    scope: string,
    callback: observe.AsyncHandler<Key> | observe.Handler<Key>,
    key?: Key,
  ): destructor.Destructor {
    this.deleteListeners.set(callback, { scope, key });
    return () => this.deleteListeners.delete(callback);
  }

  private notifySet(scope: string, key: Key, value: Value, variant: SetExtra) {
    this.setListeners.forEach((listenerKey, callback) => {
      const matchesKey = listenerKey.key == null || listenerKey.key === key;
      const matchesScope = listenerKey.scope !== scope;
      if (matchesKey && matchesScope)
        this.handleError(
          async () => await callback(value, variant),
          "Failed to notify set listener",
        );
    });
  }

  private notifyDelete(scope: string, key: Key) {
    this.deleteListeners.forEach((listenerKey, callback) => {
      const matchesKey = listenerKey.key == null || listenerKey.key === key;
      const matchesScope = listenerKey.scope !== scope;
      if (matchesKey && matchesScope)
        this.handleError(
          async () => await callback(key),
          "Failed to notify delete listener",
        );
    });
  }

  scope(scope: string): UnaryStore<Key, Value, SetExtra> {
    return {
      set: (
        key: Key | Array<Value & record.Keyed<Key>> | (Value & record.Keyed<Key>),
        valueOrVariant?: SetArg<Value | undefined> | SetExtra,
        variant?: SetExtra,
      ): (() => void) => this.set(scope, key, valueOrVariant, variant),
      setIfAbsent: (
        value: (Value & record.Keyed<Key>) | Array<Value & record.Keyed<Key>>,
        variant?: SetExtra,
      ): (() => void) => this.setIfAbsent(scope, value, variant),
      get: this.get.bind(this),
      list: () => this.list(),
      has: (key: Key | Key[]) => this.has(key),
      status: (key: Key) => this.status(key),
      getTombstone: (key: Key) => this.getTombstone(key),
      delete: (
        key: Key | Key[] | ((value: Value, key: Key) => boolean),
        variant?: SetExtra,
      ): (() => void) => this.delete(scope, key, variant),
      onSet: (callback: SetHandler<Value, SetExtra>, key?: Key) =>
        this.onSet(scope, callback, key),
      onDelete: (callback, key) => this.onDelete(scope, callback, key),
    };
  }
}

/**
 * Configuration for listening to changes on a specific Synnax channel.
 *
 * @template ScopedStores - The type of the stores available to the listener
 * @template Schema - Zod schema type for validating channel data
 */
export interface ChannelListener<
  ScopedStores extends Stores = {},
  Z extends z.ZodType = z.ZodType,
> {
  /** The name of the Synnax channel to listen to */
  channel: string;
  /** Zod schema for parsing and validating channel data */
  schema: Z;
  /** Callback function invoked when the channel data changes */
  onChange: (
    params: ChannelListenerParams<ScopedStores, Z>,
  ) => Promise<unknown> | unknown;
}

/**
 * Arguments passed to a channel listener's onChange callback.
 *
 * @template ScopedStores - The type of the stores available to the listener
 * @template Z - Zod schema type for validating channel data
 */
export type ChannelListenerParams<
  ScopedStores extends Stores = {},
  Z extends z.ZodType = z.ZodType,
> = {
  /** The parsed and validated data that changed */
  changed: z.output<Z>;
  /** The stores available to the listener */
  store: ScopedStores;
};

/**
 * Configuration for a single UnaryStore including its channel listeners.
 *
 * @template ScopedStores - The type of the scoped stores
 */
export interface UnaryStoreConfig<
  ScopedStores extends Stores = {},
  Key extends record.Key = record.Key,
  Value extends State = State,
> {
  /** Function to determine if two values are equal */
  equal?: (a: Value, b: Value, key: Key) => boolean;
  /** Array of channel listeners to register for this store */
  listeners: ChannelListener<ScopedStores>[];
  /**
   * Fetches the current value of the given keys, returning only the entries
   * that still exist. Used by epoch reconciliation: keys missing from the
   * result are tombstoned, returned values refresh the store. Stores without
   * a refetch are skipped during reconciliation.
   */
  refetch?: (keys: Key[]) => Promise<Array<Value & record.Keyed<Key>>>;
}

/**
 * Configuration object for creating a store with multiple UnaryStore instances.
 * Keys are store names and values are their configurations.
 *
 * @template ScopedStores - The type of the scoped stores
 */
export interface StoreConfig<ScopedStores extends Stores = {}> {
  [key: string]: UnaryStoreConfig<ScopedStores, any, any>;
}

/**
 * Read surface of a single resource's cache: lookups, presence, tombstones,
 * and change subscriptions. Writes flow through the owning domain client.
 */
export type Store<
  Key extends record.Key = record.Key,
  Value extends State = State,
  SetExtra extends unknown | undefined = undefined,
> = {
  get(key: Key): Value | undefined;
  get(keys: Key[] | ((value: Value) => boolean)): Value[];
  list(): Value[];
  has(key: Key | Key[]): boolean;
  status(key: Key): EntryStatus;
  getTombstone(key: Key): Tombstone<Value> | undefined;
  onSet(callback: SetHandler<Value, SetExtra>, key?: Key): destructor.Destructor;
  onDelete(callback: DeleteHandler<Key>, key?: Key): destructor.Destructor;
};

export type UnaryStore<
  Key extends record.Key = record.Key,
  Value extends State = State,
  SetExtra extends unknown | undefined = undefined,
> = Store<Key, Value, SetExtra> &
  (narrow.IsUndefined<SetExtra> extends true
    ? {
        set(key: Key, value: SetArg<Value | undefined>): () => void;
        set(
          value: (Value & record.Keyed<Key>) | Array<Value & record.Keyed<Key>>,
        ): () => void;
        setIfAbsent(
          value: (Value & record.Keyed<Key>) | Array<Value & record.Keyed<Key>>,
        ): () => void;
        delete(key: Key | Key[] | ((value: Value, key: Key) => boolean)): () => void;
      }
    : {
        set(key: Key, value: SetArg<Value | undefined>, variant: SetExtra): () => void;
        set(
          values: (Value & record.Keyed<Key>) | Array<Value & record.Keyed<Key>>,
          variant: SetExtra,
        ): () => void;
        setIfAbsent(
          values: (Value & record.Keyed<Key>) | Array<Value & record.Keyed<Key>>,
          variant: SetExtra,
        ): () => void;
        delete(
          key: Key | Key[] | ((value: Value, key: Key) => boolean),
          variant?: SetExtra,
        ): () => void;
      });

/**
 * Base interface for a collection of UnaryStore instances.
 * Each property is a UnaryStore with its own key-value type.
 */
export interface Stores {
  [key: string]: UnaryStore<any, any, undefined> | UnaryStore<string, any, unknown>;
}

export interface InternalStore {
  [key: string]: ScopedUnaryStore<any, any, any>;
}

/**
 * Creates a new store instance from the provided configuration.
 * Each key in the config becomes a UnaryStore in the resulting store.
 *
 * @template ScopedStores - The type of the stores to create
 * @param config - Configuration object defining the store structure
 * @returns A new store instance with UnaryStore instances for each config key
 */
export const createStore = <ScopedStores extends Stores>(
  config: StoreConfig<ScopedStores>,
  handleError: ErrorHandler,
): InternalStore =>
  Object.fromEntries(
    Object.entries(config).map(([key, { equal }]) => [
      key,
      new ScopedUnaryStore<string, State, string>(handleError, equal),
    ]),
  );

export const scopeStore = <ScopedStores extends Stores>(
  store: InternalStore,
  scope: string,
): ScopedStores =>
  Object.fromEntries(
    Object.entries(store).map(([key]): [string, unknown] => [
      key,
      store[key].scope(scope),
    ]),
  ) as ScopedStores;

export const partialUpdate = <Key extends record.Key, Value extends Record<any, any>>(
  store: UnaryStore<Key, Value>,
  key: Key,
  value: Partial<Value>,
): destructor.Destructor =>
  store.set(key, (p) => (p == null ? undefined : { ...p, ...value }));
