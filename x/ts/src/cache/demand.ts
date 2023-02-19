// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { KV } from "@/kv";

export class DemandCache<K, V> implements KV<K, DemandCacheEntry<K, V>, K, V> {
  private readonly entries: Map<K, DemandCacheEntry<K, V>>;

  constructor() {
    this.entries = new Map();
  }

  get(key: K): DemandCacheEntry<K, V> | null {
    const entry = this.entries.get(key);
    if (entry == null) return null;
    entry.acquire();
    return entry;
  }

  set(key: K, value: V): void {
    this.entries.set(key, new DemandCacheEntry(key, value));
  }

  delete(key: K): void {
    this.entries.delete(key);
  }

  getDemandUnder(threshold: number = 1): Array<DemandCacheEntry<K, V>> {
    const entries: Array<DemandCacheEntry<K, V>> = [];
    this.entries.forEach((entry) => {
      if (entry.demand < threshold) {
        entries.push(entry);
      }
    });
    return entries;
  }
}

export class DemandCacheEntry<K, V> {
  public readonly key: K;
  public readonly value: V;
  public demand: number = 0;

  constructor(key: K, value: V) {
    this.key = key;
    this.value = value;
  }

  acquire(): void {
    this.demand++;
  }

  release(): void {
    this.demand--;
  }
}
