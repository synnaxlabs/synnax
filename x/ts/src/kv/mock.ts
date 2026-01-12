// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Async, type KV } from "@/kv/types";

/**
 * A mock implementation of an async key-value store that keeps all data in a map
 * in memory.
 */
export class MockAsync implements Async {
  readonly store: Map<string, unknown> = new Map();

  async get<V>(key: string): Promise<V | null> {
    return (this.store.get(key) as V) ?? null;
  }

  async set<V>(key: string, value: V): Promise<void> {
    this.store.set(key, value);
  }

  async delete(key: string): Promise<void> {
    this.store.delete(key);
  }

  async length(): Promise<number> {
    return this.store.size;
  }

  async clear(): Promise<void> {
    this.store.clear();
  }
}

/**
 * A mock implementation of a sync key-value store that keeps all data in a map
 * in memory.
 */
export class MockSync implements KV {
  readonly store: Map<string, unknown> = new Map();

  get<V>(key: string): V | null {
    return (this.store.get(key) as V) ?? null;
  }

  set<V>(key: string, value: V): void {
    this.store.set(key, value);
  }

  delete(key: string): void {
    this.store.delete(key);
  }
}
