// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { observe, toArray } from "@synnaxlabs/x";
import { type AsyncTermSearcher } from "@synnaxlabs/x/search";
import { z } from "zod";

import { QueryError } from "@/errors";
import { type framer } from "@/framer";
import { group } from "@/ontology/group";
import {
  type CrudeID,
  ID,
  type IDPayload,
  idZ,
  parseRelationship,
  type RelationshipChange,
  type RelationshipDirection,
  type Resource,
  type ResourceChange,
  resourceSchemaZ,
  resourceTypeZ,
} from "@/ontology/payload";
import { Writer } from "@/ontology/writer";

const RETRIEVE_ENDPOINT = "/ontology/retrieve";

const retrieveReqZ = z.object({
  ids: idZ.array().optional(),
  children: z.boolean().optional(),
  parents: z.boolean().optional(),
  includeSchema: z.boolean().optional(),
  excludeFieldData: z.boolean().optional(),
  term: z.string().optional(),
  limit: z.number().optional(),
  offset: z.number().optional(),
  types: resourceTypeZ.array().optional(),
});

type RetrieveRequest = z.infer<typeof retrieveReqZ>;

export type RetrieveOptions = Pick<
  RetrieveRequest,
  "includeSchema" | "excludeFieldData" | "types"
>;

const retrieveResZ = z.object({
  resources: resourceSchemaZ.array(),
});

export const parseIDs = (ids: CrudeID | CrudeID[] | string | string[]): IDPayload[] =>
  toArray(ids).map((id) => new ID(id).payload);

/** The core client class for executing queries against a Synnax cluster ontology */
export class Client implements AsyncTermSearcher<string, string, Resource> {
  readonly type: string = "ontology";
  groups: group.Client;
  private readonly client: UnaryClient;
  private readonly writer: Writer;
  private readonly framer: framer.Client;

  constructor(unary: UnaryClient, framer: framer.Client) {
    this.client = unary;
    this.writer = new Writer(unary);
    this.groups = new group.Client(unary);
    this.framer = framer;
  }

  /**
   * Executes a fuzzy search on the ontology for resources with names/fields similar to the
   * given term.
   *
   * @param term The search term.
   * @param options Additional options for the search.
   * @param options.includeSchema Whether to include the schema of the resources in the
   * results.
   * @param options.excludeFieldData Whether to exclude the field data of the resources in
   * the results.
   * @returns A list of resources that match the search term.
   */
  async search(term: string, options?: RetrieveOptions): Promise<Resource[]> {
    return await this.execRetrieve({ term, ...options });
  }

  /**
   * Retrieves the resource in the ontology with the given ID.
   * @param id - The ID of the resource to retrieve.
   * @param options - Additional options for the retrieval.
   * @param options.includeSchema - Whether to include the schema of the resource in the
   * results.
   * @param options.excludeFieldData - Whether to exclude the field data of the resource
   * in the results.
   * @returns The resource with the given ID.
   * @throws {QueryError} If no resource is found with the given ID.
   */
  async retrieve(id: CrudeID, options?: RetrieveOptions): Promise<Resource>;

  /**
   * Retrieves the resources in the ontology with the given IDs.
   *
   * @param ids - The IDs of the resources to retrieve.
   * @param options - Additional options for the retrieval.
   * @param options.includeSchema - Whether to include the schema of the resources in
   * the results.
   * @param options.excludeFieldData - Whether to exclude the field data of the
   * resources in the results.
   * @returns The resources with the given IDs.
   * @throws {QueryError} If no resource is found with any of the given IDs.
   */
  async retrieve(ids: CrudeID[], options?: RetrieveOptions): Promise<Resource[]>;

  async retrieve(
    ids: CrudeID | CrudeID[],
    options?: RetrieveOptions,
  ): Promise<Resource | Resource[]> {
    const resources = await this.execRetrieve({ ids: parseIDs(ids), ...options });
    if (Array.isArray(ids)) return resources;
    if (resources.length === 0)
      throw new QueryError(`No resource found with ID ${ids.toString()}`);
    return resources[0];
  }

  /**
   * Retrieves resources from the ontology in a paginated manner.
   *
   * @param offset - The offset of the page (i.e. how many resources to skip before
   * returning results).
   * @param limit - The maximum number of resources to return.
   * @param options - Additional options for the retrieval.
   * @returns A list of resources in the ontology.
   */
  async page(
    offset: number,
    limit: number,
    options?: RetrieveOptions,
  ): Promise<Resource[]> {
    return await this.execRetrieve({ offset, limit, ...options });
  }

  /**
   * Retrieves the children of the resources with the given IDs.
   * @param ids - The IDs of the resources whose children to retrieve.
   * @param options - Additional options for the retrieval.
   * @param options.includeSchema - Whether to include the schema of the children in the
   * results.
   * @param options.excludeFieldData - Whether to exclude the field data of the children in
   * the results.
   * @returns The children of the resources with the given IDs.
   */
  async retrieveChildren(
    ids: CrudeID | CrudeID[],
    options?: RetrieveOptions,
  ): Promise<Resource[]> {
    return await this.execRetrieve({ ids: parseIDs(ids), children: true, ...options });
  }

