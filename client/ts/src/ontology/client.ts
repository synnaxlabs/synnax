// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { deep, type destructor, primitive, strings } from "@synnaxlabs/x";
import { z } from "zod";

import { cache } from "@/cache";
import { QueryError } from "@/errors";
import {
  type ID,
  idsEqual,
  idToString,
  idZ,
  oppositeRelationshipDirection,
  PARENT_OF_RELATIONSHIP_TYPE,
  parseIDs,
  type Relationship,
  type RelationshipDirection,
  relationshipToString,
  relationshipZ,
  type Resource,
  type ResourceType,
  resourceTypeZ,
  resourceZ,
} from "@/ontology/payload";
import {
  RELATIONSHIP_DELETE_CHANNEL_NAME,
  RELATIONSHIP_SET_CHANNEL_NAME,
  RESOURCE_DELETE_CHANNEL_NAME,
  RESOURCE_SET_CHANNEL_NAME,
  type Stores,
} from "@/ontology/store";
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

const parentRel = (from: ID, to: ID): Relationship => ({
  from,
  type: PARENT_OF_RELATIONSHIP_TYPE,
  to,
});

/** A retrieve request with IDs normalized to stable, sorted strings. */
type NormalizedRequest = {
  ids?: string[];
  children?: boolean;
  parents?: boolean;
  excludeFieldData?: boolean;
  types?: ResourceType[];
  searchTerm?: string;
  limit?: number;
  offset?: number;
};

/** Request key for children/parents listings of a set of resources. */
type DependentRequest = {
  ids: string[];
  types?: ResourceType[];
  searchTerm?: string;
  offset?: number;
  limit?: number;
  excludeFieldData?: boolean;
};

type DependentOptions = Pick<
  RetrieveRequest,
  "types" | "searchTerm" | "offset" | "limit" | "excludeFieldData"
>;

/** Query fields only the server can evaluate. Dependent flags reach the
 *  request space only in shapes it cannot check locally. */
const REQUEST_SERVER_FIELDS = [
  "searchTerm",
  "limit",
  "offset",
  "children",
  "parents",
] as const;

const DEPENDENT_SERVER_FIELDS = ["searchTerm", "limit", "offset"] as const;

const normalizeIDs = (ids: ID | ID[]): string[] =>
  [...new Set(idToString(parseIDs(ids)))].sort();

const normalizeDependents = (
  ids: ID | ID[],
  options?: DependentOptions,
): DependentRequest => {
  const out: DependentRequest = { ids: normalizeIDs(ids) };
  const { types, searchTerm, offset, limit, excludeFieldData } = options ?? {};
  if (types != null) out.types = [...types].sort();
  if (searchTerm != null) out.searchTerm = searchTerm;
  if (offset != null) out.offset = offset;
  if (limit != null) out.limit = limit;
  if (excludeFieldData != null) out.excludeFieldData = excludeFieldData;
  return out;
};

const normalizeRequest = (req: RetrieveRequest): NormalizedRequest => {
  const out: NormalizedRequest = {};
  if (req.ids != null) out.ids = normalizeIDs(req.ids);
  if (req.children != null) out.children = req.children;
  if (req.parents != null) out.parents = req.parents;
  if (req.excludeFieldData != null) out.excludeFieldData = req.excludeFieldData;
  if (req.types != null) out.types = [...req.types].sort();
  if (req.searchTerm != null) out.searchTerm = req.searchTerm;
  if (req.limit != null) out.limit = req.limit;
  if (req.offset != null) out.offset = req.offset;
  return out;
};

// Type filters and dependent flags change the server's answer, so only plain
// single reads are served by the single query space.
const isSingleCacheable = (options?: RetrieveOptions): boolean =>
  options == null ||
  (options.children !== true && options.parents !== true && options.types == null);

/**
 * Client-side matching for a request: exact for id and type sets.
 * Server-computed shapes (search, limit/offset, dependents) never reach this
 * filter; they refetch instead.
 */
const requestFilter = (req: NormalizedRequest): ((r: Resource) => boolean) => {
  const idSet = primitive.isNonZero(req.ids) ? new Set(req.ids) : undefined;
  const typeSet = primitive.isNonZero(req.types) ? new Set(req.types) : undefined;
  return (r) => {
    if (idSet != null && !idSet.has(r.key)) return false;
    if (typeSet != null && !typeSet.has(r.id.type)) return false;
    return true;
  };
};

