// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { primitive, type record, zod } from "@synnaxlabs/x";
import { z } from "zod";

import { type Type, typeZ } from "@/resource/types.gen";

/** Zod schema parsing a `"type:key"` string into an {@link ID}. */
export const stringIDZ = z.string().transform((v, ctx) => {
  const [type, key] = v.split(":");
  const res = typeZ.safeParse(type);
  if (!res.success) {
    ctx.addIssue({ code: "custom", message: `invalid resource type: ${type}` });
    return z.NEVER;
  }
  return { type: res.data, key: key ?? "" };
});

/** Zod schema for {@link ID}, accepting the object or the `"type:key"` string. */
export const idZ = z.object({ type: typeZ, key: z.string() }).or(stringIDZ);

/** Names one resource: what kind it is, and which one. */
export type ID = z.infer<typeof idZ>;

/** Builds {@link ID}s of one resource type, singly or in bulk. */
export interface CreateID<K extends record.Key> {
  (key: K): ID;
  (keys: K[]): ID[];
  (keys: K | K[]): ID | ID[];
}

/**
 * @returns a {@link CreateID} that stamps the given resource type onto keys. Each
 * resource package exports one as `resourceID`.
 *
 * @example const resourceID = createIDFactory<Key>("channel");
 */
export const createIDFactory = <K extends record.Key>(type: Type): CreateID<K> => {
  const id = (key: K) => ({ type, key: primitive.isZero(key) ? "" : key.toString() });
  return ((key: K | K[]) => {
    if (Array.isArray(key)) return key.map(id);
    return id(key);
  }) as CreateID<K>;
};

export interface IDToString {
  (id: ID | string): string;
  (ids: (ID | string)[]): string[];
}

/** Renders an {@link ID} as `"type:key"`, the form used as a map key. */
export const idToString = ((id: ID | string | (ID | string)[]) => {
  if (typeof id === "string") id = zod.parse(stringIDZ, id, { label: "resource ID" });
  if (Array.isArray(id)) return id.map((id) => idToString(id));
  return `${id.type}:${id.key}`;
}) as IDToString;

/** @returns whether the two IDs name the same resource. */
export const idsEqual = (a: ID, b: ID) => a.type === b.type && a.key === b.key;

/**
 * Parses a crude ID (an object or a "type:key" string) into an ID.
 * @throws {zod.ParseError} if the value is not a valid resource ID.
 */
export const parseID = (id: unknown): ID =>
  zod.parse(idZ, id, { label: "resource ID" });
