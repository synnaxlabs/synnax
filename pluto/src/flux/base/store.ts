// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax } from "@synnaxlabs/client";
import {
  array,
  type destructor,
  type narrow,
  type observe,
  type record,
} from "@synnaxlabs/x";
import type z from "zod";

import { state } from "@/state";
import { type status } from "@/status/aether";

interface ListenerScope<K extends record.Key> {
  scope: string;
  key?: K;
}

export interface SetHandler<Value, SetExtra extends unknown | undefined = undefined> {
  (value: Value, variant: SetExtra): void | Promise<void>;
}

export interface DeleteHandler<K extends record.Key> {
  (key: K): void | Promise<void>;
}

export class ScopedUnaryStore<
  Key extends record.Key = record.Key,
  Value extends state.State = state.State,
  SetExtra extends unknown | undefined = undefined,
> {
  private readonly entries: Map<Key, Value> = new Map();
  private readonly setListeners: Map<SetHandler<Value, SetExtra>, ListenerScope<Key>> =
    new Map();
  private readonly deleteListeners: Map<DeleteHandler<Key>, ListenerScope<Key>> =
    new Map();
  private readonly handleError: status.ErrorHandler;
  private readonly equal: (a: Value, b: Value, key: Key) => boolean;

  constructor(
    handleError: status.ErrorHandler,
    equal: (a: Value, b: Value, key: Key) => boolean = () => false,
  ) {
    this.handleError = handleError;
    this.equal = equal;
  }

  private setOne(
    scope: string,
    key: Key,
    value: state.SetArg<Value | undefined>,
    variant: SetExtra,
  ): destructor.Destructor | undefined {
    const prev = this.entries.get(key);
    const next = state.executeSetter(value, prev);
    if (next == null || (prev != null && this.equal(next, prev, key))) return undefined;
    this.entries.set(key, next);
    this.notifySet(scope, key, next, variant);

    return () => {
      if (prev === undefined) {
        this.entries.delete(key);
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
    value?: state.SetArg<Value | undefined> | SetExtra,
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
        value as state.SetArg<Value | undefined>,
        extra as SetExtra,
      );
      if (rollback != null) rollbacks.push(rollback);
    }

    return () => rollbacks.reverse().forEach((r) => r());
  }

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

  /**
   * Deletes entries from the store and notifies delete listeners.
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

    toDelete.forEach(({ key: k }) => {
      this.entries.delete(k);
      this.notifyDelete(scope, k);
    });

    return () =>
      toDelete.forEach(({ key: k, value }) => {
        if (value == null) return;
        this.entries.set(k, value);
        this.notifySet(scope, k, value, variant as SetExtra);
      });
  }

  clear() {
    this.entries.clear();
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
        valueOrVariant?: state.SetArg<Value | undefined> | SetExtra,
        variant?: SetExtra,
      ): (() => void) => this.set(scope, key, valueOrVariant, variant),
      get: ((key: Key | Key[] | ((value: Value) => boolean)) =>
        this.get(key)) as UnaryStore<Key, Value>["get"],
      list: () => this.list(),
      has: (key: Key | Key[]) => this.has(key),
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
 * @template ScopedStore - The type of the store available to the listener
 * @template Schema - Zod schema type for validating channel data
 */
export interface ChannelListener<
  ScopedStore extends Store = {},
  Z extends z.ZodType = z.ZodType,
> {
  /** The name of the Synnax channel to listen to */
  channel: string;
  /** Zod schema for parsing and validating channel data */
  schema: Z;
  /** Callback function invoked when the channel data changes */
  onChange: (args: ChannelListenerArgs<ScopedStore, Z>) => Promise<unknown> | unknown;
}

/**
 * Arguments passed to a channel listener's onChange callback.
 *
 * @template ScopedStore - The type of the store available to the listener
 * @template Z - Zod schema type for validating channel data
 */
export type ChannelListenerArgs<
  ScopedStore extends Store = {},
  Z extends z.ZodType = z.ZodType,
> = {
  /** The parsed and validated data that changed */
  changed: z.output<Z>;
  /** The Synnax client instance for making additional API calls */
  client: Synnax;
  /** The store instance available to the listener */
  store: ScopedStore;
};

/**
 * Configuration for a single UnaryStore including its channel listeners.
 *
 * @template ScopedStore - The type of the scoped store
 */
export interface UnaryStoreConfig<
  ScopedStore extends Store = {},
  Key extends record.Key = record.Key,
  Value extends state.State = state.State,
> {
  /** Function to determine if two values are equal */
  equal?: (a: Value, b: Value, key: Key) => boolean;
  /** Array of channel listeners to register for this store */
  listeners: ChannelListener<ScopedStore>[];
}

/**
 * Configuration object for creating a store with multiple UnaryStore instances.
 * Keys are store names and values are their configurations.
 *
 * @template ScopedStore - The type of the scoped store
 */
export interface StoreConfig<ScopedStore extends Store = {}> {
  [key: string]: UnaryStoreConfig<ScopedStore, any, any>;
}

export type UnaryStore<
  Key extends record.Key = record.Key,
  Value extends state.State = state.State,
  SetExtra extends unknown | undefined = undefined,
> = {
  get(key: Key): Value | undefined;
  get(keys: Key[] | ((value: Value) => boolean)): Value[];
  list(): Value[];
  has(key: Key | Key[]): boolean;
  onSet(callback: SetHandler<Value, SetExtra>, key?: Key): destructor.Destructor;
  onDelete(callback: DeleteHandler<Key>, key?: Key): destructor.Destructor;
} & (narrow.IsUndefined<SetExtra> extends true
  ? {
      set(key: Key, value: state.SetArg<Value | undefined>): () => void;
      set(
        value: (Value & record.Keyed<Key>) | Array<Value & record.Keyed<Key>>,
      ): () => void;
      delete(key: Key | Key[] | ((value: Value, key: Key) => boolean)): () => void;
    }
  : {
      set(
        key: Key,
        value: state.SetArg<Value | undefined>,
        variant: SetExtra,
      ): () => void;
      set(
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
export interface Store {
  [key: string]: UnaryStore<any, any, undefined> | UnaryStore<string, any, unknown>;
}

export interface InternalStore {
  [key: string]: ScopedUnaryStore<string, state.State, string>;
}

/**
 * Creates a new store instance from the provided configuration.
 * Each key in the config becomes a UnaryStore in the resulting store.
 *
 * @template ScopedStore - The type of the store to create
 * @param config - Configuration object defining the store structure
 * @returns A new store instance with UnaryStore instances for each config key
 */
export const createStore = <ScopedStore extends Store>(
  config: StoreConfig<ScopedStore>,
  handleError: status.ErrorHandler,
): InternalStore =>
  Object.fromEntries(
    Object.entries(config).map(([key, { equal }]) => [
      key,
      new ScopedUnaryStore<string, state.State, string>(handleError, equal),
    ]),
  );

export const scopeStore = <ScopedStore extends Store>(
  store: InternalStore,
  scope: string,
): ScopedStore =>
  Object.fromEntries(
    Object.entries(store).map(([key]): [string, UnaryStore<string, any, unknown>] => [
      key,
      store[key].scope(scope),
    ]),
  ) as ScopedStore;

export const partialUpdate = <Key extends record.Key, Value extends Record<any, any>>(
  store: UnaryStore<Key, Value>,
  key: Key,
  value: Partial<Value>,
): destructor.Destructor =>
  store.set(key, (p) => (p == null ? undefined : { ...p, ...value }));