type Routed =
  | { space: "children" | "parents"; query: DependentRequest }
  | { space: "request"; query: NormalizedRequest };

/** The main client class for executing queries against a Synnax cluster ontology */
export class Client {
  readonly type: string = "ontology";
  /** The ontology tables injected into sibling domain clients at wiring. */
  readonly stores: Stores;
  private readonly client: UnaryClient;
  private readonly writer: Writer;
  private readonly answers: {
    single: cache.Answers<string, Resource, string, Resource>;
    request: cache.Answers<NormalizedRequest, Resource[], string, Resource>;
    children: cache.Answers<DependentRequest, Resource[], string, Resource>;
    parents: cache.Answers<DependentRequest, Resource[], string, Resource>;
  };

  constructor(unary: UnaryClient, engine: cache.Cache) {
    this.client = unary;
    this.writer = new Writer(unary);
    this.stores = this.createTables(engine);
    this.answers = {
      single: engine.answers({
        name: "resource",
        table: this.resources,
        fetch: async (query) => [(await this.fetchSingle(query)).key],
        compose: (records) => records[0],
        keyOf: (query) => query,
        single: true,
      }),
      request: engine.answers({
        name: "resources",
        table: this.resources,
        fetch: async (query) => (await this.fetchRequest(query)).map((r) => r.key),
        compose: (records) => records,
        matches: (resource, query) => requestFilter(query)(resource),
        serverFields: REQUEST_SERVER_FIELDS,
      }),
      children: this.dependentAnswers(engine, "children", "to"),
      parents: this.dependentAnswers(engine, "parents", "from"),
    };
  }

  private createTables(engine: cache.Cache): Stores {
    const relationships = engine.createTable<string, Relationship>({
      name: "relationships",
      equal: (a, b) =>
        idsEqual(a.from, b.from) && idsEqual(a.to, b.to) && a.type === b.type,
    });
    const relationshipSet: cache.ChannelListener<typeof relationshipZ> = {
      channel: RELATIONSHIP_SET_CHANNEL_NAME,
      schema: relationshipZ,
      onChange: (changed) => relationships.set(relationshipToString(changed), changed),
    };
    const relationshipDelete: cache.ChannelListener<typeof relationshipZ> = {
      channel: RELATIONSHIP_DELETE_CHANNEL_NAME,
      schema: relationshipZ,
      onChange: (changed) => relationships.delete(relationshipToString(changed)),
    };
    engine.addListeners(relationships, relationshipSet, relationshipDelete);

    const resources = engine.createTable<string, Resource>({
      name: "resources",
      equal: (a, b) => deep.equal(a, b),
      // Reconciliation refetches must bypass the query cache.
      refetch: async (keys) => await this.execRetrieve({ ids: parseIDs(keys) }),
    });
    const resourceSet: cache.ChannelListener<typeof resourceZ> = {
      channel: RESOURCE_SET_CHANNEL_NAME,
      schema: resourceZ,
      onChange: (changed) =>
        resources.set(changed.key, (p) => (p == null ? changed : { ...p, ...changed })),
    };
    const resourceDelete: cache.ChannelListener<typeof idZ> = {
      channel: RESOURCE_DELETE_CHANNEL_NAME,
      schema: idZ,
      // The store is keyed by the full "type:key" string, not the bare key.
      onChange: (changed) => resources.delete(idToString(changed)),
    };
    engine.addListeners(resources, resourceSet, resourceDelete);
    return { relationships, resources };
  }

  /** Read surface of the relationship cache. */
  get relationships(): cache.Table<string, Relationship> {
    return this.stores.relationships;
  }

  /** Read surface of the resource cache. */
  get resources(): cache.Table<string, Resource> {
    return this.stores.resources;
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
      return await this.routeRetrieve(ids);
    const parsedIDs = parseIDs(ids);
    if (!Array.isArray(ids) && isSingleCacheable(options))
      return await this.answers.single.retrieve(idToString(parsedIDs[0]));
    const resources = await this.routeRetrieve({ ids: parsedIDs, ...options });
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
    return await this.answers.children.retrieve(normalizeDependents(ids, options));
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
    return await this.answers.parents.retrieve(normalizeDependents(ids, options));
  }

