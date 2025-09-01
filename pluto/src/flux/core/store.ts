// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type Synnax } from "@synnaxlabs/client";
import {
  array,
  type Destructor,
  type IsExactlyUndefined,
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
  ): void {
    const prev = this.entries.get(key);
    const next = state.executeSetter(value, prev);
    if (next == null || (prev != null && this.equal(next, prev, key))) return;
    this.entries.set(key, next);
    this.notifySet(scope, key, next, variant);
  }

  /**
   * Sets a value for the given key in the store.
   *
   * @param key - The key to set
   * @param value - The value to set, or a function to compute the value from the previous state
   * @param opts - Options for the set operation
   */
  set(
    scope: string,
    key: Key | Array<Value & record.Keyed<Key>>,
    value?: state.SetArg<Value | undefined> | SetExtra,
    extra?: SetExtra,
  ): void {
    if (Array.isArray(key))
      return key.forEach((val) => this.setOne(scope, val.key, val, value as SetExtra));
    this.setOne(
      scope,
      key,
      value as state.SetArg<Value | undefined>,
      extra as SetExtra,
    );
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

  has(key: Key): boolean {
    return this.entries.has(key);
  }

  /**
   * Deletes an entry from the store and notifies delete listeners.
   * @param key - The key to delete
   */
  delete(scope: string, key: Key | Key[]) {
    array.toArray(key).forEach((k) => {
      this.entries.delete(k);
      this.notifyDelete(scope, k);
    });
  }

  clear() {
    this.entries.clear();
  }

  /**
   * Registers a listener for set operations.
   *
   * @param callback - Function to call when a value is set
   * @param key - Optional key to filter notifications (if provided, only changes to this key trigger the callback)
   * @returns A destructor function to remove the listener
   */
  onSet(scope: string, callback: SetHandler<Value, SetExtra>, key?: Key): Destructor {
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
  ): Destructor {
    this.deleteListeners.set(callback, { scope, key });
    return () => this.deleteListeners.delete(callback);
  }

  private notifySet(scope: string, key: Key, value: Value, variant: SetExtra) {
    this.setListeners.forEach((listenerKey, callback) => {
      const matchesKey = listenerKey.key == null || listenerKey.key === key;
      const matchesScope = listenerKey.scope !== scope;
      if (matchesKey && matchesScope)
        this.handleError(
          async () => callback(value, variant),
          "Failed to notify set listener",
        );
    });
  }

  private notifyDelete(scope: string, key: Key) {
    this.deleteListeners.forEach((listenerKey, callback) => {
      const matchesKey = listenerKey.key == null || listenerKey.key === key;
      const matchesScope = listenerKey.scope !== scope;
      if (matchesKey && matchesScope)
        this.handleError(async () => callback(key), "Failed to notify delete listener");
    });
  }

  scope(scope: string): UnaryStore<Key, Value, SetExtra> {
    return {
      set: (
        key: Key | Array<Value & record.Keyed<Key>>,
        valueOrVariant?: state.SetArg<Value | undefined> | SetExtra,
        variant?: SetExtra,
      ) => this.set(scope, key, valueOrVariant, variant),
      get: ((key: Key | Key[] | ((value: Value) => boolean)) =>
        this.get(key)) as UnaryStore<Key, Value>["get"],
      list: () => this.list(),
      has: (key: Key) => this.has(key),
      delete: (key: Key | Key[]) => this.delete(scope, key),
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
  channel: channel.Name;
  /** Zod schema for parsing and validating channel data */
  schema: Z;
  /** Callback function invoked when the channel data changes */
  onChange: (args: ChannelListenerArgs<ScopedStore, Z>) => Promise<void> | void;
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
  has(key: Key): boolean;
  delete(key: Key | Key[]): void;
  onSet(callback: SetHandler<Value, SetExtra>, key?: Key): Destructor;
  onDelete(callback: DeleteHandler<Key>, key?: Key): Destructor;
} & (IsExactlyUndefined<SetExtra> extends true
  ? {
      set(key: Key, value: state.SetArg<Value | undefined>): void;
      set(values: Array<Value & record.Keyed<Key>>): void;
    }
  : {
      set(key: Key, value: state.SetArg<Value | undefined>, variant: SetExtra): void;
      set(values: Array<Value & record.Keyed<Key>>, variant: SetExtra): void;
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
