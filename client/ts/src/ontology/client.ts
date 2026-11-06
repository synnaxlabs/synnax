// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { deep, type destructor, primitive } from "@synnaxlabs/x";
import { z } from "zod";

import { NotFoundError } from "@/errors";
import {
  oppositeRelationshipDirection,
  PARENT_OF_RELATIONSHIP_TYPE,
  parseIDs,
  type Relationship,
  type RelationshipDirection,
  relationshipToString,
  relationshipZ,
  type Resource,
  resourceZ,
  ROOT_ID,
} from "@/ontology/payload";
import {
  Cache,
  RELATIONSHIP_DELETE_CHANNEL_NAME,
  RELATIONSHIP_SET_CHANNEL_NAME,
  type RelationshipIndexes,
  RESOURCE_DELETE_CHANNEL_NAME,
  RESOURCE_SET_CHANNEL_NAME,
} from "@/ontology/store";
import { Writer } from "@/ontology/writer";
import { query } from "@/query";
import { resource } from "@/resource";
import {
  type ID,
  idsEqual,
  idToString,
  idZ,
  parseID,
  stringIDZ,
} from "@/resource/payload";

const wireReqZ = z.object({
  ids: idZ.array().optional(),
  children: z.boolean().optional(),
  parents: z.boolean().optional(),
  excludeFieldData: z.boolean().optional(),
  types: resource.typeZ.array().optional(),
  searchTerm: z.string().optional(),
  limit: z.int().optional(),
  offset: z.int().optional(),
  ignoreNotFoundError: z.boolean().optional(),
  root: idZ.optional(),
  ancestors: z.boolean().optional(),
});
/** Everything an ontology retrieval can filter on. */
export interface RetrieveRequest extends z.infer<typeof wireReqZ> {}

/** The filters a retrieval accepts alongside its IDs. */
export interface RetrieveOptions extends Pick<
  RetrieveRequest,
  "excludeFieldData" | "types"
> {}

const retrieveResZ = z.object({
  resources: resourceZ.array(),
  ancestors: resourceZ.array().optional(),
  relationships: relationshipZ.array().optional(),
});

const parentRel = (from: ID, to: ID): Relationship => ({
  from,
  type: PARENT_OF_RELATIONSHIP_TYPE,
  to,
});

/** A retrieve request with IDs normalized to stable, sorted strings. */
type NormalizedRequest = {
  ids?: string[];
  excludeFieldData?: boolean;
  types?: resource.Type[];
  searchTerm?: string;
  root?: string;
  limit?: number;
  offset?: number;
};

/** Request key for children/parents listings of a set of resources. */
type DependentRequest = {
  ids: string[];
  types?: resource.Type[];
  searchTerm?: string;
  offset?: number;
  limit?: number;
  excludeFieldData?: boolean;
};

/** Query addressing the children or parents of a set of resources. */
export type DependentParams = {
  ids: ID | ID[];
  types?: resource.Type[];
  searchTerm?: string;
  offset?: number;
  limit?: number;
  excludeFieldData?: boolean;
};

const DEPENDENT_SERVER_FIELDS = ["searchTerm", "limit", "offset"] as const;

/** Query addressing the ancestors of a set of resources. */
export type AncestorParams = {
  ids: ID | ID[];
  /** Where the walk up ends. Defaults to the ontology root. */
  root?: ID;
};

/** Request key for the ancestors of a set of resources. */
type AncestorRequest = { ids: string[]; root?: string };

/** The ancestors of a set of resources. */
export interface Ancestry {
  /** Every resource between the IDs and the root, leaving out the root. */
  resources: Resource[];
  /**
   * The paths from the root down to the parent of each ID, keyed by the string form of
   * the ID. A path lists IDs from the top down and leaves out the root.
   */
  paths: Record<string, ID[][]>;
}

