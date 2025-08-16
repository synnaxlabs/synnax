import { type channel, type Synnax } from "@synnaxlabs/client";
import { type Destructor, type observe, type record } from "@synnaxlabs/x";
import type z from "zod";

import { state } from "@/state";

/**
 * Options for setting values in the store.
 */
interface SetOptions {
  /** Whether to notify listeners of the change. Defaults to true. */
  notify?: boolean;
}

/**
 * A key-value store that manages state with reactive update notifications.
 * Supports setting, getting, and deleting entries while notifying registered listeners.
 *
 * @template K - The type of keys used in the store
 * @template V - The type of values stored, must extend state.State
 */
export class UnaryStore<
  K extends record.Key = record.Key,
  V extends state.State = state.State,
> {
  private entries: Map<K, V> = new Map();
  private setListeners: Map<observe.AsyncHandler<V>, K | undefined> = new Map();
  private deleteListeners: Map<observe.AsyncHandler<K>, K | undefined> = new Map();

  /**
   * Sets a value for the given key in the store.
   *
   * @param key - The key to set
   * @param value - The value to set, or a function to compute the value from the previous state
   * @param opts - Options for the set operation
   */
  set(key: K, value: state.SetArg<V | undefined>, opts: SetOptions = {}): void {
    const { notify = true } = opts;
    const prev = this.entries.get(key);
    const next = state.executeSetter(value, prev);
    if (next == null) return;
    this.entries.set(key, next);
    if (notify) this.notifySet(key, next);
  }

  /**
   * Gets a value from the store by key.
   * @param key - The key to retrieve
   * @returns The value associated with the key, or undefined if not found
   */
  get(key: K): V | undefined;
  /**
   * Gets values from the store matching a filter predicate.
   * @param filter - A function to test each value
   * @returns An array of matching values
   */
  get(filter: (value: V) => boolean): V[];
  /**
   * Gets multiple values from the store by their keys.
   * @param keys - An array of keys to retrieve
   * @returns An array of values (undefined values are filtered out)
   */
  get(keys: K[]): V[];

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
  delete(key: K) {
    this.entries.delete(key);
    this.notifyDelete(key);
  }

  /**
   * Registers a listener for set operations.
   *
   * @param callback - Function to call when a value is set
   * @param key - Optional key to filter notifications (if provided, only changes to this key trigger the callback)
   * @returns A destructor function to remove the listener
   */
  onSet(callback: observe.AsyncHandler<V>, key?: K): Destructor {
    this.setListeners.set(callback, key);
    return () => this.setListeners.delete(callback);
  }

  /**
   * Registers a listener for delete operations.
   *
   * @param callback - Function to call when a value is deleted
   * @param key - Optional key to filter notifications (if provided, only deletion of this key triggers the callback)
   * @returns A destructor function to remove the listener
   */
  onDelete(callback: observe.AsyncHandler<K>, key?: K): Destructor {
    this.deleteListeners.set(callback, key);
    return () => {
      this.deleteListeners.delete(callback);
    };
  }

  private notifySet(key: K, value: V) {
    this.setListeners.forEach((listenerKey, callback) => {
      if (listenerKey == null || listenerKey === key) void callback(value);
    });
  }

  private notifyDelete(key: K) {
    this.deleteListeners.forEach((listenerKey, callback) => {
      if (listenerKey == null || listenerKey === key) void callback(key);
    });
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
export interface UnaryStoreConfig<ScopedStore extends Store = {}> {
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
  [key: string]: UnaryStoreConfig<ScopedStore>;
}

/**
 * Base interface for a collection of UnaryStore instances.
 * Each property is a UnaryStore with its own key-value type.
 */
export interface Store {
  [key: string]: UnaryStore<any, any>;
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
): ScopedStore =>
  Object.fromEntries(
    Object.entries(config).map(([key]) => [key, new UnaryStore<string, state.State>()]),
  ) as ScopedStore;