  /**
   * Retrieves the parents of the resources with the given IDs.
   *
   * @param ids - the IDs of the resources whose parents to retrieve
   * @param options - additional options for the retrieval
   * @param options.includeSchema - whether to include the schema of the parents in the
   * results
   * @param options.excludeFieldData - whether to exclude the field data of the parents
   * in the results
   * @returns the parents of the resources with the given IDs
   */
  async retrieveParents(
    ids: CrudeID | CrudeID[],
    options?: RetrieveOptions,
  ): Promise<Resource[]> {
    return await this.execRetrieve({ ids: parseIDs(ids), parents: true, ...options });
  }

  /**
   * Adds children to a resource in the ontology.
   * @param id - The ID of the resource to add children to.
   * @param children - The IDs of the children to add.
   */
  async addChildren(id: CrudeID, ...children: CrudeID[]): Promise<void> {
    return await this.writer.addChildren(id, ...children);
  }

  /**
   * Removes children from a resource in the ontology.
   * @param id - The ID of the resource to remove children from.
   * @param children - The IDs of the children to remove.
   */
  async removeChildren(id: CrudeID, ...children: CrudeID[]): Promise<void> {
    return await this.writer.removeChildren(id, ...children);
  }

  /**
   * Moves children from one resource to another in the ontology.
   * @param from - The ID of the resource to move children from.
   * @param to - The ID of the resource to move children to.
   * @param children - The IDs of the children to move.
   */
  async moveChildren(
    from: CrudeID,
    to: CrudeID,
    ...children: CrudeID[]
  ): Promise<void> {
    return await this.writer.moveChildren(from, to, ...children);
  }

  /**
   * Opens an observable that can be used to subscribe to changes in both the ontology's
   * resources and relationships.
   * @link ChangeTracker for more information.
   * @returns An observable that emits changes to the ontology's resources and relationships.
   */
  async openChangeTracker(): Promise<ChangeTracker> {
    return await ChangeTracker.open(this.framer, this);
  }

  async openDependentTracker(
    props: DependentTrackerProps,
  ): Promise<observe.ObservableAsyncCloseable<Resource[]>> {
    return await DependentTracker.open(props, this.framer, this);
  }

  newSearcherWithOptions(
    options: RetrieveOptions,
  ): AsyncTermSearcher<string, string, Resource> {
    return {
      type: this.type,
      search: (term: string) => this.search(term, options),
      retrieve: (ids: string[]) => this.retrieve(ids, options),
      page: (offset: number, limit: number) => this.page(offset, limit, options),
    };
  }

  private async execRetrieve(request: RetrieveRequest): Promise<Resource[]> {
    const { resources } = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      request,
      retrieveReqZ,
      retrieveResZ,
    );
    return resources;
  }
}

const RESOURCE_SET_NAME = "sy_ontology_resource_set";
const RESOURCE_DELETE_NAME = "sy_ontology_resource_delete";
const RELATIONSHIP_SET_NAME = "sy_ontology_relationship_set";
const RELATIONSHIP_DELETE_NAME = "sy_ontology_relationship_delete";

/**
 * A class that tracks changes to the ontology's resources and relationships.
 */
export class ChangeTracker {
  /**
   * An observable that emits changes to the ontology's relationships.
   */
  readonly relationships: observe.Observable<RelationshipChange[]>;
  /**
   * An observable that emits changes to the ontology's resources.
   */
  readonly resources: observe.Observable<ResourceChange[]>;

  private readonly resourceObs: observe.Observer<ResourceChange[]>;
  private readonly relationshipObs: observe.Observer<RelationshipChange[]>;
  private readonly streamer: framer.Streamer;
  private readonly client: Client;
  private readonly closePromise: Promise<void>;

  constructor(streamer: framer.Streamer, client: Client) {
    this.relationshipObs = new observe.Observer<RelationshipChange[]>();
    this.relationships = this.relationshipObs;
    this.resourceObs = new observe.Observer<ResourceChange[]>();
    this.resources = this.resourceObs;
    this.client = client;
    this.streamer = streamer;
    this.closePromise = this.start();
  }

  async close(): Promise<void> {
    this.streamer.close();
    await this.closePromise;
  }

  private async start(): Promise<void> {
    for await (const frame of this.streamer) await this.update(frame);
  }

  private async update(frame: framer.Frame): Promise<void> {
    const resSets = await this.parseResourceSets(frame);
    const resDeletes = this.parseResourceDeletes(frame);
    const allResources = resSets.concat(resDeletes);
    if (allResources.length > 0) this.resourceObs.notify(resSets.concat(resDeletes));
    const relSets = this.parseRelationshipSets(frame);
    const relDeletes = this.parseRelationshipDeletes(frame);
    const allRelationships = relSets.concat(relDeletes);
    if (allRelationships.length > 0)
      this.relationshipObs.notify(relSets.concat(relDeletes));
  }

