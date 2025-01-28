// Copyright 2024 Synnax Labs, Inc.
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
