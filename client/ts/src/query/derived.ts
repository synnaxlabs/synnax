// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor, type record, type state } from "@synnaxlabs/x";

import { type Keyed, type Table, type TableEvent } from "@/query/table";

/**
 * A foreign-table trigger for a derived table: projects that table's events onto the
 * source keys whose composition they change. Build with {@link deriveWatch}.
 */
export interface DeriveWatch<K extends record.Key> {
  attach: (recompose: (keys: K[] | "all") => void) => destructor.Destructor;
}

/**
 * Declares that events on the given table change composed entries. `affects`
 * maps an event to the source keys whose composition it touches, "all" to
 * recompose every entry, or null when unaffected.
 */
export const deriveWatch = <
  K extends record.Key,
  ForeignKey extends record.Key,
  ForeignValue extends state.State,
>(
  table: Table<ForeignKey, ForeignValue>,
  affects: (event: TableEvent<ForeignKey, ForeignValue>) => K[] | "all" | null,
): DeriveWatch<K> => ({
  attach: (recompose) =>
    // Batched so one foreign write recomposes once: an "all" verdict for any event
    // supersedes the batch's keys, which otherwise union.
    table.subscribeBatch((events) => {
      let keys: K[] = [];
      for (const event of events) {
        const result = affects(event);
        if (result == null) continue;
        if (result === "all") return recompose("all");
        keys = keys.concat(result);
      }
      if (keys.length > 0) recompose(keys);
    }),
});

/** Derivation wiring shared by {@link Cache.derive}. */
export interface DeriveParams<
  K extends record.Key,
  V extends Keyed<K>,
  CV extends Keyed<K>,
> {
  /** The table owning the raw records the derivation reads. */
  source: Table<K, V>;
  /** Builds the composed entry for one source record. Pure; no network. */
  compose: (record: V) => CV;
  /** Foreign tables whose events change composition. */
  watch?: Array<DeriveWatch<K>>;
}

/**
 * Keeps a derived table materialized from its source: composed entries are
 * replaced (never mutated) on source and watch events, so an entry's identity
 * changes exactly when its composition changes, with equal recompositions
 * silenced by the table's equality check. Existing source entries are composed
 * immediately. Returns a destructor that detaches every subscription.
 */
export const bindDerived = <
  K extends record.Key,
  V extends Keyed<K>,
  CV extends Keyed<K>,
>(
  into: Table<K, CV>,
  { source, compose, watch }: DeriveParams<K, V, CV>,
): destructor.Destructor => {
  const recompose = (keys: K[] | "all"): void => {
    const targets = keys === "all" ? source.get() : source.get(keys);
    if (targets.length > 0) into.set(targets.map(compose));
  };
  const detach = [
    // Batched so a source batch flushes the derived table once, preserving per-key
    // set/delete order within the batch.
    source.subscribeBatch((events) =>
      into.batch(() =>
        events.forEach((event) => {
          if (event.variant === "set") into.set(compose(event.value));
          // A source that keeps no corpse never held the record: mirroring that keeps a
          // derived tombstone from outliving a deletion that never was.
          else if (source.status(event.key) === "tombstoned") into.delete(event.key);
          else into.evict(event.key);
        }),
      ),
    ),
    ...(watch ?? []).map((w) => w.attach(recompose)),
  ];
  recompose("all");
  return () => detach.forEach((d) => d());
};