  private parseRelationshipSets(frame: framer.Frame): RelationshipChange[] {
    const relationships = frame.get(RELATIONSHIP_SET_NAME);
    if (relationships.length === 0) return [];
    return Array.from(relationships.as("string")).map((rel) => ({
      variant: "set",
      key: parseRelationship(rel),
      value: undefined,
    }));
  }

  private parseRelationshipDeletes(frame: framer.Frame): RelationshipChange[] {
    const relationships = frame.get(RELATIONSHIP_DELETE_NAME);
    if (relationships.length === 0) return [];
    return Array.from(relationships.as("string")).map((rel) => ({
      variant: "delete",
      key: parseRelationship(rel),
    }));
  }

  private async parseResourceSets(frame: framer.Frame): Promise<ResourceChange[]> {
    const sets = frame.get(RESOURCE_SET_NAME);
    if (sets.length === 0) return [];
    // We should only ever get one series of sets
    const ids = Array.from(sets.as("string")).map((id: string) => new ID(id));
    try {
      const resources = await this.client.retrieve(ids);
      return resources.map((resource) => ({
        variant: "set",
        key: resource.id,
        value: resource,
      }));
    } catch (e) {
      if (e instanceof QueryError) return [];
      throw e;
    }
  }

  private parseResourceDeletes(frame: framer.Frame): ResourceChange[] {
    const deletes = frame.get(RESOURCE_DELETE_NAME);
    if (deletes.length === 0) return [];
    // We should only ever get one series of deletes
    return Array.from(deletes.as("string")).map((str) => ({
      variant: "delete",
      key: new ID(str),
    }));
  }

  static async open(client: framer.Client, retriever: Client): Promise<ChangeTracker> {
    const streamer = await client.openStreamer([
      RESOURCE_SET_NAME,
      RESOURCE_DELETE_NAME,
      RELATIONSHIP_SET_NAME,
      RELATIONSHIP_DELETE_NAME,
    ]);
    return new ChangeTracker(streamer, retriever);
  }
}

const oppositeDirection = (dir: RelationshipDirection): RelationshipDirection =>
  dir === "from" ? "to" : "from";

interface DependentTrackerProps {
  target: ID;
  dependents: Resource[];
  relationshipType?: string;
  relationshipDirection?: RelationshipDirection;
  resourceType?: string;
}

/**
 * A class that tracks a resource (called the 'target' resource) and related resources
 * (called 'dependents') of a particular type (called the 'type') in a Synnax cluster
 * ontology.
 */
export class DependentTracker
  extends observe.Observer<Resource[]>
  implements observe.ObservableAsyncCloseable<Resource[]>
{
  private readonly internal: ChangeTracker;
  private readonly target: ID;
  private readonly relDir: RelationshipDirection;
  private readonly resourceType?: string;
  private dependents: Resource[];
  private readonly client: Client;
  private readonly relType: string;

  private constructor(
    {
      target,
      dependents,
      relationshipType = "parent",
      relationshipDirection = "from",
      resourceType,
    }: DependentTrackerProps,
    internal: ChangeTracker,
    client: Client,
  ) {
    super();
    this.resourceType = resourceType;
    this.internal = internal;
    this.target = target;
    this.dependents = dependents;
    if (this.resourceType != null)
      this.dependents = this.dependents.filter((r) => r.id.type === this.resourceType);
    this.client = client;
    this.relType = relationshipType;
    this.relDir = relationshipDirection;
    this.internal.resources.onChange(this.handleResourceChange);
    this.internal.relationships.onChange(this.handleRelationshipChange);
  }
  static async open(
    props: DependentTrackerProps,
    framer: framer.Client,
    client: Client,
  ): Promise<DependentTracker> {
    const internal = await ChangeTracker.open(framer, client);
    return new DependentTracker(props, internal, client);
  }

  private handleResourceChange = (changes: ResourceChange[]): void => {
    this.dependents = this.dependents.map((child) => {
      const change = changes.find((c) => c.key.toString() == child.id.toString());
      if (change == null || change.variant === "delete") return child;
      return change.value;
    });
    this.notify(this.dependents);
  };

  private handleRelationshipChange = (changes: RelationshipChange[]): void => {
    const deletes = changes.filter(
      (c) =>
        c.variant === "delete" &&
        c.key[this.relDir].toString() === this.target.toString() &&
        (this.resourceType == null ||
          c.key[oppositeDirection(this.relDir)].type === this.resourceType),
    );
    this.dependents = this.dependents.filter(
      (child) =>
        !deletes.some(
          (del) =>
            del.key.to.toString() === child.id.toString() &&
            del.key.type === this.relType,
        ),
    );
    const sets = changes.filter(
      (c) =>
        c.variant === "set" &&
        c.key.type === this.relType &&
        c.key[this.relDir].toString() === this.target.toString() &&
        (this.resourceType == null ||
          c.key[oppositeDirection(this.relDir)].type === this.resourceType),
    );
    if (sets.length === 0) return this.notify(this.dependents);
    this.client.retrieve(sets.map((s) => s.key.to)).then((resources) => {
      this.dependents = this.dependents.concat(resources);
      this.notify(this.dependents);
    });
  };

  async close(): Promise<void> {
    await this.internal.close();
  }
}
