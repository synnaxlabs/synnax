// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, type destructor, primitive, strings } from "@synnaxlabs/x";
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
  bindStores,
  RELATIONSHIPS_STORE_KEY,
  RESOURCES_STORE_KEY,
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

const MOUNT_SCOPE = "ontology.mounts";

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
 * Client-side approximation of the server's matching for a request: exact for
 * id and type sets, permissive for server-computed shapes (search), which
 * accept every change and drift toward the server's answer.
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
  private readonly client: UnaryClient;
  private readonly writer: Writer;
  private readonly relationships_?: cache.Store<string, Relationship>;
  private readonly resources_?: cache.Store<string, Resource>;
  private readonly engine_?: cache.Engine;
  private readonly queries_?: {
    single: cache.Queries<string, Resource>;
    request: cache.Queries<NormalizedRequest, Resource[]>;
    children: cache.Queries<DependentRequest, Resource[]>;
    parents: cache.Queries<DependentRequest, Resource[]>;
  };

  constructor(unary: UnaryClient, engine?: cache.Engine) {
    this.client = unary;
    this.writer = new Writer(unary);
    if (engine == null) return;
    // Reconciliation refetches must bypass the query cache.
    bindStores(engine, async (ids) => await this.execRetrieve({ ids }));
    this.engine_ = engine;
    this.relationships_ = engine.store(RELATIONSHIPS_STORE_KEY);
    this.resources_ = engine.store(RESOURCES_STORE_KEY);
    const ensureStreaming = async () => await engine.ensureStreaming();
    this.queries_ = {
      single: new cache.Queries({
        name: "resource",
        fetch: async (query) => await this.fetchSingle(query),
        mount: (params) => this.mountSingle(params),
        ensureStreaming,
      }),
      request: new cache.Queries({
        name: "resources",
        fetch: async (query) => await this.fetchRequest(query),
        mount: (params) => this.mountRequest(params),
        ensureStreaming,
      }),
      children: new cache.Queries({
        name: "children",
        fetch: async (query) => await this.fetchDependents(query, "to"),
        mount: (params) => this.mountDependents(params, "to"),
        ensureStreaming,
      }),
      parents: new cache.Queries({
        name: "parents",
        fetch: async (query) => await this.fetchDependents(query, "from"),
        mount: (params) => this.mountDependents(params, "from"),
        ensureStreaming,
      }),
    };
  }

  /**
   * Read surface of the relationship cache.
   * @throws when the cache was disabled at client construction.
   */
  get relationships(): cache.Store<string, Relationship> {
    if (this.relationships_ == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.relationships_;
  }

  /**
   * Read surface of the resource cache.
   * @throws when the cache was disabled at client construction.
   */
  get resources(): cache.Store<string, Resource> {
    if (this.resources_ == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.resources_;
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
    if (!Array.isArray(ids) && this.queries_ != null && isSingleCacheable(options))
      return await this.queries_.single.retrieve(idToString(parsedIDs[0]));
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
    if (this.queries_ == null)
      return await this.execRetrieve({
        ids: array.toArray(ids),
        children: true,
        ...options,
      });
    return await this.queries_.children.retrieve(normalizeDependents(ids, options));
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
    if (this.queries_ == null)
      return await this.execRetrieve({
        ids: array.toArray(ids),
        parents: true,
        ...options,
      });
    return await this.queries_.parents.retrieve(normalizeDependents(ids, options));
  }

  /**
   * Subscribes to changes in the cached answer to the given query. Single
   * queries deliver a resource; every other shape delivers the matching
   * resources.
   * @throws when the cache was disabled at client construction.
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
    const queries = this.requireQueries();
    if (!Array.isArray(params) && "key" in params)
      return queries.single.onChange(
        idToString(params),
        handler as cache.ChangeHandler<Resource>,
      );
    const routed = this.routeRequest(Array.isArray(params) ? { ids: params } : params);
    const h = handler as cache.ChangeHandler<Resource[]>;
    if (routed.space === "request") return queries.request.onChange(routed.query, h);
    return queries[routed.space].onChange(routed.query, h);
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached.
   * @throws when the cache was disabled at client construction.
   */
  getCached(params: ID): cache.Cached<Resource> | undefined;
  getCached(params: ID[] | RetrieveRequest): cache.Cached<Resource[]> | undefined;
  getCached(
    params: ID | ID[] | RetrieveRequest,
  ): cache.Cached<Resource> | cache.Cached<Resource[]> | undefined {
    const queries = this.requireQueries();
    if (!Array.isArray(params) && "key" in params)
      return queries.single.getCached(idToString(params));
    const routed = this.routeRequest(Array.isArray(params) ? { ids: params } : params);
    if (routed.space === "request") return queries.request.getCached(routed.query);
    return queries[routed.space].getCached(routed.query);
  }

  /**
   * Adds children to a resource in the ontology.
   * @param id - The ID of the resource to add children to.
   * @param children - The IDs of the children to add.
   */
  async addChildren(id: ID, ...children: ID[]): Promise<void> {
    await this.writer.addChildren(id, ...children);
    const rels = this.relationshipWrites;
    if (rels == null) return;
    children.forEach((child) => {
      const rel = parentRel(id, child);
      rels.set(relationshipToString(rel), rel);
    });
  }

  /**
   * Removes children from a resource in the ontology.
   * @param id - The ID of the resource to remove children from.
   * @param children - The IDs of the children to remove.
   */
  async removeChildren(id: ID, ...children: ID[]): Promise<void> {
    await this.writer.removeChildren(id, ...children);
    const rels = this.relationshipWrites;
    children.forEach((child) =>
      rels?.delete(relationshipToString(parentRel(id, child))),
    );
  }

  /**
   * Moves children from one resource to another in the ontology.
   * @param from - The ID of the resource to move children from.
   * @param to - The ID of the resource to move children to.
   * @param children - The IDs of the children to move.
   */
  async moveChildren(from: ID, to: ID, ...children: ID[]): Promise<void> {
    const rels = this.relationshipWrites;
    const move = (): destructor.Destructor[] => {
      if (rels == null) return [];
      const deletions = children.map((child) =>
        rels.delete(relationshipToString(parentRel(from, child))),
      );
      const insertions = children.map((child) => {
        const rel = parentRel(to, child);
        return rels.set(relationshipToString(rel), rel);
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

  private get relationshipWrites(): cache.UnaryStore<string, Relationship> | undefined {
    return this.engine_?.store(RELATIONSHIPS_STORE_KEY);
  }

  private get resourceStore(): cache.UnaryStore<string, Resource> {
    return this.requireEngine().store(RESOURCES_STORE_KEY);
  }

  private get relationshipStore(): cache.UnaryStore<string, Relationship> {
    return this.requireEngine().store(RELATIONSHIPS_STORE_KEY);
  }

  // Query mounts subscribe in their own scope: stores suppress notifications
  // to listeners in the writer's scope, and the streamer writes in the default
  // scope, which would silence default-scope subscriptions entirely.
  /** Subscribes to every resource set delivered to the cache. */
  onResourceSet(handler: (resource: Resource) => void): destructor.Destructor {
    return this.resourceEvents.onSet(handler);
  }

  /** Subscribes to every resource delete delivered to the cache. */
  onResourceDelete(handler: (id: ID) => void): destructor.Destructor {
    return this.resourceEvents.onDelete((key) => handler(idZ.parse(key)));
  }

  /** Subscribes to every relationship set delivered to the cache. */
  onRelationshipSet(handler: (rel: Relationship) => void): destructor.Destructor {
    return this.relationshipEvents.onSet(handler);
  }

  /** Subscribes to every relationship delete delivered to the cache. */
  onRelationshipDelete(handler: (rel: Relationship) => void): destructor.Destructor {
    return this.relationshipEvents.onDelete((key) => handler(relationshipZ.parse(key)));
  }

  private get resourceEvents(): cache.UnaryStore<string, Resource> {
    return this.requireEngine().store(RESOURCES_STORE_KEY, MOUNT_SCOPE);
  }

  private get relationshipEvents(): cache.UnaryStore<string, Relationship> {
    return this.requireEngine().store(RELATIONSHIPS_STORE_KEY, MOUNT_SCOPE);
  }

  private requireEngine(): cache.Engine {
    if (this.engine_ == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.engine_;
  }

  private requireQueries(): NonNullable<typeof this.queries_> {
    if (this.queries_ == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.queries_;
  }

  private routeRequest(request: RetrieveRequest): Routed {
    const hasIDs = request.ids != null && request.ids.length > 0;
    if (hasIDs && request.children === true && request.parents !== true)
      return {
        space: "children",
        query: normalizeDependents(request.ids as ID[], request),
      };
    if (hasIDs && request.parents === true && request.children !== true)
      return {
        space: "parents",
        query: normalizeDependents(request.ids as ID[], request),
      };
    return { space: "request", query: normalizeRequest(request) };
  }

  private async routeRetrieve(request: RetrieveRequest): Promise<Resource[]> {
    const queries = this.queries_;
    if (queries == null) return await this.execRetrieve(request);
    const routed = this.routeRequest(request);
    if (routed.space === "request") return await queries.request.retrieve(routed.query);
    return await queries[routed.space].retrieve(routed.query);
  }

  /** Writes fetched resources without clobbering cached field data. */
  private writeResources(resources: Resource[]): void {
    resources.forEach((r) =>
      this.resourceStore.set(r.key, (p) =>
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

  private mountSingle({ query, update, remove }: cache.MountParams<string, Resource>) {
    return [
      this.resourceEvents.onSet((resource) => {
        if (resource.key === query) update(resource);
      }),
      this.resourceEvents.onDelete((key) => {
        if (key === query) remove(this.resourceStore.getTombstone(key)?.corpse);
      }),
    ];
  }

  private async fetchRequest(query: NormalizedRequest): Promise<Resource[]> {
    const { ids, ...rest } = query;
    const resources = await this.execRetrieve(
      ids == null ? rest : { ids: parseIDs(ids), ...rest },
    );
    this.writeResources(resources);
    return resources;
  }

  private mountRequest({
    query,
    update,
  }: cache.MountParams<NormalizedRequest, Resource[]>) {
    const matches = requestFilter(query);
    return [
      this.resourceEvents.onSet((resource) => {
        update((prev) => {
          if (prev == null) return prev;
          const existing = prev.some((r) => r.key === resource.key);
          if (!matches(resource))
            return existing ? prev.filter((r) => r.key !== resource.key) : prev;
          if (existing) return prev.map((r) => (r.key === resource.key ? resource : r));
          return [...prev, resource];
        });
      }),
      this.resourceEvents.onDelete((key) => {
        update((prev) => prev?.filter((r) => r.key !== key));
      }),
    ];
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
        this.relationshipStore.set(relationshipToString(rel), rel);
      });
    }
    return resources;
  }

  private mountDependents(
    { query, update }: cache.MountParams<DependentRequest, Resource[]>,
    direction: RelationshipDirection,
  ) {
    const anchor = oppositeRelationshipDirection(direction);
    const ids = new Set(query.ids);
    const types = query.types == null ? null : new Set(query.types);
    const isMember = (rel: Relationship): boolean =>
      rel.type === PARENT_OF_RELATIONSHIP_TYPE &&
      ids.has(idToString(rel[anchor])) &&
      (types == null || types.has(rel[direction].type));
    const isCachedMember = (resource: Resource): boolean =>
      (types == null || types.has(resource.id.type)) &&
      this.relationshipStore.get(
        (rel) =>
          rel.type === PARENT_OF_RELATIONSHIP_TYPE &&
          ids.has(idToString(rel[anchor])) &&
          idsEqual(rel[direction], resource.id),
      ).length > 0;
    const upsert = (prev: Resource[], resource: Resource): Resource[] => {
      if (prev.some((r) => r.key === resource.key))
        return prev.map((r) => (r.key === resource.key ? resource : r));
      return [...prev, resource];
    };
    return [
      this.resourceEvents.onSet((resource) => {
        update((prev) => {
          if (prev == null) return prev;
          if (prev.some((r) => r.key === resource.key) || isCachedMember(resource))
            return upsert(prev, resource);
          return prev;
        });
      }),
      this.resourceEvents.onDelete((key) => {
        update((prev) => prev?.filter((r) => r.key !== key));
      }),
      this.relationshipEvents.onSet((rel) => {
        if (!isMember(rel)) return;
        // Mounts stay synchronous: an uncached member is picked up by the
        // resource set event that follows it.
        const resource = this.resourceStore.get(idToString(rel[direction]));
        if (resource == null) return;
        update((prev) => (prev == null ? prev : upsert(prev, resource)));
      }),
      this.relationshipEvents.onDelete((relKey) => {
        const rel = relationshipZ.parse(relKey);
        if (!isMember(rel)) return;
        const key = idToString(rel[direction]);
        update((prev) => prev?.filter((r) => r.key !== key));
      }),
    ];
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
