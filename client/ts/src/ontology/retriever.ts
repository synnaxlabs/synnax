// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { UnaryClient } from "@synnaxlabs/freighter";
import { toArray } from "@synnaxlabs/x";
import { z } from "zod";

import { QueryError } from "@/errors";
import {
  OntologyID,
  OntologyResource,
  ontologyResourceSchema,
} from "@/ontology/payload";

const requestSchema = z.object({
  ids: z.string().array(),
  children: z.boolean().optional(),
  parents: z.boolean().optional(),
  includeSchema: z.boolean().optional(),
  includeFieldData: z.boolean().optional(),
});

type Request = z.infer<typeof requestSchema>;

const responseSchema = z.object({
  resources: ontologyResourceSchema.array(),
});

export class OntologyRetriever {
  private static readonly ENDPOINT = "/ontology/retrieve";
  private readonly client: UnaryClient;

  constructor(unary: UnaryClient) {
    this.client = unary;
  }

  async retrieve(
    id: OntologyID,
    includeSchema?: boolean,
    includeFieldData?: boolean
  ): Promise<OntologyResource>;

  async retrieve(
    ids: OntologyID[],
    includeSchema?: boolean,
    includeFieldData?: boolean
  ): Promise<OntologyResource[]>;

  async retrieve(
    ids: OntologyID | OntologyID[],
    includeSchema?: boolean,
    includeFieldData?: boolean
  ): Promise<OntologyResource | OntologyResource[]> {
    const resources = await this.execute({
      ids: toArray(ids).map((id) => id.toString()),
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
      ids: toArray(ids).map((id) => id.toString()),
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
      ids: toArray(ids).map((id) => id.toString()),
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
