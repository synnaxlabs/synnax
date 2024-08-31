// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { change, UnknownRecord } from "@synnaxlabs/x";
import { z } from "zod";

export type ResourceChange = change.Change<ID, Resource>;
export type ResourceSet = change.Set<ID, Resource>;
export type ResourceDelete = change.Delete<ID, Resource>;
export type RelationshipChange = change.Change<Relationship, undefined>;
export type RelationshipSet = change.Set<Relationship, undefined>;
export type RelationshipDelete = change.Delete<Relationship, undefined>;

export const resourceTypeZ = z.union([
  z.literal("label"),
  z.literal("builtin"),
  z.literal("cluster"),
  z.literal("channel"),
  z.literal("node"),
  z.literal("group"),
  z.literal("range"),
  z.literal("range-alias"),
  z.literal("user"),
  z.literal("workspace"),
  z.literal("schematic"),
  z.literal("lineplot"),
  z.literal("rack"),
  z.literal("device"),
  z.literal("task"),
  z.literal("policy"),
]);

export type ResourceType = z.infer<typeof resourceTypeZ>;

export const BuiltinOntologyType = "builtin" as ResourceType;
export const ClusterOntologyType = "cluster" as ResourceType;
export const NodeOntologyType = "node" as ResourceType;

export const idZ = z.object({ type: resourceTypeZ, key: z.string() });

export type IDPayload = z.infer<typeof idZ>;

export const stringIDZ = z.string().transform((v) => {
  const [type, key] = v.split(":");
  return { type: type as ResourceType, key };
});

export const crudeIDZ = z.union([stringIDZ, idZ]);

export type CrudeID = { type: ResourceType; key: string } | string;

export class ID {
  type: ResourceType;
  key: string;

  constructor(args: z.input<typeof crudeIDZ> | ID) {
    if (args instanceof ID) {
      this.type = args.type;
      this.key = args.key;
    } else if (typeof args === "string") {
      const [type, key] = args.split(":");
      this.type = type as ResourceType;
      this.key = key;
    } else {
      this.type = args.type;
      this.key = args.key;
    }
  }

  toString(): string {
    return `${this.type}:${this.key}`;
  }

  get payload(): z.infer<typeof idZ> {
    return {
      type: this.type,
      key: this.key,
    };
  }

  static readonly z = z.union([crudeIDZ, z.instanceof(ID)]).transform((v) => new ID(v));
}

export const Root = new ID({ type: "builtin", key: "root" });

export const schemaFieldZ = z.object({
  type: z.number(),
});

export type SchemaField = z.infer<typeof schemaFieldZ>;

export const schemaZ = z.object({
  type: resourceTypeZ,
  fields: z.record(schemaFieldZ),
});

export type Schema = z.infer<typeof schemaZ>;

export const resourceSchemaZ = z
  .object({
    id: ID.z,
    name: z.string(),
    schema: schemaZ.optional().nullable(),
    data: z.record(z.unknown()).optional().nullable(),
  })
  .transform((resource) => {
    return {
      key: resource.id.toString(),
      ...resource,
    };
  });

export type Resource<T extends UnknownRecord = UnknownRecord> = Omit<
  z.output<typeof resourceSchemaZ>,
  "data"
> & { data?: T | null };

export type RelationshipDirection = "from" | "to";

export const relationshipSchemaZ = z.object({
  from: ID.z,
  type: z.string(),
  to: ID.z,
});

export type Relationship = z.infer<typeof relationshipSchemaZ>;

export const parseRelationship = (str: string): Relationship => {
  const [from, type, to] = str.split("->");
  return { from: new ID(from), type, to: new ID(to) };
};
