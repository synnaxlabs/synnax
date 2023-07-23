// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

const ontologyResourceTypeSchema = z.union([
  z.literal("builtin"),
  z.literal("cluster"),
  z.literal("channel"),
  z.literal("node"),
  z.literal("group"),
  z.literal("range"),
]);

export type OntologyResourceType = z.infer<typeof ontologyResourceTypeSchema>;

export const ontologyID = z.object({
  type: ontologyResourceTypeSchema,
  key: z.string(),
});

export const crudeOntologyID = z.union([z.string(), ontologyID]);

export class OntologyID {
  type: OntologyResourceType;
  key: string;

  constructor(args: z.input<typeof crudeOntologyID> | OntologyID) {
    if (args instanceof OntologyID) {
      this.type = args.type;
      this.key = args.key;
    } else if (typeof args === "string") {
      const [type, key] = args.split(":");
      this.type = type as OntologyResourceType;
      this.key = key;
    } else {
      this.type = args.type;
      this.key = args.key;
    }
  }

  toString(): string {
    return `${this.type}:${this.key}`;
  }

  get payload(): z.infer<typeof ontologyID> {
    return {
      type: this.type,
      key: this.key,
    };
  }

  static readonly z = z
    .union([crudeOntologyID, z.instanceof(OntologyID)])
    .transform((v) => new OntologyID(v));
}

export const OntologyRoot = new OntologyID({ type: "builtin", key: "root" });

export const ontologySchemaFieldSchema = z.object({
  type: z.number(),
});

export type OntologySchemaField = z.infer<typeof ontologySchemaFieldSchema>;

export const ontologySchemaSchema = z.object({
  type: ontologyResourceTypeSchema,
  fields: z.record(ontologySchemaFieldSchema),
});

export type OntologySchema = z.infer<typeof ontologySchemaSchema>;

export const ontologyResourceSchema = z
  .object({
    id: OntologyID.z,
    name: z.string(),
    schema: ontologySchemaSchema.optional(),
    data: z.record(z.unknown()).optional(),
  })
  .transform((resource) => {
    return {
      key: resource.id.toString(),
      ...resource,
    };
  });

export type OntologyResource = z.infer<typeof ontologyResourceSchema>;
