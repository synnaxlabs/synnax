// Copyright 2022 Synnax Labs, Inc.
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
]);

export type OntologyResourceType = z.infer<typeof ontologyResourceTypeSchema>;

export const ontologyIdSchema = z.object({
  type: ontologyResourceTypeSchema,
  key: z.string(),
});

export class OntologyID {
  type: OntologyResourceType;
  key: string;

  constructor(type: OntologyResourceType, key: string) {
    this.type = type;
    this.key = key;
  }

  toString(): string {
    return `${this.type}:${this.key}`;
  }

  static parseString(str: string): OntologyID {
    const [type, key] = str.split(":");
    return new OntologyID(type as OntologyResourceType, key);
  }
}

export const OntologyRoot = new OntologyID("builtin", "root");

export const ontologySchemaFieldSchema = z.object({
  type: z.number(),
});

export type OntologySchemaField = z.infer<typeof ontologySchemaFieldSchema>;

export const ontologySchemaSchema = z.object({
  type: ontologyResourceTypeSchema,
  fields: z.record(ontologySchemaFieldSchema),
});

export type OntologySchema = z.infer<typeof ontologySchemaSchema>;

export const ontologyResourceSchema = z.object({
  id: ontologyIdSchema.transform((id) => new OntologyID(id.type, id.key)),
  entity: z.object({
    schema: ontologySchemaSchema,
    name: z.string(),
    data: z.record(z.unknown()),
  }),
});

export type OntologyResource = z.infer<typeof ontologyResourceSchema>;
