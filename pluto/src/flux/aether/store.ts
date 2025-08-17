import { type channel, type Synnax } from "@synnaxlabs/client";
import { array, type Destructor, type observe, type record } from "@synnaxlabs/x";
import type z from "zod";

import { state } from "@/state";
import { type status } from "@/status/aether";

interface ListenerScope<K extends record.Key> {
  scope: string;
  key?: K;
}

type Handler<V> = observe.Handler<V> | observe.AsyncHandler<V>;

export class ScopedUnaryStore<
  K extends record.Key = record.Key,
  V extends state.State = state.State,
> {
  private readonly entries: Map<K, V> = new Map();
  private readonly setListeners: Map<Handler<V>, ListenerScope<K>> = new Map();
  private readonly deleteListeners: Map<Handler<K>, ListenerScope<K>> = new Map();
  private readonly handleError: status.ErrorHandler;
  private readonly equal: (a: V, b: V, key: K) => boolean;

  constructor(
    handleError: status.ErrorHandler,
    equal: (a: V, b: V, key: K) => boolean = () => false,
  ) {
    this.handleError = handleError;
    this.equal = equal;
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
    key: K | Array<V & record.Keyed<K>>,
    value?: state.SetArg<V | undefined>,
  ): void {
    if (Array.isArray(key)) return key.forEach((v) => this.set(scope, v.key, v));
    const prev = this.entries.get(key);
    const next = state.executeSetter(value, prev);
    if (next == null || (prev != null && this.equal(next, prev, key))) return;
    this.entries.set(key, next);
    this.notifySet(scope, key, next);
  }

  get(keys: K | K[] | ((value: V) => boolean)): V | V[] | undefined {
    if (typeof keys === "function")
      return Array.from(this.entries.values()).filter(keys);
    if (Array.isArray(keys))
      return keys.map((key) => this.entries.get(key)).filter((e) => e != null) as V[];
    return this.entries.get(keys);
  }

  /**
   * Deletes an entry from the store and notifies delete listeners.
   * @param key - The key to delete
   */
  delete(scope: string, key: K | K[]) {
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
  onSet(
    scope: string,
    callback: observe.AsyncHandler<V> | observe.Handler<V>,
    key?: K,
  ): Destructor {
    this.setListeners.set(callback, { scope, key });
    return () => {
      this.setListeners.delete(callback);
    };
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
    callback: observe.AsyncHandler<K> | observe.Handler<K>,
    key?: K,
  ): Destructor {
    this.deleteListeners.set(callback, { scope, key });
    return () => this.deleteListeners.delete(callback);
  }

  private notifySet(scope: string, key: K, value: V) {
    this.setListeners.forEach((listenerKey, callback) => {
      const matchesKey = listenerKey.key == null || listenerKey.key === key;
      const matchesScope = listenerKey.scope !== scope;
      if (matchesKey && matchesScope)
        this.handleError(async () => callback(value), "Failed to notify set listener");
    });
  }

  private notifyDelete(scope: string, key: K) {
    this.deleteListeners.forEach((listenerKey, callback) => {
      const matchesKey = listenerKey.key == null || listenerKey.key === key;
      const matchesScope = listenerKey.scope !== scope;
      if (matchesKey && matchesScope)
        this.handleError(async () => callback(key), "Failed to notify delete listener");
    });
  }

  scope(scope: string): UnaryStore<K, V> {
    return {
      set: (key: K | Array<V & record.Keyed<K>>, value?: state.SetArg<V | undefined>) =>
        this.set(scope, key, value),
      get: ((key: K | K[] | ((value: V) => boolean)) => this.get(key)) as UnaryStore<
        K,
        V
      >["get"],
      delete: (key: K | K[]) => this.delete(scope, key),
      onSet: (callback: observe.AsyncHandler<V> | observe.Handler<V>, key?: K) =>
        this.onSet(scope, callback, key),
      onDelete: (callback, key) => this.onDelete(scope, callback, key),
    };
  }
}

/**
 * Configuration for listening to changes on a specific Synnax channel.
 *
 * @template ScopedStore - The type of the store available to the listener
 * @template Z - Zod schema type for validating channel data
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
  K extends record.Key = record.Key,
  V extends state.State = state.State,
> {
  /** Function to determine if two values are equal */
  equal?: (a: V, b: V, key: K) => boolean;
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

export interface UnaryStore<
  K extends record.Key = record.Key,
  V extends state.State = state.State,
> {
  set(values: Array<V & record.Keyed<K>>): void;
  set(key: K, value: state.SetArg<V | undefined>): void;
  get(key: K): V | undefined;
  get(keys: K[] | ((value: V) => boolean)): V[];
  delete(key: K | K[]): void;
  onSet(callback: observe.AsyncHandler<V> | observe.Handler<V>, key?: K): Destructor;
  onDelete(callback: observe.AsyncHandler<K> | observe.Handler<K>, key?: K): Destructor;
}

/**
 * Base interface for a collection of UnaryStore instances.
 * Each property is a UnaryStore with its own key-value type.
 */
export interface Store {
  [key: string]: UnaryStore<any, any>;
}

export interface InternalStore {
  [key: string]: ScopedUnaryStore<string, state.State>;
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
      new ScopedUnaryStore<string, state.State>(handleError, equal),
    ]),
  );

export const scopeStore = <ScopedStore extends Store>(
  store: InternalStore,
  scope: string,
): ScopedStore =>
  Object.fromEntries(
    Object.entries(store).map(([key]): [string, UnaryStore<any, any>] => [
      key,
      store[key].scope(scope),
    ]),
  ) as ScopedStore;
