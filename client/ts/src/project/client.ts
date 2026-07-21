// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, caseconv, type destructor, primitive, record } from "@synnaxlabs/x";
import { z } from "zod";

import { cache } from "@/cache";
import { ontology } from "@/ontology";
import { bindStore, STORE_KEY } from "@/project/store";
import {
  type Key,
  keyZ,
  type New,
  ontologyID,
  type Project,
  projectZ,
} from "@/project/types.gen";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

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

const isKeysOnly = (req: RetrieveRequest): boolean =>
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

export class Client {
  private readonly client: UnaryClient;
  private readonly cache: cache.Cache;
  private readonly answers: {
    single: cache.Answers<Key, Project, Key, Project>;
    request: cache.Answers<RetrieveRequest, Project[], Key, Project>;
  };

  constructor(client: UnaryClient, cache_: cache.Cache) {
    this.client = client;
    bindStore(cache_);
    this.cache = cache_;
    this.answers = {
      single: cache_.answers({
        name: "project",
        table: this.projectStore,
        fetch: async (query) => [await this.fetchSingle(query)].map((p) => p.key),
        compose: (records) => records[0],
        keyOf: (query) => query,
        single: true,
      }),
      request: cache_.answers({
        name: "projects",
        table: this.projectStore,
        fetch: async (query) => (await this.fetchRequest(query)).map((p) => p.key),
        compose: (records) => records,
        matches: (project, query) => requestFilter(query)(project),
        serverFields: SERVER_FIELDS,
      }),
    };
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
    this.writes.setMany(res.projects);
    return isMany ? res.projects : res.projects[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    await this.client.send("/project/rename", { key, name }, renameReqZ, emptyResZ);
    this.mergeThrough(key, { name });
    ontology.renameCachedResource(this.cache, ontologyID(key), name);
  }

  async setLayout(
    key: Key,
    layout: record.Unknown,
    opts: cache.WriteOptions = {},
  ): Promise<void> {
    const rollback = new cache.Rollback();
    rollback.add(cache.partialUpdate(this.writes, key, { layout }));
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

  async retrieve(key: Key): Promise<Project>;
  async retrieve(keys: Key[]): Promise<Project[]>;
  async retrieve(req: RetrieveRequest): Promise<Project[]>;
  async retrieve(keys: Key | Key[] | RetrieveRequest): Promise<Project | Project[]> {
    const isSingle = typeof keys === "string";
    if (isSingle) return await this.answers.single.retrieve(keys);
    return await this.answers.request.retrieve(toRequest(keys));
  }

  /**
   * Subscribes to changes in the cached answer to the given query. Single
   * queries deliver a project; every other shape delivers the matching
   * projects.
   */
  onChange(key: Key, handler: cache.ChangeHandler<Project>): destructor.Destructor;
  onChange(keys: Key[], handler: cache.ChangeHandler<Project[]>): destructor.Destructor;
  onChange(
    req: RetrieveRequest,
    handler: cache.ChangeHandler<Project[]>,
  ): destructor.Destructor;
  onChange(
    keys: Key | Key[] | RetrieveRequest,
    handler: cache.ChangeHandler<Project> | cache.ChangeHandler<Project[]>,
  ): destructor.Destructor {
    const answers = this.answers;
    if (typeof keys === "string")
      return answers.single.onChange(keys, handler as cache.ChangeHandler<Project>);
    return answers.request.onChange(
      toRequest(keys),
      handler as cache.ChangeHandler<Project[]>,
    );
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached.
   */
  getCached(key: Key): cache.Cached<Project> | undefined;
  getCached(keys: Key[]): cache.Cached<Project[]> | undefined;
  getCached(req: RetrieveRequest): cache.Cached<Project[]> | undefined;
  getCached(
    keys: Key | Key[] | RetrieveRequest,
  ): cache.Cached<Project> | cache.Cached<Project[]> | undefined {
    const answers = this.answers;
    if (typeof keys === "string") return answers.single.getCached(keys);
    return answers.request.getCached(toRequest(keys));
  }

  async delete(key: Key, opts?: cache.WriteOptions): Promise<void>;
  async delete(keys: Key[], opts?: cache.WriteOptions): Promise<void>;
  async delete(keys: Key | Key[], opts: cache.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const rollback = new cache.Rollback();
    const writes = this.writes;
    rollback.add(ontology.deleteCachedResources(this.cache, ontologyID(keysArr)));
    rollback.add(writes.delete(keysArr));
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
    this.writes.delete(keysArr);
  }

  private get writes(): cache.Table<Key, Project> {
    return this.cache.table(STORE_KEY);
  }

  private get projectStore(): cache.Table<Key, Project> {
    return this.cache.table(STORE_KEY);
  }

  // Undefined fields are dropped: the server keeps prior values for them.
  private mergeThrough(key: Key, changes: Partial<Project>): void {
    const store = this.writes;
    const prev = store.get(key);
    if (prev != null) store.set(key, { ...prev, ...record.purgeUndefined(changes) });
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
      const cached = this.projectStore.get(key);
      if (cached != null) results.push(cached);
      else misses.push(key);
    }
    if (misses.length > 0) {
      const fetched = await this.execRetrieve({ keys: misses });
      this.projectStore.setMany(fetched);
      results.push(...fetched);
    }
    return cache.orderByKeys(keys, results, (p) => p.key);
  }

  private async fetchSingle(query: Key): Promise<Project> {
    const cached = this.projectStore.get(query);
    if (cached != null) return cached;
    const projects = await this.execRetrieve({ keys: [query] });
    checkForMultipleOrNoResults("Project", query, projects, true);
    this.projectStore.setMany(projects);
    return projects[0];
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Project[]> {
    if (isKeysOnly(query)) return await this.fetchKeys(query.keys as Key[]);
    const projects = await this.execRetrieve(query);
    this.projectStore.setMany(projects);
    return projects;
  }
}
