// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array } from "@synnaxlabs/x";
import { z } from "zod";

import { type ID, idZ, type Resource, resourceZ } from "@/resource/payload";
import { typeZ } from "@/resource/types.gen";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

const searchReqZ = z.object({
  /** The text that a resource must match. An empty term matches every resource. */
  term: z.string(),
  /** Limits the search to resources of these types. Empty searches every type. */
  types: typeZ.array().optional(),
  /** The maximum number of resources to return. Zero returns at most 100. */
  limit: z.int().optional(),
  /** The number of closest matches to skip. */
  offset: z.int().optional(),
});
/** Selects the resources that {@link Client.search} returns. */
export interface SearchParams extends z.input<typeof searchReqZ> {}

const retrieveReqZ = z.object({ ids: idZ.array() });
const resZ = z.object({ resources: resourceZ.array().default(() => []) });

/**
 * Searches and names records of every type on a Core. Reach it through
 * `client.resources`.
 */
export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  /**
   * @returns the resources whose text matches the term, closest match first.
   * @throws {ValidationError} if a type is unknown, or the limit or offset is negative.
   */
  async search(params: SearchParams): Promise<Resource[]> {
    const { resources } = await this.client.send(
      "/resource/search",
      params,
      searchReqZ,
      resZ,
    );
    return resources;
  }

  /**
   * Retrieves the ID and name of each resource. The list form returns the resources in
   * the order of the IDs and leaves out each ID that names no resource.
   * @throws {NotFoundError} if the single ID names no resource.
   * @throws {ValidationError} if an ID has an unknown type.
   */
  async retrieve(id: ID): Promise<Resource>;
  async retrieve(ids: ID[]): Promise<Resource[]>;
  async retrieve(ids: ID | ID[]): Promise<Resource | Resource[]> {
    const isSingle = !Array.isArray(ids);
    const { resources } = await this.client.send(
      "/resource/retrieve",
      { ids: array.toArray(ids) },
      retrieveReqZ,
      resZ,
    );
    checkForMultipleOrNoResults("Resource", ids, resources, isSingle);
    return isSingle ? resources[0] : resources;
  }
}
