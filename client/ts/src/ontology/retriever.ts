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
import {
  OntologyID,
  OntologyResource,
  ontologyID,
  ontologyResourceSchema,
} from "@/ontology/payload";

const requestSchema = z.object({
  ids: ontologyID.array().optional(),
  children: z.boolean().optional(),
  parents: z.boolean().optional(),
  includeSchema: z.boolean().optional(),
  includeFieldData: z.boolean().optional(),
  term: z.string().optional(),
});

type Request = z.infer<typeof requestSchema>;

const responseSchema = z.object({
  resources: ontologyResourceSchema.array(),
});

export class OntologyRetriever
  implements AsyncTermSearcher<string, string, OntologyResource>
{
  private static readonly ENDPOINT = "/ontology/retrieve";
  private readonly client: UnaryClient;

  constructor(unary: UnaryClient) {
    this.client = unary;
  }

  async search(term: string): Promise<OntologyResource[]> {
    const resources = await this.execute({ term });
    return resources;
  }

  async retrieve(
    id: OntologyID | string,
    includeSchema?: boolean,
    includeFieldData?: boolean
  ): Promise<OntologyResource>;

  async retrieve(
    ids: OntologyID[] | string[],
    includeSchema?: boolean,
    includeFieldData?: boolean
  ): Promise<OntologyResource[]>;

  async retrieve(
    ids: OntologyID | OntologyID[] | string | string[],
    includeSchema?: boolean,
    includeFieldData?: boolean
  ): Promise<OntologyResource | OntologyResource[]> {
    const resources = await this.execute({
      ids: toArray(ids).map((id) => new OntologyID(id).payload),
      includeFieldData,
      includeSchema,
    });
    if (Array.isArray(ids)) return resources;
    if (resources.length === 0)
      throw new QueryError(`No resource found with ID ${ids.toString()}`);
    return resources[0];
  }

  async retrieveChildren(
    ids: OntologyID | OntologyID[],
    includeSchema?: boolean,
    includeFieldData?: boolean
  ): Promise<OntologyResource[]> {
    return await this.execute({
      ids: toArray(ids).map((id) => new OntologyID(id).payload),
      children: true,
      includeSchema,
      includeFieldData,
    });
  }

  async retrieveParents(
    ids: OntologyID | OntologyID[],
    includeSchema?: boolean,
    includeFieldData?: boolean
  ): Promise<OntologyResource[]> {
    return await this.execute({
      ids: toArray(ids).map((id) => new OntologyID(id).payload),
      parents: true,
      includeSchema,
      includeFieldData,
    });
  }

  private async execute(request: Request): Promise<OntologyResource[]> {
    const [res, err] = await this.client.send(
      OntologyRetriever.ENDPOINT,
      request,
      responseSchema
    );
    if (err != null) throw err;
    return res.resources;
  }
}
