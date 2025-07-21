// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, type change, record } from "@synnaxlabs/x";
import { z } from "zod";

import {
  ALLOW_ALL_ONTOLOGY_TYPE as ALLOW_ALL_TYPE,
  ONTOLOGY_TYPE as POLICY_TYPE,
} from "@/access/policy/ontology";
import { ONTOLOGY_TYPE as ANNOTATION_TYPE } from "@/annotation/payload";
import { ONTOLOGY_TYPE as CHANNEL_TYPE } from "@/channel/payload";
import { ONTOLOGY_TYPE as EFFECT_TYPE } from "@/effect/payload";
import { ONTOLOGY_TYPE as FRAMER_TYPE } from "@/framer/frame";
import { ONTOLOGY_TYPE as DEVICE_TYPE } from "@/hardware/device/payload";
import { ONTOLOGY_TYPE as RACK_TYPE } from "@/hardware/rack/payload";
import { ONTOLOGY_TYPE as TASK_TYPE } from "@/hardware/task/payload";
import { ONTOLOGY_TYPE as LABEL_TYPE } from "@/label/payload";
import { ONTOLOGY_TYPE as GROUP_TYPE } from "@/ontology/group/payload";
import {
  ALIAS_ONTOLOGY_TYPE as RANGE_ALIAS_TYPE,
  ONTOLOGY_TYPE as RANGE_TYPE,
} from "@/ranger/payload";
import { ONTOLOGY_TYPE as SLATE_TYPE } from "@/slate/payload";
import { ONTOLOGY_TYPE as USER_TYPE } from "@/user/payload";
import { ONTOLOGY_TYPE as LINE_PLOT_TYPE } from "@/workspace/lineplot/payload";
import { ONTOLOGY_TYPE as LOG_TYPE } from "@/workspace/log/payload";
import { ONTOLOGY_TYPE as WORKSPACE_TYPE } from "@/workspace/payload";
import { ONTOLOGY_TYPE as SCHEMATIC_TYPE } from "@/workspace/schematic/payload";
import { ONTOLOGY_TYPE as TABLE_TYPE } from "@/workspace/table/payload";
export type ResourceChange = change.Change<ID, Resource>;
export interface ResourceSet extends change.Set<ID, Resource> {}
export interface ResourceDelete extends change.Delete<ID, Resource> {}
export type RelationshipChange = change.Change<Relationship, undefined>;
export interface RelationshipSet extends change.Set<Relationship, undefined> {}
export interface RelationshipDelete extends change.Delete<Relationship, undefined> {}

export const BUILTIN_TYPE = "builtin";
export const CLUSTER_TYPE = "cluster";
export const NODE_TYPE = "node";

export const resourceTypeZ = z.enum([
  LABEL_TYPE,
  LOG_TYPE,
  ALLOW_ALL_TYPE,
  BUILTIN_TYPE,
  CLUSTER_TYPE,
  CHANNEL_TYPE,
  NODE_TYPE,
  GROUP_TYPE,
  RANGE_TYPE,
  FRAMER_TYPE,
  RANGE_ALIAS_TYPE,
  USER_TYPE,
  WORKSPACE_TYPE,
  SCHEMATIC_TYPE,
  LINE_PLOT_TYPE,
  RACK_TYPE,
  DEVICE_TYPE,
  TASK_TYPE,
  POLICY_TYPE,
  TABLE_TYPE,
  EFFECT_TYPE,
  SLATE_TYPE,
  ANNOTATION_TYPE,
]);
export type ResourceType = z.infer<typeof resourceTypeZ>;

const stringIDZ = z.string().transform((v) => {
  const [type, key] = v.split(":");
  return { type: resourceTypeZ.parse(type), key: key ?? "" };
});

export const idZ = z.object({ type: resourceTypeZ, key: z.string() }).or(stringIDZ);

export type ID = z.infer<typeof idZ>;

export const ROOT_ID: ID = { type: BUILTIN_TYPE, key: "root" };

export const idToString = (id: ID) => `${id.type}:${id.key}`;

export const idsEqual = (a: ID, b: ID) => a.type === b.type && a.key === b.key;

export const parseIDs = (
  ids: ID | ID[] | string | string[] | Resource | Resource[],
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
export interface Resource<T extends record.Unknown = record.Unknown>
  extends Omit<z.infer<typeof resourceZ>, "data"> {
  data?: T | null;
}

export type RelationshipDirection = "to" | "from";

export const oppositeRelationshipDirection = (
  direction: RelationshipDirection,
): RelationshipDirection => (direction === "to" ? "from" : "to");

export const relationShipZ = z.object({ from: idZ, type: z.string(), to: idZ }).or(
  z.string().transform((v) => {
    const [from, type, to] = v.split("->");
    return { from: idZ.parse(from), type, to: idZ.parse(to) };
  }),
);
export type Relationship = z.infer<typeof relationShipZ>;

export const PARENT_OF_RELATIONSHIP_TYPE = "parent";
