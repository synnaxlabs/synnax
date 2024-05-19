// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

interface API<V> {
  get: (key: string) => Promise<string>;
  set: (any: V) => Promise<void>;
  delete: () => Promise<void>;
}

const KV_API_KEY = "kvAPI";

/**
 * TauriKV an implementation of AsyncKV that communicates with a rust key-value
 * store running on the backend.
 */
export class ElectronKV {
  private readonly store: API;

  constructor() {
    if (!(KV_API_KEY in window)) {
      throw new Error("ElectronKV API not found.");
    }
    this.store = (window as { [KV_API_KEY]: API })[KV_API_KEY];
  }

  async get<V>(key: string): Promise<V | null> {
    return (await this.store.get(key)) as V | null;
  }

  async set<V>(key: string, value: V): Promise<void> {
    (await this.store.set(key, value)) as V;
  }

  async delete(key: string): Promise<void> {
    await this.store.delete(key);
  }
}
