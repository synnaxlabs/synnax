// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

const resourceTypeZ = z.union([
  z.literal("builtin"),
  z.literal("cluster"),
  z.literal("channel"),
  z.literal("node"),
  z.literal("group"),
  z.literal("range"),
  z.literal("user"),
]);

export type ResourceType = z.infer<typeof resourceTypeZ>;

export const idZ = z.object({
  type: resourceTypeZ,
  key: z.string(),
});

export const crudeIDZ = z.union([z.string(), idZ]);

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
    schema: schemaZ.optional(),
    data: z.record(z.unknown()).optional(),
  })
  .transform((resource) => {
    return {
      key: resource.id.toString(),
      ...resource,
    };
  });

export type Resource = z.infer<typeof resourceSchemaZ>;
