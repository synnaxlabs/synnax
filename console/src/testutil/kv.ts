// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Session } from "@/session";

/** An in-memory SugaredKV. Nothing survives a reload. */
export class MemoryKV implements Session.Persist.SugaredKV {
  readonly store = new Map<string, unknown>();

  async get<V>(key: string): Promise<V | null> {
    return (this.store.get(key) as V) ?? null;
  }

  async set<V>(key: string, value: V): Promise<void> {
    this.store.set(key, value);
  }

  async setMany(entries: Session.Persist.Entry[]): Promise<void> {
    entries.forEach(({ key, value }) => this.store.set(key, value));
  }

  async delete(key: string): Promise<void> {
    await this.deleteMany([key]);
  }

  async deleteMany(keys: string[]): Promise<void> {
    keys.forEach((key) => this.store.delete(key));
  }

  async length(): Promise<number> {
    return this.store.size;
  }

  async keys(): Promise<string[]> {
    return Array.from(this.store.keys());
  }

  async clear(): Promise<void> {
    this.store.clear();
  }
}
