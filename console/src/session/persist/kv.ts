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

import { Runtime } from "@/session/runtime";

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
    await this.store.save();
  }

  async length(): Promise<number> {
    return await this.store.length();
  }

  async clear(): Promise<void> {
    await this.store.clear();
    await this.store.save();
  }
}

const OBJECT_STORE = "kv";

/**
 * Browser-side store. IndexedDB rather than localStorage: a session holds up to four
 * slots for each of its three partitions, which outgrows localStorage's few-megabyte
 * cap on a heavy workspace, and its quota errors surface only as a failed write.
 */
class IndexedDBKV implements SugaredKV {
  private readonly name: string;
  private db: Promise<IDBDatabase> | null = null;

  constructor(name: string) {
    this.name = name;
  }

  async get<V>(key: string): Promise<V | null> {
    return await this.run("readonly", (store) => store.get(key) as IDBRequest<V>).then(
      (value) => value ?? null,
    );
  }

  async set<V>(key: string, value: V): Promise<void> {
    await this.run("readwrite", (store) => store.put(value, key));
  }

  async delete(key: string): Promise<void> {
    await this.run("readwrite", (store) => store.delete(key));
  }

  async length(): Promise<number> {
    return await this.run("readonly", (store) => store.count());
  }

  async clear(): Promise<void> {
    await this.run("readwrite", (store) => store.clear());
  }

  private open(): Promise<IDBDatabase> {
    this.db ??= new Promise<IDBDatabase>((resolve, reject) => {
      const request = indexedDB.open(this.name, 1);
      request.onupgradeneeded = () => request.result.createObjectStore(OBJECT_STORE);
      request.onsuccess = () => resolve(request.result);
      request.onerror = () =>
        reject(new Error(`failed to open ${this.name}`, { cause: request.error }));
    });
    return this.db;
  }

  private async run<V>(
    mode: IDBTransactionMode,
    op: (store: IDBObjectStore) => IDBRequest<V>,
  ): Promise<V> {
    const db = await this.open();
    return await new Promise<V>((resolve, reject) => {
      const request = op(db.transaction(OBJECT_STORE, mode).objectStore(OBJECT_STORE));
      request.onsuccess = () => resolve(request.result);
      request.onerror = () =>
        reject(new Error(`${this.name} operation failed`, { cause: request.error }));
    });
  }
}

/**
 * @param name - The store's name, without an extension.
 * @returns A new SugaredKV instance.
 */
export const openSugaredKV = (name: string): SugaredKV =>
  Runtime.ENGINE === "tauri"
    ? new TauriKV(new LazyStore(`${name}.json`, { autoSave: false, defaults: {} }))
    : new IndexedDBKV(`synnax-${name}`);
