// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { deep, destructor, primitive } from "@synnaxlabs/x";
import { z } from "zod";

import { NotFoundError } from "@/errors";
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
import { query } from "@/query";

const wireReqZ = z.object({
  ids: idZ.array().optional(),
  children: z.boolean().optional(),
  parents: z.boolean().optional(),
  excludeFieldData: z.boolean().optional(),
  types: resourceTypeZ.array().optional(),
  searchTerm: z.string().optional(),
  limit: z.int().optional(),
  offset: z.int().optional(),
  ignoreNotFoundError: z.boolean().optional(),
});
export interface RetrieveRequest extends z.infer<typeof wireReqZ> {}

export interface RetrieveOptions extends Pick<
  RetrieveRequest,
  "excludeFieldData" | "types"
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

/** Query addressing the children or parents of a set of resources. */
export type DependentParams = {
  ids: ID | ID[];
  types?: ResourceType[];
  searchTerm?: string;
  offset?: number;
  limit?: number;
  excludeFieldData?: boolean;
};

const DEPENDENT_SERVER_FIELDS = ["searchTerm", "limit", "offset"] as const;

const normalizeIDs = (ids: ID | ID[]): string[] =>
  [...new Set(idToString(parseIDs(ids)))].sort();

const normalizeDependents = (
  ids: ID | ID[],
  options?: Omit<DependentParams, "ids">,
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

const retrieveRequestZ = z
  .object({
    ids: idZ.array().optional(),
    excludeFieldData: z.boolean().optional(),
    types: resourceTypeZ.array().optional(),
    searchTerm: z.string().optional(),
    limit: z.int().optional(),
    offset: z.int().optional(),
  })
  .transform((req): NormalizedRequest => {
    const out: NormalizedRequest = {};
    if (req.ids != null) out.ids = normalizeIDs(req.ids);
    if (req.excludeFieldData != null) out.excludeFieldData = req.excludeFieldData;
    if (req.types != null) out.types = [...req.types].sort();
    if (req.searchTerm != null) out.searchTerm = req.searchTerm;
    if (req.limit != null) out.limit = req.limit;
    if (req.offset != null) out.offset = req.offset;
    return out;
  });

export type RetrieveParams = z.input<typeof retrieveRequestZ>;

/**
 * Client-side matching for a request: exact for id and type sets.
 * Server-computed shapes (search, limit/offset) never reach this filter; they
 * refetch instead.
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

export interface ClientParams {
  unary: UnaryClient;
  cache: query.Cache;
}

/** The main client class for executing queries against a Synnax cluster ontology */
export class Client extends query.Retriever<
  typeof retrieveRequestZ,
  string,
  Resource,
  Resource,
  ID
> {
  readonly type: string = "ontology";
  /** The ontology tables injected into sibling domain clients at wiring. */
  readonly stores: Stores;
  /** Cached read surface for the children of a set of resources. */
  readonly children: query.Retrieves<DependentParams, Resource[]>;
  /** Cached read surface for the parents of a set of resources. */
  readonly parents: query.Retrieves<DependentParams, Resource[]>;
  private readonly unary: UnaryClient;
  private readonly writer: Writer;

  constructor({ unary, cache }: ClientParams) {
    const relationships = cache.createTable<string, Relationship>({
      name: "relationships",
      equal: (a, b) =>
        idsEqual(a.from, b.from) && idsEqual(a.to, b.to) && a.type === b.type,
      listen: [
        query.createSetListener(RELATIONSHIP_SET_CHANNEL_NAME, relationshipZ, {
          key: (changed) => relationshipToString(changed),
        }),
        query.createDeleteListener(RELATIONSHIP_DELETE_CHANNEL_NAME, relationshipZ, {
          key: (changed) => relationshipToString(changed),
        }),
      ],
    });
    const resources = cache.createTable<string, Resource>({
      name: "resources",
      equal: (a, b) => deep.equal(a, b),
      // Fetches bypass the query cache: sibling domain deletes never write
      // through to the resource store, so cached entries may be stale.
      fetch: async (keys) =>
        await this.execRetrieve({ ids: parseIDs(keys), ignoreNotFoundError: true }),
      listen: [
        query.createSetListener(RESOURCE_SET_CHANNEL_NAME, resourceZ, {
          value: (changed, prev) => (prev == null ? changed : { ...prev, ...changed }),
        }),
        // The store is keyed by the full "type:key" string, not the bare key.
        query.createDeleteListener(RESOURCE_DELETE_CHANNEL_NAME, idZ, {
          key: (changed) => idToString(changed),
        }),
      ],
    });
    const single = cache.queries<string, Resource, string, Resource>({
      name: "resource",
      table: resources,
      fetch: async (q) => [(await this.fetchSingle(q)).key],
      compose: (records) => records[0],
      keyOf: (q) => q,
      single: true,
    });
    super(cache, {
      name: "resource",
      table: resources,
      request: {
        schema: retrieveRequestZ,
        fetch: async (req) => await this.fetchRequest(req),
        matches: (resource, req) => requestFilter(req)(resource),
      },
      single: {
        is: (params): params is ID =>
          typeof params === "object" && params !== null && "key" in params,
        normalize: (id) => idToString(id),
        space: single as query.Retrieves<query.Params, Resource>,
      },
    });
    this.unary = unary;
    this.writer = new Writer(unary);
    this.stores = { relationships, resources };
    this.children = this.dependentSurface(cache, "children", "to");
    this.parents = this.dependentSurface(cache, "parents", "from");
  }

  /** Read surface of the relationship cache. */
  get relationships(): query.Table<string, Relationship> {
    return this.stores.relationships;
  }

  /** Read surface of the resource cache. */
  get resources(): query.Table<string, Resource> {
    return this.stores.resources;
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
    const rollback = new destructor.Chain();
    rollback.add(...move());
    await rollback.guard(
      async () => await this.writer.moveChildren(from, to, ...children),
    );
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

  /** Merges a fetched resource with cached field data it may lack. */
  private mergeFieldData(r: Resource): Resource {
    const p = this.resources.get(r.key);
    return p == null ? r : { ...r, data: r.data ?? p.data };
  }

  private writeResources(resources: Resource[]): void {
    this.resources.set(resources.map((r) => this.mergeFieldData(r)));
  }

  // Resources are deleted through other domain clients whose write-through
  // never reaches the resource store, so fetches always hit the network
  // rather than serving a possibly stale cached entry.
  private async fetchSingle(q: string): Promise<Resource> {
    const resources = await this.execRetrieve({ ids: [idZ.parse(q)] });
    if (resources.length === 0)
      throw new NotFoundError(`No resource found with ID ${q}`);
    this.writeResources(resources);
    return resources[0];
  }

  private async fetchRequest(req: NormalizedRequest): Promise<Resource[]> {
    const { ids, ...rest } = req;
    const resources = await this.execRetrieve(
      ids == null ? rest : { ids: parseIDs(ids), ...rest },
    );
    return resources.map((r) => this.mergeFieldData(r));
  }

  private async fetchDependents(
    q: DependentRequest,
    direction: RelationshipDirection,
  ): Promise<Resource[]> {
    const { ids, ...rest } = q;
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

  private dependentSurface(
    cache: query.Cache,
    name: string,
    direction: RelationshipDirection,
  ): query.Retrieves<DependentParams, Resource[]> {
    const anchor = oppositeRelationshipDirection(direction);
    const space = cache.queries<DependentRequest, Resource[], string, Resource>({
      name,
      table: this.resources,
      fetch: async (q) => (await this.fetchDependents(q, direction)).map((r) => r.key),
      compose: (records) => records,
      matches: (resource, q) => this.isDependent(resource, q, direction),
      serverFields: DEPENDENT_SERVER_FIELDS,
      watch: [
        query.watch(this.relationships, (event, q: DependentRequest) => {
          const rel =
            event.variant === "set" ? event.value : relationshipZ.parse(event.key);
          if (rel.type !== PARENT_OF_RELATIONSHIP_TYPE) return null;
          if (!q.ids.includes(idToString(rel[anchor]))) return null;
          if (q.types != null && !q.types.includes(rel[direction].type)) return null;
          return [idToString(rel[direction])];
        }),
      ],
    });
    return {
      retrieve: async ({ ids, ...options }, opts) =>
        await space.retrieve(normalizeDependents(ids, options), opts),
      onChange: ({ ids, ...options }, handler) =>
        space.onChange(normalizeDependents(ids, options), handler),
      getCached: ({ ids, ...options }) =>
        space.getCached(normalizeDependents(ids, options)),
    };
  }

  /** Whether a cached relationship makes the resource a dependent of the query. */
  private isDependent(
    resource: Resource,
    q: DependentRequest,
    direction: RelationshipDirection,
  ): boolean {
    if (q.types != null && !q.types.includes(resource.id.type)) return false;
    const anchor = oppositeRelationshipDirection(direction);
    return (
      this.relationships.get(
        (rel) =>
          rel.type === PARENT_OF_RELATIONSHIP_TYPE &&
          q.ids.includes(idToString(rel[anchor])) &&
          idsEqual(rel[direction], resource.id),
      ).length > 0
    );
  }

  private async execRetrieve(request: RetrieveRequest): Promise<Resource[]> {
    const { resources } = await this.unary.send(
      "/ontology/retrieve",
      request,
      wireReqZ,
      retrieveResZ,
    );
    return resources;
  }
}
