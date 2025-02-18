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

import * as policy from "@/access/policy/ontology";
import * as channel from "@/channel/payload";
import * as framer from "@/framer/frame";
import * as device from "@/hardware/device/payload";
import * as rack from "@/hardware/rack/payload";
import * as task from "@/hardware/task/payload";
import * as label from "@/label/payload";
import * as group from "@/ontology/group/payload";
import * as ranger from "@/ranger/payload";
import * as user from "@/user/payload";
import * as linePlot from "@/workspace/lineplot/payload";
import * as log from "@/workspace/log/payload";
import * as workspace from "@/workspace/payload";
import * as schematic from "@/workspace/schematic/payload";
import * as table from "@/workspace/table/payload";

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
  label.ONTOLOGY_TYPE,
  log.ONTOLOGY_TYPE,
  policy.ALLOW_ALL_ONTOLOGY_TYPE,
  BUILTIN_TYPE,
  CLUSTER_TYPE,
  channel.ONTOLOGY_TYPE,
  NODE_TYPE,
  group.ONTOLOGY_TYPE,
  ranger.ONTOLOGY_TYPE,
  framer.ONTOLOGY_TYPE,
  ranger.ALIAS_ONTOLOGY_TYPE,
  user.ONTOLOGY_TYPE,
  workspace.ONTOLOGY_TYPE,
  schematic.ONTOLOGY_TYPE,
  linePlot.ONTOLOGY_TYPE,
  rack.ONTOLOGY_TYPE,
  device.ONTOLOGY_TYPE,
  task.ONTOLOGY_TYPE,
  policy.ONTOLOGY_TYPE,
  table.ONTOLOGY_TYPE,
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

export const ROOT_ID = new ID({ type: BUILTIN_TYPE, key: "root" });

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
