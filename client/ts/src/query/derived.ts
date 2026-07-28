// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor, type record, type state } from "@synnaxlabs/x";

import { type Table, type TableEvent } from "@/query/table";

/**
 * A foreign-table trigger for a derived table: projects that table's events
 * onto the source keys whose composition they change. Build with
 * {@link deriveWatch}.
 */
export interface DeriveWatch<K extends record.Key> {
  attach: (recompose: (keys: K[] | "all") => void) => destructor.Destructor;
}

/**
 * Declares that events on the given table change composed rows. `affects`
 * maps an event to the source keys whose composition it touches, "all" to
 * recompose every row, or null when unaffected.
 */
export const deriveWatch = <
  K extends record.Key,
  FK extends record.Key,
  FV extends state.State,
>(
  table: Table<FK, FV>,
  affects: (event: TableEvent<FK, FV>) => K[] | "all" | null,
): DeriveWatch<K> => ({
  attach: (recompose) =>
    table.subscribe((event) => {
      const result = affects(event);
      if (result == null) return;
      if (result === "all" || result.length > 0) recompose(result);
    }),
});

/** Derivation wiring shared by {@link Cache.derive}. */
export interface DeriveParams<
  K extends record.Key,
  V extends state.State & record.Keyed<K>,
  CV extends state.State & record.Keyed<K>,
> {
  /** The table owning the raw records the derivation reads. */
  source: Table<K, V>;
  /** Builds the composed row for one source record. Pure; no network. */
  compose: (record: V) => CV;
  /** Foreign tables whose events change composition. */
  watch?: Array<DeriveWatch<K>>;
}

/**
 * Keeps a derived table materialized from its source: composed rows are
 * replaced (never mutated) on source and watch events, so a row's identity
 * changes exactly when its composition changes, with equal recompositions
 * silenced by the table's equality check. Existing source rows are composed
 * immediately. Returns a destructor that detaches every subscription.
 */
export const bindDerived = <
  K extends record.Key,
  V extends state.State & record.Keyed<K>,
  CV extends state.State & record.Keyed<K>,
>(
  into: Table<K, CV>,
  { source, compose, watch }: DeriveParams<K, V, CV>,
): destructor.Destructor => {
  const recompose = (keys: K[] | "all"): void => {
    const targets = keys === "all" ? source.get() : source.get(keys);
    if (targets.length > 0) into.set(targets.map(compose));
  };
  const detach = [
    source.subscribe((event) => {
      if (event.variant === "set") into.set(compose(event.value));
      else into.delete(event.key);
    }),
    ...(watch ?? []).map((w) => w.attach(recompose)),
  ];
  recompose("all");
  return () => detach.forEach((d) => d());
};
