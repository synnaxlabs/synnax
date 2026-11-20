// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, type patch, type record } from "@synnaxlabs/x";

import type { actions } from "@/actions";
import type { batch } from "@/batch";
import type { Key } from "@/group/types.gen";

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
 * when the group is null or undefined.
 */
export interface MoveRequest<K extends record.Key> {
  (keys: K | K[], to?: Key | null): batch.Request<MovePayload<K>>[];
}

/**
 * Builds the batch entry that sends actions to one record, such as
 * `actions.Controller.request` bound to the dispatch endpoint of its type.
 */
export interface EntryBuilder<K extends record.Key> {
  (key: K, actions: patch.Action[]): batch.Request<MovePayload<K>>;
}

/**
 * Returns the actions that move a record into the group, or out of its group when the
 * group is null or undefined.
 */
export const moveActions = (to?: Key | null): patch.Action[] => [
  { type: "set", set: { path: ["group"], value: to ?? null, guards: [] } },
];

/** Returns a builder of move entries that builds each entry with build. */
export const createMoveRequest =
  <K extends record.Key>(build: EntryBuilder<K>): MoveRequest<K> =>
  (keys, to) =>
    array.toArray(keys).map((key) => build(key, moveActions(to)));
