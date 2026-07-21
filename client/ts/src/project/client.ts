// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, caseconv, primitive, record } from "@synnaxlabs/x";
import { z } from "zod";

import { cache } from "@/cache";
import { ontology } from "@/ontology";
import {
  type Key,
  keyZ,
  type New,
  ontologyID,
  type Project,
  projectZ,
} from "@/project/types.gen";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export const SET_CHANNEL_NAME = "sy_project_set";
export const DELETE_CHANNEL_NAME = "sy_project_delete";

const retrieveReqZ = z.object({
  keys: keyZ.array().optional(),
  searchTerm: z.string().optional(),
  offset: z.int().optional(),
  limit: z.int().optional(),
});
export interface RetrieveRequest extends z.infer<typeof retrieveReqZ> {}
const createReqZ = z.object({ projects: projectZ.array() });
const renameReqZ = z.object({ key: keyZ, name: z.string() });
const setLayoutReqZ = z.object({
  key: keyZ,
  layout: caseconv.preserveCase(record.unknownZ()),
});
const deleteReqZ = z.object({ keys: keyZ.array() });

const retrieveResZ = z.object({ projects: projectZ.array().default(() => []) });
const createResZ = z.object({ projects: projectZ.array() });
const emptyResZ = z.object({});

export interface SetLayoutParams extends z.input<typeof setLayoutReqZ> {}

/** Query fields only the server can evaluate. */
const SERVER_FIELDS = ["searchTerm", "limit", "offset"] as const;

const isKeysOnly = (req: RetrieveRequest): req is RetrieveRequest & { keys: Key[] } =>
  primitive.isNonZero(req.keys) &&
  req.searchTerm == null &&
  req.limit == null &&
  req.offset == null;

/**
 * Client-side matching for a request: key sets. Server-computed shapes
 * (search, limit/offset) never reach this filter; they refetch instead.
 */
const requestFilter = (req: RetrieveRequest): ((p: Project) => boolean) => {
  const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
  return (p) => keySet == null || keySet.has(p.key);
};

const toRequest = (params: Key[] | RetrieveRequest): RetrieveRequest =>
  retrieveReqZ.parse(Array.isArray(params) ? { keys: params } : params);

const createTable = (engine: cache.Cache): cache.Table<Key, Project> => {
  const table = engine.createTable<Key, Project>({ name: "projects" });
  const set: cache.ChannelListener<typeof projectZ> = {
    channel: SET_CHANNEL_NAME,
    schema: projectZ,
    onChange: (changed) => table.set(changed),
  };
  const del: cache.ChannelListener<typeof keyZ> = {
    channel: DELETE_CHANNEL_NAME,
    schema: keyZ,
    onChange: (changed) => table.delete(changed),
  };
  engine.addListeners(table, set, del);
  return table;
};

export class Client extends cache.Reader<
  Key,
  Key[] | RetrieveRequest,
  Key,
  RetrieveRequest,
  Project
> {
  private readonly client: UnaryClient;
  private readonly store: cache.Table<Key, Project>;
  private readonly ontology: ontology.Stores;

  constructor(
    client: UnaryClient,
    engine: cache.Cache,
    ontologyStores: ontology.Stores,
  ) {
    const store = createTable(engine);
    super({
      single: engine.answers({
        name: "project",
        table: store,
        fetch: async (query) => [await this.fetchSingle(query)].map((p) => p.key),
        compose: (records) => records[0],
        keyOf: (query) => query,
        single: true,
      }),
      request: engine.answers({
        name: "projects",
        table: store,
        fetch: async (query) => (await this.fetchRequest(query)).map((p) => p.key),
        compose: (records) => records,
        matches: (project, query) => requestFilter(query)(project),
        serverFields: SERVER_FIELDS,
      }),
      isSingle: (params) => typeof params === "string",
      normalizeSingle: (key) => key,
      normalizeRequest: toRequest,
    });
    this.client = client;
    this.ontology = ontologyStores;
    this.store = store;
  }

  async create(project: New): Promise<Project>;
  async create(projects: New[]): Promise<Project[]>;
  async create(projects: New | New[]): Promise<Project | Project[]> {
    const isMany = Array.isArray(projects);
    const res = await this.client.send(
      "/project/create",
      { projects: array.toArray(projects) },
      createReqZ,
      createResZ,
    );
    this.store.set(res.projects);
    return isMany ? res.projects : res.projects[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    await this.client.send("/project/rename", { key, name }, renameReqZ, emptyResZ);
    this.mergeThrough(key, { name });
    ontology.renameCachedResource(this.ontology, ontologyID(key), name);
  }

  async setLayout(
    key: Key,
    layout: record.Unknown,
    opts: cache.WriteOptions = {},
  ): Promise<void> {
    const rollback = new cache.Rollback();
    rollback.add(cache.partialUpdate(this.store, key, { layout }));
    await opts.onOptimistic?.();
    await rollback.guard(
      async () =>
        await this.client.send(
          "/project/set-layout",
          { key, layout },
          setLayoutReqZ,
          emptyResZ,
        ),
    );
    this.mergeThrough(key, { layout });
  }

  async delete(key: Key, opts?: cache.WriteOptions): Promise<void>;
  async delete(keys: Key[], opts?: cache.WriteOptions): Promise<void>;
  async delete(keys: Key | Key[], opts: cache.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const rollback = new cache.Rollback();
    rollback.add(ontology.deleteCachedResources(this.ontology, ontologyID(keysArr)));
    rollback.add(this.store.delete(keysArr));
    await opts.onOptimistic?.();
    await rollback.guard(
      async () =>
        await this.client.send(
          "/project/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    );
    this.store.delete(keysArr);
  }

  // Undefined fields are dropped: the server keeps prior values for them.
  private mergeThrough(key: Key, changes: Partial<Project>): void {
    const prev = this.store.get(key);
    if (prev != null)
      this.store.set(key, { ...prev, ...record.purgeUndefined(changes) });
  }

  private async execRetrieve(req: RetrieveRequest): Promise<Project[]> {
    const res = await this.client.send(
      "/project/retrieve",
      req,
      retrieveReqZ,
      retrieveResZ,
    );
    return res.projects;
  }

  /**
   * Fetches the given keys, serving cached entries and fetching only the
   * misses. Preserves the caller's key order.
   */
  private async fetchKeys(keys: Key[]): Promise<Project[]> {
    const results: Project[] = [];
    const misses: Key[] = [];
    for (const key of keys) {
      const cached = this.store.get(key);
      if (cached != null) results.push(cached);
      else misses.push(key);
    }
    if (misses.length > 0) {
      const fetched = await this.execRetrieve({ keys: misses });
      this.store.set(fetched);
      results.push(...fetched);
    }
    return cache.orderByKeys(keys, results, (p) => p.key);
  }

  private async fetchSingle(query: Key): Promise<Project> {
    const cached = this.store.get(query);
    if (cached != null) return cached;
    const projects = await this.execRetrieve({ keys: [query] });
    checkForMultipleOrNoResults("Project", query, projects, true);
    this.store.set(projects);
    return projects[0];
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Project[]> {
    if (isKeysOnly(query)) return await this.fetchKeys(query.keys);
    const projects = await this.execRetrieve(query);
    this.store.set(projects);
    return projects;
  }
}
