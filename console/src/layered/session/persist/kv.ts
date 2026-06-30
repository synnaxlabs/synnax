// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type kv } from "@synnaxlabs/x";
import { LazyStore } from "@tauri-apps/plugin-store";

import { Runtime } from "@/runtime";

/**
 * A SugaredKV is a spiced up key-value store that provides a few extra goodies needed
 * for efficient persistence.
 */
export interface SugaredKV extends kv.Async {
  /** Get the number of key-value pairs in the store. */
  length(): Promise<number>;
  /** Clear the store of all key-value pairs. */
  clear(): Promise<void>;
}

class TauriKV implements SugaredKV {
  private store: LazyStore;

  constructor(store: LazyStore) {
    this.store = store;
  }

  async get<V>(key: string): Promise<V | null> {
    return (await this.store.get(key)) as V;
  }

  async set<V>(key: string, value: V): Promise<void> {
    await this.store.set(key, value);
    await this.store.save();
  }

  async delete(key: string): Promise<void> {
    await this.store.delete(key);
  }

  async length(): Promise<number> {
    return await this.store.length();
  }

  async clear(): Promise<void> {
    await this.store.clear();
  }
}

class LocalStorageKV implements SugaredKV {
  private store: Storage;
  baseKey: string;

  constructor(baseKey: string) {
    this.baseKey = baseKey;
    this.store = localStorage;
  }

  async get<V>(key: string): Promise<V | null> {
    const item = this.store.getItem(`${this.baseKey}:${key}`);
    return item ? JSON.parse(item) : null;
  }

  async set<V>(key: string, value: V): Promise<void> {
    this.store.setItem(`${this.baseKey}:${key}`, JSON.stringify(value));
  }

  async delete(key: string): Promise<void> {
    this.store.removeItem(`${this.baseKey}:${key}`);
  }

  async length(): Promise<number> {
    let count = 0;
    for (let i = 0; i < this.store.length; i++) {
      const key = this.store.key(i);
      if (key && key.startsWith(`${this.baseKey}:`)) count++;
    }
    return count;
  }

  async clear(): Promise<void> {
    const keysToRemove: string[] = [];
    for (let i = 0; i < this.store.length; i++) {
      const key = this.store.key(i);
      if (key && key.startsWith(`${this.baseKey}:`)) keysToRemove.push(key);
    }
    keysToRemove.forEach((key) => this.store.removeItem(key));
  }
}

/**
 * Open a new SugaredKV instance.
 * @param dir - The directory to store the key-value store in.
 * @returns A new SugaredKV instance.
 */
export const openSugaredKV = (dir: string): SugaredKV =>
  Runtime.ENGINE === "tauri"
    ? new TauriKV(new LazyStore(dir, { autoSave: true, defaults: {} }))
    : new LocalStorageKV(dir);