  /**
   * Subscribes to changes in the cached answer to the given query. Single
   * queries deliver a resource; every other shape delivers the matching
   * resources.
   */
  onChange(params: ID, handler: cache.ChangeHandler<Resource>): destructor.Destructor;
  onChange(
    params: ID[] | RetrieveRequest,
    handler: cache.ChangeHandler<Resource[]>,
  ): destructor.Destructor;
  onChange(
    params: ID | ID[] | RetrieveRequest,
    handler: cache.ChangeHandler<Resource> | cache.ChangeHandler<Resource[]>,
  ): destructor.Destructor {
    if (!Array.isArray(params) && "key" in params)
      return this.answers.single.onChange(
        idToString(params),
        handler as cache.ChangeHandler<Resource>,
      );
    const routed = this.routeRequest(Array.isArray(params) ? { ids: params } : params);
    const h = handler as cache.ChangeHandler<Resource[]>;
    if (routed.space === "request")
      return this.answers.request.onChange(routed.query, h);
    return this.answers[routed.space].onChange(routed.query, h);
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached.
   */
  getCached(params: ID): cache.Cached<Resource> | undefined;
  getCached(params: ID[] | RetrieveRequest): cache.Cached<Resource[]> | undefined;
  getCached(
    params: ID | ID[] | RetrieveRequest,
  ): cache.Cached<Resource> | cache.Cached<Resource[]> | undefined {
    if (!Array.isArray(params) && "key" in params)
      return this.answers.single.getCached(idToString(params));
    const routed = this.routeRequest(Array.isArray(params) ? { ids: params } : params);
    if (routed.space === "request") return this.answers.request.getCached(routed.query);
    return this.answers[routed.space].getCached(routed.query);
  }

  /**
   * Adds children to a resource in the ontology.
   * @param id - The ID of the resource to add children to.
   * @param children - The IDs of the children to add.
   */
  async addChildren(id: ID, ...children: ID[]): Promise<void> {
    await this.writer.addChildren(id, ...children);
    children.forEach((child) => {
      const rel = parentRel(id, child);
      this.relationships.set(relationshipToString(rel), rel);
    });
  }

  /**
   * Removes children from a resource in the ontology.
   * @param id - The ID of the resource to remove children from.
   * @param children - The IDs of the children to remove.
   */
  async removeChildren(id: ID, ...children: ID[]): Promise<void> {
    await this.writer.removeChildren(id, ...children);
    children.forEach((child) =>
      this.relationships.delete(relationshipToString(parentRel(id, child))),
    );
  }

  /**
   * Moves children from one resource to another in the ontology.
   * @param from - The ID of the resource to move children from.
   * @param to - The ID of the resource to move children to.
   * @param children - The IDs of the children to move.
   */
  async moveChildren(from: ID, to: ID, ...children: ID[]): Promise<void> {
    const move = (): destructor.Destructor[] => {
      const deletions = children.map((child) =>
        this.relationships.delete(relationshipToString(parentRel(from, child))),
      );
      const insertions = children.map((child) => {
        const rel = parentRel(to, child);
        return this.relationships.set(relationshipToString(rel), rel);
      });
      return [...deletions, ...insertions];
    };
    const rollback = new cache.Rollback();
    rollback.add(...move());
    await rollback.guard(
      async () => await this.writer.moveChildren(from, to, ...children),
    );
    // Re-applied after success: a stale streamer echo may have clobbered the
    // optimistic writes while the send was in flight.
    move();
  }

  /** Subscribes to every resource set delivered to the cache. */
  onResourceSet(handler: (resource: Resource) => void): destructor.Destructor {
    return this.resources.subscribe((event) => {
      if (event.variant === "set") handler(event.value);
    });
  }

  /** Subscribes to every resource delete delivered to the cache. */
  onResourceDelete(handler: (id: ID) => void): destructor.Destructor {
    return this.resources.subscribe((event) => {
      if (event.variant === "delete") handler(idZ.parse(event.key));
    });
  }

  /** Subscribes to every relationship set delivered to the cache. */
  onRelationshipSet(handler: (rel: Relationship) => void): destructor.Destructor {
    return this.relationships.subscribe((event) => {
      if (event.variant === "set") handler(event.value);
    });
  }

  /** Subscribes to every relationship delete delivered to the cache. */
  onRelationshipDelete(handler: (rel: Relationship) => void): destructor.Destructor {
    return this.relationships.subscribe((event) => {
      if (event.variant === "delete") handler(relationshipZ.parse(event.key));
    });
  }

  private routeRequest(request: RetrieveRequest): Routed {
    const { ids } = request;
    const hasIDs = ids != null && ids.length > 0;
    if (hasIDs && request.children === true && request.parents !== true)
      return {
        space: "children",
        query: normalizeDependents(ids, request),
      };
    if (hasIDs && request.parents === true && request.children !== true)
      return {
        space: "parents",
        query: normalizeDependents(ids, request),
      };
    return { space: "request", query: normalizeRequest(request) };
  }

  private async routeRetrieve(request: RetrieveRequest): Promise<Resource[]> {
    const routed = this.routeRequest(request);
    if (routed.space === "request")
      return await this.answers.request.retrieve(routed.query);
    return await this.answers[routed.space].retrieve(routed.query);
  }

  /** Writes fetched resources without clobbering cached field data. */
  private writeResources(resources: Resource[]): void {
    resources.forEach((r) =>
      this.resources.set(r.key, (p) =>
        p == null ? r : { ...r, data: r.data ?? p.data },
      ),
    );
  }

  // Resources are deleted through other domain clients whose write-through
  // never reaches the resource store, so fetches always hit the network
  // rather than serving a possibly stale cached entry.
  private async fetchSingle(query: string): Promise<Resource> {
    const resources = await this.execRetrieve({ ids: [idZ.parse(query)] });
    if (resources.length === 0)
      throw new QueryError(`No resource found with ID ${query}`);
    this.writeResources(resources);
    return resources[0];
  }

  private async fetchRequest(query: NormalizedRequest): Promise<Resource[]> {
    const { ids, ...rest } = query;
    const resources = await this.execRetrieve(
      ids == null ? rest : { ids: parseIDs(ids), ...rest },
    );
    this.writeResources(resources);
    return resources;
  }

  private async fetchDependents(
    query: DependentRequest,
    direction: RelationshipDirection,
  ): Promise<Resource[]> {
    const { ids, ...rest } = query;
    const resources = await this.execRetrieve({
      ids: parseIDs(ids),
      children: direction === "to",
      parents: direction === "from",
      ...rest,
    });
    this.writeResources(resources);
    // Multi-anchor answers can't attribute members to an anchor, so only
    // single-anchor queries write relationships through.
    if (ids.length === 1) {
      const anchor = idZ.parse(ids[0]);
      resources.forEach(({ id }) => {
        const rel = direction === "to" ? parentRel(anchor, id) : parentRel(id, anchor);
        this.relationships.set(relationshipToString(rel), rel);
      });
    }
    return resources;
  }

  private dependentAnswers(
    engine: cache.Cache,
    name: string,
    direction: RelationshipDirection,
  ): cache.Answers<DependentRequest, Resource[], string, Resource> {
    const anchor = oppositeRelationshipDirection(direction);
    return engine.answers({
      name,
      table: this.resources,
      fetch: async (query) =>
        (await this.fetchDependents(query, direction)).map((r) => r.key),
      compose: (records) => records,
      matches: (resource, query) => this.isDependent(resource, query, direction),
      serverFields: DEPENDENT_SERVER_FIELDS,
      watch: [
        cache.watch(this.relationships, (event, query: DependentRequest) => {
          const rel =
            event.variant === "set" ? event.value : relationshipZ.parse(event.key);
          if (rel.type !== PARENT_OF_RELATIONSHIP_TYPE) return null;
          if (!query.ids.includes(idToString(rel[anchor]))) return null;
          if (query.types != null && !query.types.includes(rel[direction].type))
            return null;
          return [idToString(rel[direction])];
        }),
      ],
      hydrate: async (keys) => {
        await this.fetchRequest({ ids: keys });
      },
    });
  }

  /** Whether a cached relationship makes the resource a dependent of the query. */
  private isDependent(
    resource: Resource,
    query: DependentRequest,
    direction: RelationshipDirection,
  ): boolean {
    if (query.types != null && !query.types.includes(resource.id.type)) return false;
    const anchor = oppositeRelationshipDirection(direction);
    return (
      this.relationships.get(
        (rel) =>
          rel.type === PARENT_OF_RELATIONSHIP_TYPE &&
          query.ids.includes(idToString(rel[anchor])) &&
          idsEqual(rel[direction], resource.id),
      ).length > 0
    );
  }

  private async execRetrieve(request: RetrieveRequest): Promise<Resource[]> {
    const { resources } = await this.client.send(
      "/ontology/retrieve",
      request,
      retrieveReqZ,
      retrieveResZ,
    );
    return resources;
  }
}
