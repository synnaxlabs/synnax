// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, type change, primitive, record, zod } from "@synnaxlabs/x";
import { z } from "zod";

import { RESOURCE_TYPES, type ResourceType, resourceTypeZ } from "@/ontology/types.gen";

export { RESOURCE_TYPES, type ResourceType, resourceTypeZ };

/** A resource created, updated, or deleted. */
export type ResourceChange = change.Change<ID, Resource>;
export interface ResourceSet extends change.Set<ID, Resource> {}
export interface ResourceDelete extends change.Delete<ID, Resource> {}
/** A relationship created or deleted. Relationships carry no value of their own. */
export type RelationshipChange = change.Change<Relationship, undefined>;
export interface RelationshipSet extends change.Set<Relationship, undefined> {}
export interface RelationshipDelete extends change.Delete<Relationship, undefined> {}

export interface RelationshipDelete extends change.Delete<Relationship, undefined> {}

/** Zod schema parsing a `"type:key"` string into an {@link ID}. */
export const stringIDZ = z.string().transform((v, ctx) => {
  const [type, key] = v.split(":");
  const res = resourceTypeZ.safeParse(type);
  if (!res.success) {
    ctx.addIssue({ code: "custom", message: `invalid resource type: ${type}` });
    return z.NEVER;
  }
  return { type: res.data, key: key ?? "" };
});

/** Zod schema for {@link ID}, accepting either the object or the `"type:key"` string. */
export const idZ = z.object({ type: resourceTypeZ, key: z.string() }).or(stringIDZ);

/** Names one resource in the ontology: what kind it is, and which one. */
export type ID = z.infer<typeof idZ>;

/** The root of the ontology. Every other resource descends from it. */
export const ROOT_ID: ID = { type: "builtin", key: "root" };

/** Builds {@link ID}s of one resource type, singly or in bulk. */
export interface CreateID<K extends record.Key> {
  (key: K): ID;
  (keys: K[]): ID[];
  (keys: K | K[]): ID | ID[];
}

/**
 * @returns a {@link CreateID} that stamps the given resource type onto keys. Each
 * resource package exports one as `ontologyID`.
 *
 * @example const ontologyID = createIDFactory<Key>("channel");
 */
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

/** Renders an {@link ID} as `"type:key"`, the form used as a map key. */
export const idToString = ((id: ID | string | (ID | string)[]) => {
  if (typeof id === "string") id = zod.parse(stringIDZ, id, { label: "ontology ID" });
  if (Array.isArray(id)) return id.map((id) => idToString(id));
  return `${id.type}:${id.key}`;
}) as IDToString;

/** @returns whether the two IDs name the same resource. */
export const idsEqual = (a: ID, b: ID) => a.type === b.type && a.key === b.key;

/**
 * Parses a crude ID (an object or a "type:key" string) into an ID.
 * @throws {zod.ParseError} if the value is not a valid ontology ID.
 */
export const parseID = (id: unknown): ID =>
  zod.parse(idZ, id, { label: "ontology ID" });

/**
 * Resolves IDs, `"type:key"` strings, or whole resources into an {@link ID} array.
 * @throws {zod.ParseError} if a value is not a valid ontology ID.
 */
export const parseIDs = (
  ids: ID | string | Resource | (ID | string | Resource)[],
): ID[] => {
  const arr = array.toArray(ids);
  if (arr.length === 0) return [];
  if (typeof arr[0] === "object" && "id" in arr[0])
    return (arr as Resource[]).map(({ id }) => id);
  return arr.map(parseID);
};

/** Zod schema for {@link Resource}. */
export const resourceZ = z
  .object({
    id: idZ,
    name: z.string(),
    data: record.unknownZ().optional().nullable(),
  })
  .transform((resource) => ({ key: idToString(resource.id), ...resource }));
/** A node in the ontology: its ID, its display name, and the payload it stands for. */
export interface Resource<T extends record.Unknown = record.Unknown> extends Omit<
  z.infer<typeof resourceZ>,
  "data"
> {
  data?: T | null;
}

/** Which end of a relationship a traversal starts from. */
export type RelationshipDirection = "to" | "from";

/** @returns the other end of a relationship. */
export const oppositeRelationshipDirection = (
  direction: RelationshipDirection,
): RelationshipDirection => (direction === "to" ? "from" : "to");

/** Zod schema for {@link Relationship}, accepting the `"from->type->to"` string too. */
export const relationshipZ = z.object({ from: idZ, type: z.string(), to: idZ }).or(
  z.string().transform((v) => {
    const [from, type, to] = v.split("->");
    return { from: idZ.parse(from), type, to: idZ.parse(to) };
  }),
);
/** A directed edge between two resources, such as a group and its child. */
export type Relationship = z.infer<typeof relationshipZ>;

/** Renders a {@link Relationship} as `"from->type->to"`. */
export const relationshipToString = (relationship: Relationship) =>
  `${idToString(relationship.from)}->${relationship.type}->${idToString(relationship.to)}`;

/** Relationship type joining a parent resource to its children. */
export const PARENT_OF_RELATIONSHIP_TYPE = "parent";

/** A relationship pattern. An absent field, or field part, matches anything. */
export interface MatchRelationshipParams {
  from?: Partial<ID>;
  type: string;
  to?: Partial<ID>;
}

/** @returns whether the relationship fits the pattern. */
export const matchRelationship = (
  relationship: Relationship,
  match: MatchRelationshipParams,
) => {
  if (match.type != null && match.type !== relationship.type) return false;
  if (match.from?.type != null && match.from.type !== relationship.from.type)
    return false;
  if (match.to?.type != null && match.to.type !== relationship.to.type) return false;
  if (match.from?.key != null && match.from.key !== relationship.from.key) return false;
  if (match.to?.key != null && match.to.key !== relationship.to.key) return false;
  return true;
};
