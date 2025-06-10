// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type kv } from "@synnaxlabs/x";
import { LazyStore } from "@tauri-apps/plugin-store";

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

/**
 * TauriKV an implementation of SugaredKV that communicates with a rust key-value store
 * running on the backend.
 */
export class TauriKV implements SugaredKV {
  store: LazyStore;

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

/**
 * Open a new SugaredKV instance.
 * @param dir - The directory to store the key-value store in.
 * @returns A new SugaredKV instance.
 */
export const openTauriKV = async (dir: string): Promise<SugaredKV> =>
  new TauriKV(new LazyStore(dir, { autoSave: true }));

/**
 * LocalStorageKV is an implementation of SugaredKV that uses the browser's localStorage API
 * for persistence. This implementation is synchronous under the hood but implements the
 * async interface for compatibility.
 */
export class LocalStorageKV implements SugaredKV {
  private prefix: string;

  constructor(prefix: string = "") {
    this.prefix = prefix;
  }

  private getKey(key: string): string {
    return `${this.prefix}${key}`;
  }

  async get<V>(key: string): Promise<V | null> {
    const value = localStorage.getItem(this.getKey(key));
    if (value === null) return null;
    try {
      return JSON.parse(value) as V;
    } catch {
      return null;
    }
  }

  async set<V>(key: string, value: V): Promise<void> {
    localStorage.setItem(this.getKey(key), JSON.stringify(value));
  }

  async delete(key: string): Promise<void> {
    localStorage.removeItem(this.getKey(key));
  }

  async length(): Promise<number> {
    if (!this.prefix) return localStorage.length;
    return Object.keys(localStorage).filter((key) => key.startsWith(this.prefix))
      .length;
  }

  async clear(): Promise<void> {
    if (!this.prefix) {
      localStorage.clear();
      return;
    }
    Object.keys(localStorage)
      .filter((key) => key.startsWith(this.prefix))
      .forEach((key) => localStorage.removeItem(key));
  }
}

/**
 * Opens a new LocalStorageKV instance.
 * @param prefix - Optional prefix for all keys to prevent naming collisions
 * @returns A new LocalStorageKV instance
 */
export const openLocalStorageKV = (prefix: string = ""): SugaredKV =>
  new LocalStorageKV(prefix);
