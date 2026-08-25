// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record, type state } from "@synnaxlabs/x";

/**
 * An exact-match secondary index over a table's live entries. `extract` reads one
 * index value from each entry. Register the index on its owning table's `indexes`
 * param; the table keeps it current across every mutation, rollbacks and resets
 * included. Tombstones are never indexed.
 */
export class LookupIndex<
  Key extends record.Key = record.Key,
  Value extends state.State = state.State,
  IndexValue extends record.Key = record.Key,
> {
  private readonly extract: (value: Value) => IndexValue;
  private readonly buckets = new Map<IndexValue, Map<Key, Value>>();
  private readonly indexed = new Map<Key, IndexValue>();

  constructor(extract: (value: Value) => IndexValue) {
    this.extract = extract;
  }

  /** Returns the entries whose extracted index value equals the given value. */
  get(value: IndexValue): Value[] {
    const bucket = this.buckets.get(value);
    if (bucket == null) return [];
    return Array.from(bucket.values());
  }

  /**
   * Records the entry, relocating it when its extracted value changed.
   * Cache-internal surface: called by the owning table, not domain code.
   */
  set(key: Key, value: Value): void {
    const next = this.extract(value);
    const prev = this.indexed.get(key);
    if (prev !== undefined && prev !== next) this.unbucket(prev, key);
    let bucket = this.buckets.get(next);
    if (bucket == null) {
      bucket = new Map();
      this.buckets.set(next, bucket);
    }
    bucket.set(key, value);
    this.indexed.set(key, next);
  }

  /**
   * Removes any mapping for the given entry key.
   * Cache-internal surface: called by the owning table, not domain code.
   */
  delete(key: Key): void {
    const prev = this.indexed.get(key);
    if (prev === undefined) return;
    this.unbucket(prev, key);
    this.indexed.delete(key);
  }

  /**
   * Discards every mapping.
   * Cache-internal surface: called by the owning table, not domain code.
   */
  reset(): void {
    this.buckets.clear();
    this.indexed.clear();
  }

  private unbucket(value: IndexValue, key: Key): void {
    const bucket = this.buckets.get(value);
    if (bucket == null) return;
    bucket.delete(key);
    if (bucket.size === 0) this.buckets.delete(value);
  }
}