const normalizeAncestors = ({ ids, root }: AncestorParams): AncestorRequest => {
  const out: AncestorRequest = { ids: normalizeIDs(ids) };
  if (root != null) out.root = idToString(root);
  return out;
};

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
    types: resource.typeZ.array().optional(),
    searchTerm: z.string().optional(),
    root: idZ.optional(),
    limit: z.int().optional(),
    offset: z.int().optional(),
  })
  .transform((req): NormalizedRequest => {
    const out: NormalizedRequest = {};
    if (req.ids != null) out.ids = normalizeIDs(req.ids);
    if (req.excludeFieldData != null) out.excludeFieldData = req.excludeFieldData;
    if (req.types != null) out.types = [...req.types].sort();
    if (req.searchTerm != null) out.searchTerm = req.searchTerm;
    if (req.root != null) out.root = idToString(req.root);
    if (req.limit != null) out.limit = req.limit;
    if (req.offset != null) out.offset = req.offset;
    return out;
  });

/** Resources are addressed by `ids`, so a bare key list is that shorthand. */
const retrieveMultiParamsZ = retrieveRequestZ.or(
  idZ.array().transform((ids): NormalizedRequest => ({ ids: normalizeIDs(ids) })),
);

/** Params for an ontology retrieval. A bare ID array is shorthand for `{ ids }`. */
export type RetrieveParams = z.input<typeof retrieveRequestZ>;

const singleParamsZ = z
  .union([z.strictObject({ type: resource.typeZ, key: z.string() }), stringIDZ])
  .transform((id) => idToString(id));

/**
 * Client-side matching for a request: exact for id and type sets. Server-computed
 * shapes (search, limit/offset) never reach this filter; they refetch instead.
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

/** Config for {@link Client}. */
export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
}

/**
 * Reads the ontology on a Core: the resources and the relationships that join them.
 * Reach it through `client.ontology`. Reads are served from a cache that a change
 * stream keeps current.
 */
export class Client extends query.Retriever<
  typeof retrieveMultiParamsZ,
  string,
  Resource,
  Resource,
  z.input<typeof singleParamsZ>
