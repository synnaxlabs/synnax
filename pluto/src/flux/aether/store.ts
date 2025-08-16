import { type channel, type Synnax } from "@synnaxlabs/client";
import { type Destructor, type observe, type record } from "@synnaxlabs/x";
import type z from "zod";

import { state } from "@/state";

interface SetOptions {
  notify?: boolean;
}

export class UnaryStore<
  K extends record.Key = record.Key,
  V extends state.State = state.State,
> {
  private entries: Map<K, V> = new Map();
  private setListeners: Map<observe.AsyncHandler<V>, K | undefined> = new Map();
  private deleteListeners: Map<observe.AsyncHandler<K>, K | undefined> = new Map();

  set(key: K, value: state.SetArg<V | undefined>, opts: SetOptions = {}): void {
    const { notify = true } = opts;
    const prev = this.entries.get(key);
    const next = state.executeSetter(value, prev);
    if (next == null) return;
    this.entries.set(key, next);
    if (notify) this.notifySet(key, next);
  }

  get(key: K): V | undefined;
  get(filter: (value: V) => boolean): V[];
  get(keys: K[]): V[];

  get(keys: K | K[] | ((value: V) => boolean)): V | V[] | undefined {
    if (typeof keys === "function")
      return Array.from(this.entries.values()).filter(keys);
    if (Array.isArray(keys))
      return keys.map((key) => this.entries.get(key)).filter((e) => e != null) as V[];
    return this.entries.get(keys);
  }

  delete(key: K) {
    this.entries.delete(key);
    this.notifyDelete(key);
  }

  onSet(callback: observe.AsyncHandler<V>, key?: K): Destructor {
    this.setListeners.set(callback, key);
    return () => this.setListeners.delete(callback);
  }

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

export interface ChannelListener<
  ScopedStore extends Store = {},
  Z extends z.ZodType = z.ZodType,
> {
  channel: channel.Name;
  schema: Z;
  onChange: (args: ChannelListenerArgs<ScopedStore, Z>) => Promise<void> | void;
}

export type ChannelListenerArgs<
  ScopedStore extends Store = {},
  Z extends z.ZodType = z.ZodType,
> = {
  changed: z.output<Z>;
  client: Synnax;
  store: ScopedStore;
};

export interface UnaryStoreConfig<ScopedStore extends Store = {}> {
  listeners: ChannelListener<ScopedStore>[];
}

export interface StoreConfig<ScopedStore extends Store = {}> {
  [key: string]: UnaryStoreConfig<ScopedStore>;
}

export interface Store {
  [key: string]: UnaryStore<any, any>;
}

export const createStore = <ScopedStore extends Store>(
  config: StoreConfig<ScopedStore>,
): ScopedStore =>
  Object.fromEntries(
    Object.entries(config).map(([key]) => [key, new UnaryStore<string, state.State>()]),
  ) as ScopedStore;
