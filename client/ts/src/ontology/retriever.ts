// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { UnaryClient } from "@synnaxlabs/freighter";
import { AsyncTermSearcher, toArray } from "@synnaxlabs/x";
import { z } from "zod";

import { QueryError } from "@/errors";
import { ID, Resource, idZ, resourceSchemaZ } from "@/ontology/payload";

const requestSchema = z.object({
  ids: idZ.array().optional(),
  children: z.boolean().optional(),
  parents: z.boolean().optional(),
  includeSchema: z.boolean().optional(),
  includeFieldData: z.boolean().optional(),
  term: z.string().optional(),
});

type Request = z.infer<typeof requestSchema>;

const responseSchema = z.object({
  resources: resourceSchemaZ.array(),
});

export class Retriever {
  private static readonly ENDPOINT = "/ontology/retrieve";
  private readonly client: UnaryClient;

  constructor(unary: UnaryClient) {
    this.client = unary;
  }

  async search(term: string): Promise<Resource[]> {
    const resources = await this.execute({ term });
    return resources;
  }

  async retrieve(
    ids: ID | ID[] | string | string[],
    includeSchema: boolean = true,
    includeFieldData: boolean = true
  ): Promise<Resource | Resource[]> {
    const resources = await this.execute({
      ids: toArray(ids).map((id) => new ID(id).payload),
      includeFieldData,
      includeSchema,
    });
    if (Array.isArray(ids)) return resources;
    if (resources.length === 0)
      throw new QueryError(`No resource found with ID ${ids.toString()}`);
    return resources[0];
  }

  async retrieveChildren(
    ids: ID | ID[],
    includeSchema: boolean = true,
    includeFieldData: boolean = true
  ): Promise<Resource[]> {
    return await this.execute({
      ids: toArray(ids).map((id) => new ID(id).payload),
      children: true,
      includeSchema,
      includeFieldData,
    });
  }

  async retrieveParents(
    ids: ID | ID[],
    includeSchema: boolean = true,
    includeFieldData: boolean = true
  ): Promise<Resource[]> {
    return await this.execute({
      ids: toArray(ids).map((id) => new ID(id).payload),
      parents: true,
      includeSchema,
      includeFieldData,
    });
  }

  private async execute(request: Request): Promise<Resource[]> {
    const [res, err] = await this.client.send(
      Retriever.ENDPOINT,
      request,
      responseSchema
    );
    if (err != null) throw err;
    return res.resources;
  }
}
