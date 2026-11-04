// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, type change, record } from "@synnaxlabs/x";
import { z } from "zod";

import { type ID, idToString, idZ, parseID } from "@/resource/payload";

/** A resource created, updated, or deleted. */
export type ResourceChange = change.Change<ID, Resource>;
export interface ResourceSet extends change.Set<ID, Resource> {}
export interface ResourceDelete extends change.Delete<ID, Resource> {}
/** A relationship created or deleted. Relationships carry no value of their own. */
export type RelationshipChange = change.Change<Relationship, undefined>;
export interface RelationshipSet extends change.Set<Relationship, undefined> {}
export interface RelationshipDelete extends change.Delete<Relationship, undefined> {}

export interface RelationshipDelete extends change.Delete<Relationship, undefined> {}

/** The root of the ontology. Every other resource descends from it. */
export const ROOT_ID: ID = { type: "builtin", key: "root" };

/**
 * Resolves IDs, `"type:key"` strings, or whole resources into an {@link ID} array.
 * @throws {zod.ParseError} if a value is not a valid resource ID.
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