> {
  readonly type: string = "ontology";
  /** Read surface of the ontology record tables. */
  readonly cache: Cache;
  /** Cached read surface for the children of a set of resources. */
  readonly children: query.Retrieves<DependentParams, Resource[]>;
  /** Cached read surface for the parents of a set of resources. */
  readonly parents: query.Retrieves<DependentParams, Resource[]>;
  /**
   * Cached read surface for the ancestors of a set of resources. Paths follow moves
   * and group renames while subscribed. A retrieve also caches the resources of the
   * IDs.
   */
  readonly ancestors: query.Retrieves<AncestorParams, Ancestry>;
  private readonly cfg: ClientConfig;
  private readonly writer: Writer;

  constructor(cfg: ClientConfig) {
    const { unary, cache } = cfg;
    const relationshipIndexes: RelationshipIndexes = {
      byTo: new query.LookupIndex<string, Relationship>((r) => idToString(r.to)),
      byFrom: new query.LookupIndex<string, Relationship>((r) => idToString(r.from)),
    };
    const relationships = cache.createTable<string, Relationship>({
      name: "relationships",
      equal: (a, b) =>
        idsEqual(a.from, b.from) && idsEqual(a.to, b.to) && a.type === b.type,
      indexes: [relationshipIndexes.byTo, relationshipIndexes.byFrom],
      // A relationship deleted and created again keeps its key.
      revivable: true,
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
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.fetchRequest(req),
        matches: (resource, req) => requestFilter(req)(resource),
        // A move changes which resources sit under the root of a search, but it sends
        // no resource change.
        watch: [
          query.watch(relationships, (event, req: NormalizedRequest) => {
            if (req.root == null || req.searchTerm == null) return null;
            const rel =
              event.variant === "set" ? event.value : relationshipZ.parse(event.key);
            return rel.type === PARENT_OF_RELATIONSHIP_TYPE ? "refetch" : null;
          }),
        ],
      },
      single: { schema: singleParamsZ, space: single },
    });
    this.cfg = cfg;
    this.writer = new Writer(unary);
    this.cache = new Cache(relationships, resources, relationshipIndexes);
    this.children = this.dependentSurface(cache, "children", "to");
    this.parents = this.dependentSurface(cache, "parents", "from");
    this.ancestors = this.ancestorSurface(cache);
  }

  /**
   * Reads the resources the params address. The ID-array form remains for migration
   * only: query with `{ ids }`, `children.retrieve`, or `parents.retrieve` instead.
   */
  retrieve(
    ids: Array<ID | string>,
    options?: RetrieveOptions & { children?: boolean; parents?: boolean },
  ): Promise<Resource[]>;
  retrieve(
    params: string | z.input<typeof singleParamsZ>,
    options?: query.FetchOptions,
  ): Promise<Resource>;
  retrieve(params: RetrieveParams, options?: query.FetchOptions): Promise<Resource[]>;
  async retrieve(
    params:
      Array<ID | string> | string | z.input<typeof singleParamsZ> | RetrieveParams,
    options?:
      | query.FetchOptions
      | (RetrieveOptions & { children?: boolean; parents?: boolean }),
  ): Promise<Resource | Resource[]> {
    if (Array.isArray(params)) {
      const ids = parseIDs(params);
      const {
        children = false,
        parents = false,
        ...rest
      } = (options ?? {}) as RetrieveOptions & {
        children?: boolean;
        parents?: boolean;
      };
      if (children) return await this.children.retrieve({ ids, ...rest });
      if (parents) return await this.parents.retrieve({ ids, ...rest });
      return await super.retrieve({ ids, ...rest });
    }
    return await super.retrieve(
      params as z.input<typeof singleParamsZ>,
      options as query.FetchOptions,
    );
  }

  /** Migration-only alias for `children.retrieve({ ids })`. */
  async retrieveChildren(
    ids: ID | ID[],
    options?: RetrieveOptions,
  ): Promise<Resource[]> {
    return await this.children.retrieve({ ids, ...options });
  }

  /** Migration-only alias for `parents.retrieve({ ids })`. */
  async retrieveParents(
    ids: ID | ID[],
    options?: RetrieveOptions,
  ): Promise<Resource[]> {
    return await this.parents.retrieve({ ids, ...options });
  }

  /** Adds children to a resource in the ontology. */
  async addChildren(id: ID, ...children: ID[]): Promise<void> {
    await this.writer.addChildren(id, ...children);
    children.forEach((child) => {
      const rel = parentRel(id, child);
      this.cache.relationships.set(relationshipToString(rel), rel);
    });
  }

  /** Removes children from a resource in the ontology. */
  async removeChildren(id: ID, ...children: ID[]): Promise<void> {
    await this.writer.removeChildren(id, ...children);
    children.forEach((child) =>
      this.cache.relationships.delete(relationshipToString(parentRel(id, child))),
    );
  }

  /** Moves children from one resource to another in the ontology. */
  async moveChildren(from: ID, to: ID, ...children: ID[]): Promise<void> {
    const move = (): destructor.Destructor[] => {
      const deletions = children.map((child) =>
        this.cache.relationships.delete(relationshipToString(parentRel(from, child))),
      );
      const insertions = children.map((child) => {
        const rel = parentRel(to, child);
        return this.cache.relationships.set(relationshipToString(rel), rel);
      });
      return [...deletions, ...insertions];
    };
    await query.optimistic({
      rollbacks: move(),
      commit: async () => await this.writer.moveChildren(from, to, ...children),
    });
    move();
  }

  /** Subscribes to every resource set delivered to the cache. */
  onResourceSet(handler: (resource: Resource) => void): destructor.Destructor {
    return this.cache.resources.subscribe((event) => {
      if (event.variant === "set") handler(event.value);
    });
  }

  /** Subscribes to every resource delete delivered to the cache. */
  onResourceDelete(handler: (id: ID) => void): destructor.Destructor {
    return this.cache.resources.subscribe((event) => {
      if (event.variant === "delete") handler(parseID(event.key));
    });
  }

  /** Subscribes to every relationship set delivered to the cache. */
  onRelationshipSet(handler: (rel: Relationship) => void): destructor.Destructor {
    return this.cache.relationships.subscribe((event) => {
      if (event.variant === "set") handler(event.value);
    });
  }

  /** Subscribes to every relationship delete delivered to the cache. */
  onRelationshipDelete(handler: (rel: Relationship) => void): destructor.Destructor {
    return this.cache.relationships.subscribe((event) => {
      if (event.variant === "delete") handler(relationshipZ.parse(event.key));
    });
  }

  /** Merges a fetched resource with cached field data it may lack. */
  private mergeFieldData(r: Resource): Resource {
    const p = this.cache.resources.get(r.key);
    return p == null ? r : { ...r, data: r.data ?? p.data };
  }

  private writeResources(resources: Resource[]): void {
    this.cache.resources.set(resources.map((r) => this.mergeFieldData(r)));
  }

  // Resources are deleted through other domain clients whose write-through
  // never reaches the resource store, so fetches always hit the network
  // rather than serving a possibly stale cached entry.
  private async fetchSingle(q: string): Promise<Resource> {
    const resources = await this.execRetrieve({
      ids: [parseID(q)],
    });
    if (resources.length === 0)
      throw new NotFoundError(`No resource found with ID ${q}`);
    this.writeResources(resources);
    return resources[0];
  }

  private async fetchRequest(req: NormalizedRequest): Promise<Resource[]> {
    const { ids, root, ...rest } = req;
    const resources = await this.execRetrieve({
      ...rest,
      ...(ids != null && { ids: parseIDs(ids) }),
      ...(root != null && { root: parseID(root) }),
    });
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
      const anchor = parseID(ids[0]);
      resources.forEach(({ id }) => {
        const rel = direction === "to" ? parentRel(anchor, id) : parentRel(id, anchor);
        this.cache.relationships.set(relationshipToString(rel), rel);
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
      table: this.cache.resources,
      fetch: async (q) => (await this.fetchDependents(q, direction)).map((r) => r.key),
      compose: (records) => records,
      matches: (resource, q) => this.isDependent(resource, q, direction),
      serverFields: DEPENDENT_SERVER_FIELDS,
      watch: [
        query.watch(this.cache.relationships, (event, q: DependentRequest) => {
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

  private async fetchAncestors({ ids, root }: AncestorRequest): Promise<string[]> {
    const table = this.cache.relationships;
    const relationshipsStamp = table.stamp();
    const resourcesStamp = this.cache.resources.stamp();
    // A link written while the request is in flight is newer than the answer.
    const written = new Set<string>();
    const stopWatching = table.subscribe(({ key }) => written.add(key));
    let res: z.infer<typeof retrieveResZ>;
    try {
      res = await this.send({
        ids: parseIDs(ids),
        root: root == null ? undefined : parseID(root),
        ancestors: true,
        ignoreNotFoundError: true,
      });
    } finally {
      stopWatching();
    }
    const { resources, ancestors = [], relationships = [] } = res;
    // The resources of the IDs come back too, so a reader can render them with no
    // second request.
    this.cache.resources.ingest(
      [...resources, ...ancestors].map((r) => this.mergeFieldData(r)),
      { mode: "set", since: resourcesStamp },
    );
    // The answer holds every parent link of each resource the walk passed, so a cached
    // link it lacks was deleted while the cache missed the change.
    const passed = [...resources.map((r) => r.id), ...relationships.map((r) => r.from)];
    const walked = new Map(passed.map((id) => [idToString(id), id]));
    walked.delete(root ?? idToString(ROOT_ID));
    const fetched = new Set(relationships.map(relationshipToString));
    const stale = [...walked.values()]
      .flatMap((id) => this.cache.relationshipsTo(id))
      .filter(({ type }) => type === PARENT_OF_RELATIONSHIP_TYPE)
      .map(relationshipToString)
      .filter((key) => !fetched.has(key) && !written.has(key));
    table.ingestDeletes(stale, relationshipsStamp);
    // The Core walks up from the IDs, so the reversed links reach the cache top down
    // and each one finds the path above it already cached.
    relationships
      .toReversed()
      .map((rel) => ({ rel, key: relationshipToString(rel) }))
      .filter(({ key }) => !written.has(key))
      .forEach(({ rel, key }) => table.set(key, rel));
    return [...new Set([...resources, ...ancestors].map((r) => r.key))];
  }

  private ancestorSurface(
    cache: query.Cache,
  ): query.Retrieves<AncestorParams, Ancestry> {
    const anchorsOf = (q: AncestorRequest): ID[] => parseIDs(q.ids);
    const rootOf = (q: AncestorRequest): ID | undefined =>
      q.root == null ? undefined : parseID(q.root);
    // The IDs are members but stay out of the resources, so a link change that moves no
    // ancestor still recomposes the paths.
    const space = cache.queries<AncestorRequest, Ancestry, string, Resource>({
      name: "ancestors",
      table: this.cache.resources,
      fetch: async (q) => await this.fetchAncestors(q),
      compose: (members, q) => {
        const ancestors = this.cache.ancestorKeys(anchorsOf(q), rootOf(q));
        return {
          resources: members.filter(({ key }) => ancestors.has(key)),
          paths: this.cache.paths(anchorsOf(q), rootOf(q)),
        };
      },
      matches: (resource, q) =>
        q.ids.includes(resource.key) ||
        this.cache.ancestorKeys(anchorsOf(q), rootOf(q)).has(resource.key),
      watch: [
        query.watch(this.cache.relationships, (event, q: AncestorRequest) => {
          const rel =
            event.variant === "set" ? event.value : relationshipZ.parse(event.key);
          if (rel.type !== PARENT_OF_RELATIONSHIP_TYPE) return null;
          const to = idToString(rel.to);
          const tracked = this.cache.ancestorKeys(anchorsOf(q), rootOf(q));
          if (!q.ids.includes(to) && !tracked.has(to)) return null;
          const from = idToString(rel.from);
          const above = [to, from, ...this.cache.ancestorKeys([rel.from], rootOf(q))];
          if (event.variant === "delete" || from === (q.root ?? idToString(ROOT_ID)))
            return above;
          // A new parent whose own path is not cached needs the Core to place it.
          const placed =
            this.cache.resources.get(from) != null &&
            this.cache
              .relationshipsTo(rel.from)
              .some((r) => r.type === PARENT_OF_RELATIONSHIP_TYPE);
          return placed ? above : "refetch";
        }),
      ],
    });
    return {
      retrieve: async (params, opts) =>
        await space.retrieve(normalizeAncestors(params), opts),
      onChange: (params, handler) =>
        space.onChange(normalizeAncestors(params), handler),
      getCached: (params) => space.getCached(normalizeAncestors(params)),
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
    const rels =
      direction === "to"
        ? this.cache.relationshipsTo(resource.id)
        : this.cache.relationshipsFrom(resource.id);
    return rels.some(
      (rel) =>
        rel.type === PARENT_OF_RELATIONSHIP_TYPE &&
        q.ids.includes(idToString(rel[anchor])),
    );
  }

  private async execRetrieve(request: RetrieveRequest): Promise<Resource[]> {
    return (await this.send(request)).resources;
  }

  private async send(request: RetrieveRequest): Promise<z.infer<typeof retrieveResZ>> {
    return await this.cfg.unary.send(
      "/ontology/retrieve",
      request,
      wireReqZ,
      retrieveResZ,
    );
  }
}
