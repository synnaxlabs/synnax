// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, type destructor, type patch, type record } from "@synnaxlabs/x";

import { type actions } from "@/actions";
import { type batch } from "@/batch";
import { type Key } from "@/group/types.gen";
import { query } from "@/query";

/** A record that can sit in a group. */
export interface Member {
  group?: Key;
}

/** Checks the groups and grouped lines of a Where against a record in the cache. */
export const MEMBER_LINES = {
  groups: ({ group }: Member, keys: Key[]): boolean =>
    group != null && keys.includes(group),
  grouped: ({ group }: Member, grouped: boolean): boolean =>
    (group != null) === grouped,
};

/**
 * Payload of a dispatch entry that moves one record. Its action sets the group of the
 * record, where null moves the record out of its group.
 */
export interface MovePayload<K extends record.Key> extends actions.DispatchPayload<
  K,
  patch.Action
> {}

/**
 * Builds one batch entry per record that moves it into the group, or out of its group
 * when the group is null or undefined. The cache write of each entry moves its cached
 * record.
 */
export interface MoveRequest<K extends record.Key> {
  (keys: K | K[], to?: Key | null): batch.Request<MovePayload<K>>[];
}

/**
 * Returns a builder of move entries for the dispatch endpoint of a record type. The
 * cache write of an entry calls move with its key and group, and move returns the
 * rollback.
 */
export const createMoveRequest =
  <K extends record.Key>(
    endpoint: string,
    move: (keys: K[], group?: Key) => destructor.Destructor,
  ): MoveRequest<K> =>
  (keys, to) =>
    array.toArray(keys).map((key) => ({
      endpoint,
      payload: {
        key,
        actions: [
          { type: "set", set: { path: ["group"], value: to ?? null, guards: [] } },
        ],
      },
      apply: () => move([key], to ?? undefined),
    }));

/**
 * Sets the group of each cached record under keys, skipping keys the table does not
 * hold. Returns a rollback that restores only the group field.
 */
export const moveCached = <K extends record.Key, V extends Member>(
  table: query.Table<K, V>,
  keys: K[],
  group?: Key,
): destructor.Destructor => {
  const prev = keys.map((key) => [key, table.get(key)?.group] as const);
  keys.forEach((key) => query.partialUpdate(table, key, { group } as Partial<V>));
  return () =>
    prev.forEach(([key, g]) =>
      query.partialUpdate(table, key, { group: g } as Partial<V>),
    );
};
