// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, type destructor, primitive, type record } from "@synnaxlabs/x";
import { z } from "zod";

import { type batch } from "@/batch";
import { type Key, keyZ } from "@/group/types.gen";
import { query } from "@/query";

/** Retrieve request fields that filter records by the group they sit in. */
export const memberFilterZ = z.object({
  /** Keeps records that sit in any of these groups. */
  groups: keyZ.array().optional(),
  /** Keeps records that sit in a group when true, and records in none when false. */
  grouped: z.boolean().optional(),
});
export interface MemberFilter extends z.infer<typeof memberFilterZ> {}

/** A record that can sit in a group. */
export interface Member {
  group?: Key;
}

/** Returns a predicate that tells whether a record passes the filter. */
export const createMemberMatcher = (
  filter: MemberFilter,
): ((member: Member) => boolean) => {
  const groups = primitive.isNonZero(filter.groups) ? new Set(filter.groups) : null;
  const { grouped } = filter;
  return ({ group }) => {
    if (groups != null && (group == null || !groups.has(group))) return false;
    if (grouped != null && (group != null) !== grouped) return false;
    return true;
  };
};

/** Schema of a set-group request for records keyed by the given schema. */
export const moveReqZ = <K extends z.ZodType<record.Key>>(key: K) =>
  z.object({ keys: key.array(), group: keyZ.optional() });

/** Payload of a set-group request. A missing group moves records out of their group. */
export interface MovePayload<K extends record.Key> {
  keys: K[];
  group?: Key;
}

/**
 * Builds a request that moves the records into the group, or out of their group when it
 * is null or undefined. Its cache write moves the cached records.
 */
export interface SetGroupRequest<K extends record.Key> {
  (keys: K | K[], to?: Key | null): batch.Request<MovePayload<K>>;
}

/**
 * Returns a builder of set-group requests to the endpoint. A request's cache write
 * calls move with its keys and group, and move returns the rollback.
 */
export const createSetGroupRequest =
  <K extends record.Key>(
    endpoint: string,
    move: (keys: K[], group?: Key) => destructor.Destructor,
  ): SetGroupRequest<K> =>
  (keys, to) => {
    const payload = { keys: array.toArray(keys), group: to ?? undefined };
    return { endpoint, payload, apply: () => move(payload.keys, payload.group) };
  };

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
