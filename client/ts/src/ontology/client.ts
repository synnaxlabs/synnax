// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { array, strings } from "@synnaxlabs/x";
import { z } from "zod";

import { QueryError } from "@/errors";
import { type framer } from "@/framer";
import {
  type ID,
  idToString,
  idZ,
  parseIDs,
  type Resource,
  resourceTypeZ,
  resourceZ,
} from "@/ontology/payload";
import { Writer } from "@/ontology/writer";

const retrieveReqZ = z.object({
  ids: idZ.array().optional(),
  children: z.boolean().optional(),
  parents: z.boolean().optional(),
  excludeFieldData: z.boolean().optional(),
  types: resourceTypeZ.array().optional(),
  searchTerm: z.string().optional(),
  limit: z.int().optional(),
  offset: z.int().optional(),
});
export interface RetrieveRequest extends z.infer<typeof retrieveReqZ> {}

export interface RetrieveOptions extends Pick<
  RetrieveRequest,
  "excludeFieldData" | "types" | "children" | "parents"
> {}

const retrieveResZ = z.object({ resources: resourceZ.array() });

/** The main client class for executing queries against a Synnax cluster ontology */
export class Client {
  readonly type: string = "ontology";
  private readonly client: UnaryClient;
  private readonly writer: Writer;
  private readonly framer: framer.Client;

  constructor(unary: UnaryClient, framer: framer.Client) {
    this.client = unary;
    this.writer = new Writer(unary);
    this.framer = framer;
  }

  /**
   * Retrieves the resource in the ontology with the given ID.
   * @param id - The ID of the resource to retrieve.
   * @param options - Additional options for the retrieval.
   * @param options.excludeFieldData - Whether to exclude the field data of the resource
   * in the results.
   * @returns The resource with the given ID.
   * @throws {QueryError} If no resource is found with the given ID.
   */
  async retrieve(id: ID, options?: RetrieveOptions): Promise<Resource>;

  /**
   * Retrieves the resources in the ontology with the given IDs.
   *
   * @param ids - The IDs of the resources to retrieve.
   * @param options - Additional options for the retrieval.
   * @param options.excludeFieldData - Whether to exclude the field data of the
   * resources in the results.
   * @returns The resources with the given IDs.
   * @throws {QueryError} If no resource is found with any of the given IDs.
   */
  async retrieve(ids: ID[], options?: RetrieveOptions): Promise<Resource[]>;

  async retrieve(params: RetrieveRequest): Promise<Resource[]>;

  async retrieve(
    ids: ID | ID[] | RetrieveRequest,
    options?: RetrieveOptions,
  ): Promise<Resource | Resource[]> {
    if (!Array.isArray(ids) && typeof ids === "object" && !("key" in ids))
      return this.execRetrieve(ids);
    const parsedIDs = parseIDs(ids);
    const resources = await this.execRetrieve({ ids: parsedIDs, ...options });
    if (Array.isArray(ids)) return resources;
    if (resources.length === 0)
      throw new QueryError(
        `No resource found with ID ${strings.naturalLanguageJoin(
          parsedIDs.map((id) => idToString(id)),
        )}`,
      );
    return resources[0];
  }

  /**
   * Retrieves the children of the resources with the given IDs.
   * @param ids - The IDs of the resources whose children to retrieve.
   * @param options - Additional options for the retrieval.
   * the results.
   * @returns The children of the resources with the given IDs.
   */
  async retrieveChildren(
    ids: ID | ID[],
    options?: RetrieveOptions,
  ): Promise<Resource[]> {
    return await this.execRetrieve({
      ids: array.toArray(ids),
      children: true,
      ...options,
    });
  }

  /**
   * Retrieves the parents of the resources with the given IDs.
   *
   * @param ids - the IDs of the resources whose parents to retrieve
   * @param options - additional options for the retrieval
   * @param options.excludeFieldData - whether to exclude the field data of the parents
   * in the results
   * @returns the parents of the resources with the given IDs
   */
  async retrieveParents(
    ids: ID | ID[],
    options?: RetrieveOptions,
  ): Promise<Resource[]> {
    return await this.execRetrieve({
      ids: array.toArray(ids),
      parents: true,
      ...options,
    });
  }

  /**
   * Adds children to a resource in the ontology.
   * @param id - The ID of the resource to add children to.
   * @param children - The IDs of the children to add.
   */
  async addChildren(id: ID, ...children: ID[]): Promise<void> {
    return await this.writer.addChildren(id, ...children);
  }

  /**
   * Removes children from a resource in the ontology.
   * @param id - The ID of the resource to remove children from.
   * @param children - The IDs of the children to remove.
   */
  async removeChildren(id: ID, ...children: ID[]): Promise<void> {
    return await this.writer.removeChildren(id, ...children);
  }

  /**
   * Moves children from one resource to another in the ontology.
   * @param from - The ID of the resource to move children from.
   * @param to - The ID of the resource to move children to.
   * @param children - The IDs of the children to move.
   */
  async moveChildren(from: ID, to: ID, ...children: ID[]): Promise<void> {
    return await this.writer.moveChildren(from, to, ...children);
  }

  private async execRetrieve(request: RetrieveRequest): Promise<Resource[]> {
    const { resources } = await sendRequired(
      this.client,
      "/ontology/retrieve",
      request,
      retrieveReqZ,
      retrieveResZ,
    );
    return resources;
  }
}

export const RESOURCE_SET_CHANNEL_NAME = "sy_ontology_resource_set";
export const RESOURCE_DELETE_CHANNEL_NAME = "sy_ontology_resource_delete";
export const RELATIONSHIP_SET_CHANNEL_NAME = "sy_ontology_relationship_set";
export const RELATIONSHIP_DELETE_CHANNEL_NAME = "sy_ontology_relationship_delete";
