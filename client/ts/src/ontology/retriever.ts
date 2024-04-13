// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { toArray } from "@synnaxlabs/x";
import { z } from "zod";

import { ID, type Resource, idZ, resourceSchemaZ } from "@/ontology/payload";

const reqZ = z.object({
  ids: idZ.array().optional(),
  children: z.boolean().optional(),
  parents: z.boolean().optional(),
  includeSchema: z.boolean().optional(),
  includeFieldData: z.boolean().optional(),
  term: z.string().optional(),
});

type Request = z.infer<typeof reqZ>;

const resZ = z.object({
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
    includeFieldData: boolean = true,
  ): Promise<Resource[]> {
    return await this.execute({
      ids: toArray(ids).map((id) => new ID(id).payload),
      includeFieldData,
      includeSchema,
    });
  }

  async retrieveChildren(
    ids: ID | ID[],
    includeSchema: boolean = true,
    includeFieldData: boolean = true,
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
    includeFieldData: boolean = true,
  ): Promise<Resource[]> {
    return await this.execute({
      ids: toArray(ids).map((id) => new ID(id).payload),
      parents: true,
      includeSchema,
      includeFieldData,
    });
  }

  private async execute(request: Request): Promise<Resource[]> {
    return (await sendRequired(
      this.client,
      Retriever.ENDPOINT,
      request,
      reqZ,
      resZ,
    )).resources;
  }
}
