// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, type change, primitive, record } from "@synnaxlabs/x";
import { z } from "zod";

export type ResourceChange = change.Change<ID, Resource>;
export interface ResourceSet extends change.Set<ID, Resource> {}
export interface ResourceDelete extends change.Delete<ID, Resource> {}
export type RelationshipChange = change.Change<Relationship, undefined>;
export interface RelationshipSet extends change.Set<Relationship, undefined> {}
export interface RelationshipDelete extends change.Delete<Relationship, undefined> {}

export interface RelationshipDelete extends change.Delete<Relationship, undefined> {}

export const resourceTypeZ = z.enum([
  "label",
  "log",
  "builtin",
  "cluster",
  "channel",
  "node",
  "group",
  "range",
  "framer",
  "range-alias",
  "user",
  "workspace",
  "schematic",
  "lineplot",
  "rack",
  "device",
  "task",
  "policy",
  "role",
  "table",
  "arc",
  "schematic_symbol",
  "status",
  "view",
]);
export type ResourceType = z.infer<typeof resourceTypeZ>;

const stringIDZ = z.string().transform((v) => {
  const [type, key] = v.split(":");
  return { type: resourceTypeZ.parse(type), key: key ?? "" };
});

export const idZ = z.object({ type: resourceTypeZ, key: z.string() }).or(stringIDZ);

export type ID = z.infer<typeof idZ>;

export const ROOT_ID: ID = { type: "builtin", key: "root" };

export interface CreateID<K extends record.Key> {
  (key: K): ID;
  (keys: K[]): ID[];
  (keys: K | K[]): ID | ID[];
}

export const createIDFactory = <K extends record.Key>(
  type: ResourceType,
): CreateID<K> => {
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

export const idToString = ((id: ID | string | (ID | string)[]) => {
  if (typeof id === "string") id = stringIDZ.parse(id);
  if (Array.isArray(id)) return id.map((id) => idToString(id));
  return `${id.type}:${id.key}`;
}) as IDToString;

export const idsEqual = (a: ID, b: ID) => a.type === b.type && a.key === b.key;

export const parseIDs = (
  ids: ID | string | Resource | (ID | string | Resource)[],
): ID[] => {
  const arr = array.toArray(ids);
  if (arr.length === 0) return [];
  if (typeof arr[0] === "object" && "id" in arr[0])
    return (arr as Resource[]).map(({ id }) => id);
  return arr.map((id) => idZ.parse(id));
};

export const resourceZ = z
  .object({
    id: idZ,
    name: z.string(),
    data: record.unknownZ.optional().nullable(),
  })
  .transform((resource) => ({ key: idToString(resource.id), ...resource }));
export interface Resource<T extends record.Unknown = record.Unknown> extends Omit<
  z.infer<typeof resourceZ>,
  "data"
> {
  data?: T | null;
}

export type RelationshipDirection = "to" | "from";

export const oppositeRelationshipDirection = (
  direction: RelationshipDirection,
): RelationshipDirection => (direction === "to" ? "from" : "to");

export const relationshipZ = z.object({ from: idZ, type: z.string(), to: idZ }).or(
  z.string().transform((v) => {
    const [from, type, to] = v.split("->");
    return { from: idZ.parse(from), type, to: idZ.parse(to) };
  }),
);
export type Relationship = z.infer<typeof relationshipZ>;

export const relationshipToString = (relationship: Relationship) =>
  `${idToString(relationship.from)}->${relationship.type}->${idToString(relationship.to)}`;

export const PARENT_OF_RELATIONSHIP_TYPE = "parent";

export interface MatchRelationshipArgs {
  from?: Partial<ID>;
  type: string;
  to?: Partial<ID>;
}

export const matchRelationship = (
  relationship: Relationship,
  match: MatchRelationshipArgs,
) => {
  if (match.type != null && match.type !== relationship.type) return false;
  if (match.from?.type != null && match.from.type !== relationship.from.type)
    return false;
  if (match.to?.type != null && match.to.type !== relationship.to.type) return false;
  if (match.from?.key != null && match.from.key !== relationship.from.key) return false;
  if (match.to?.key != null && match.to.key !== relationship.to.key) return false;
  return true;
};
