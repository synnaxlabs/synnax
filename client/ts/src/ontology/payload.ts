// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type change, type UnknownRecord } from "@synnaxlabs/x";
import { z } from "zod";

import {
  ALLOW_ALL_ONTOLOGY_TYPE as ALLOW_ALL_TYPE,
  ONTOLOGY_TYPE as POLICY_TYPE,
} from "@/access/policy/ontology";
import { ONTOLOGY_TYPE as CHANNEL_TYPE } from "@/channel/payload";
import { ONTOLOGY_TYPE as FRAME_TYPE } from "@/framer/frame";
import { ONTOLOGY_TYPE as DEVICE_TYPE } from "@/hardware/device/payload";
import { ONTOLOGY_TYPE as RACK_TYPE } from "@/hardware/rack/payload";
import { ONTOLOGY_TYPE as TASK_TYPE } from "@/hardware/task/payload";
import { ONTOLOGY_TYPE as LABEL_TYPE } from "@/label/payload";
import { ONTOLOGY_TYPE as GROUP_TYPE } from "@/ontology/group/payload";
import {
  ALIAS_ONTOLOGY_TYPE as RANGE_ALIAS_TYPE,
  ONTOLOGY_TYPE as RANGE_TYPE,
} from "@/ranger/payload";
import { ONTOLOGY_TYPE as USER_TYPE } from "@/user/payload";
import { ONTOLOGY_TYPE as LINEPLOT_TYPE } from "@/workspace/lineplot/payload";
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
  FRAME_TYPE,
  RANGE_ALIAS_TYPE,
  USER_TYPE,
  WORKSPACE_TYPE,
  SCHEMATIC_TYPE,
  LINEPLOT_TYPE,
  RACK_TYPE,
  DEVICE_TYPE,
  TASK_TYPE,
  POLICY_TYPE,
  TABLE_TYPE,
]);
export type ResourceType = z.infer<typeof resourceTypeZ>;

export const idZ = z.object({ type: resourceTypeZ, key: z.string() });
export interface IDPayload extends z.infer<typeof idZ> {}

export const stringIDZ = z.string().transform((v) => {
  const [type, key] = v.split(":");
  return { type: resourceTypeZ.parse(type), key: key ?? "" };
});

export const crudeIDZ = z.union([stringIDZ, idZ]);
export type CrudeID = z.input<typeof crudeIDZ>;

export class ID {
  type: ResourceType;
  key: string;

  constructor(args: z.input<typeof crudeIDZ> | ID) {
    if (args instanceof ID) {
      this.type = args.type;
      this.key = args.key;
      return;
    }
    if (typeof args === "string") {
      const [type, key] = args.split(":");
      this.type = type as ResourceType;
      this.key = key ?? "";
      return;
    }
    this.type = args.type;
    this.key = args.key;
  }

  toString(): string {
    return `${this.type}:${this.key}`;
  }

  isType(): boolean {
    return this.key === "";
  }

  matchesType(type: ResourceType): boolean {
    return this.type === type && this.isType();
  }

  get payload(): IDPayload {
    return { type: this.type, key: this.key };
  }

  static readonly z = z.union([z.instanceof(ID), crudeIDZ.transform((v) => new ID(v))]);
}

export const RootID = new ID({ type: BUILTIN_TYPE, key: "root" });

export const schemaFieldZ = z.object({ type: z.number() });
export interface SchemaField extends z.infer<typeof schemaFieldZ> {}

export const schemaZ = z.object({
  type: resourceTypeZ,
  fields: z.record(schemaFieldZ),
});
export interface Schema extends z.infer<typeof schemaZ> {}

export const resourceZ = z
  .object({
    id: ID.z,
    name: z.string(),
    schema: schemaZ.optional().nullable(),
    data: z.record(z.unknown()).optional().nullable(),
  })
  .transform((resource) => ({ key: resource.id.toString(), ...resource }));
export interface Resource<T extends UnknownRecord = UnknownRecord>
  extends Omit<z.output<typeof resourceZ>, "data"> {
  data?: T | null;
}

export const TO_RELATIONSHIP_DIRECTION = "to";
export const FROM_RELATIONSHIP_DIRECTION = "from";
export type RelationshipDirection =
  | typeof TO_RELATIONSHIP_DIRECTION
  | typeof FROM_RELATIONSHIP_DIRECTION;

export const relationshipSchemaZ = z.object({ from: ID.z, type: z.string(), to: ID.z });
export interface Relationship extends z.infer<typeof relationshipSchemaZ> {}

export const parseRelationship = (str: string): Relationship => {
  const [from, type, to] = str.split("->");
  return { from: new ID(from), type, to: new ID(to) };
};
