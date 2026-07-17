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

const MOUNT_SCOPE = "project.mounts";

const isKeysOnly = (req: RetrieveRequest): boolean =>
  primitive.isNonZero(req.keys) &&
  req.searchTerm == null &&
  req.limit == null &&
  req.offset == null;

/**
 * Client-side approximation of the server's matching for a request: exact for
 * key sets, permissive for server-computed shapes (search), which accept
 * every change and drift toward the server's answer.
 */
const requestFilter = (req: RetrieveRequest): ((p: Project) => boolean) => {
  const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
  return (p) => keySet == null || keySet.has(p.key);
};

const toRequest = (params: Key[] | RetrieveRequest): RetrieveRequest =>
  retrieveReqZ.parse(Array.isArray(params) ? { keys: params } : params);

export class Client {
  private readonly client: UnaryClient;
  private readonly engine_?: cache.Engine;
  private readonly queries_?: {
    single: cache.Queries<Key, Project>;
    request: cache.Queries<RetrieveRequest, Project[]>;
  };

  constructor(client: UnaryClient, engine?: cache.Engine) {
    this.client = client;
    if (engine == null) return;
    bindStore(engine);
    this.engine_ = engine;
    const ensureStreaming = async () => await engine.ensureStreaming();
    this.queries_ = {
      single: new cache.Queries({
        name: "project",
        fetch: async (query) => await this.fetchSingle(query),
        mount: (params) => this.mountSingle(params),
        ensureStreaming,
      }),
      request: new cache.Queries({
        name: "projects",
        fetch: async (query) => await this.fetchRequest(query),
        mount: (params) => this.mountRequest(params),
        ensureStreaming,
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
    this.writes?.set(res.projects);
    return isMany ? res.projects : res.projects[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    await this.client.send("/project/rename", { key, name }, renameReqZ, emptyResZ);
    this.mergeThrough(key, { name });
    if (this.engine_ != null)
      ontology.renameCachedResource(this.engine_, ontologyID(key), name);
  }

  async setLayout(
    key: Key,
    layout: record.Unknown,
    opts: cache.WriteOptions = {},
  ): Promise<void> {
    const rollback = new cache.Rollback();
    const writes = this.writes;
    if (writes != null) rollback.add(cache.partialUpdate(writes, key, { layout }));
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
    if (this.queries_ == null) {
      const req =
        isSingle || Array.isArray(keys) ? { keys: array.toArray(keys) } : keys;
      const projects = await this.execRetrieve(req);
      return isSingle ? projects[0] : projects;
    }
    if (isSingle) return await this.queries_.single.retrieve(keys);
    return await this.queries_.request.retrieve(toRequest(keys));
  }

  /**
   * Subscribes to changes in the cached answer to the given query. Single
   * queries deliver a project; every other shape delivers the matching
   * projects.
   * @throws when the cache was disabled at client construction.
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
    const queries = this.requireQueries();
    if (typeof keys === "string")
      return queries.single.onChange(keys, handler as cache.ChangeHandler<Project>);
    return queries.request.onChange(
      toRequest(keys),
      handler as cache.ChangeHandler<Project[]>,
    );
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached.
   * @throws when the cache was disabled at client construction.
   */
  getCached(key: Key): cache.Cached<Project> | undefined;
  getCached(keys: Key[]): cache.Cached<Project[]> | undefined;
  getCached(req: RetrieveRequest): cache.Cached<Project[]> | undefined;
  getCached(
    keys: Key | Key[] | RetrieveRequest,
  ): cache.Cached<Project> | cache.Cached<Project[]> | undefined {
    const queries = this.requireQueries();
    if (typeof keys === "string") return queries.single.getCached(keys);
    return queries.request.getCached(toRequest(keys));
  }

  async delete(key: Key, opts?: cache.WriteOptions): Promise<void>;
  async delete(keys: Key[], opts?: cache.WriteOptions): Promise<void>;
  async delete(keys: Key | Key[], opts: cache.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const rollback = new cache.Rollback();
    const writes = this.writes;
    if (this.engine_ != null && writes != null) {
      rollback.add(ontology.deleteCachedResources(this.engine_, ontologyID(keysArr)));
      rollback.add(writes.delete(keysArr));
    }
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
    this.writes?.delete(keysArr);
  }

  private get writes(): cache.UnaryStore<Key, Project> | undefined {
    return this.engine_?.store(STORE_KEY);
  }

  private get projectStore(): cache.UnaryStore<Key, Project> {
    return this.requireEngine().store(STORE_KEY);
  }

  // Query mounts subscribe in their own scope: stores suppress notifications
  // to listeners in the writer's scope, and the streamer writes in the default
  // scope, which would silence default-scope subscriptions entirely.
  private get projectEvents(): cache.UnaryStore<Key, Project> {
    return this.requireEngine().store(STORE_KEY, MOUNT_SCOPE);
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

  // Undefined fields are dropped: the server keeps prior values for them.
  private mergeThrough(key: Key, changes: Partial<Project>): void {
    const store = this.writes;
    if (store == null) return;
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
      this.projectStore.set(fetched);
      results.push(...fetched);
    }
    return cache.orderByKeys(keys, results, (p) => p.key);
  }

  private async fetchSingle(query: Key): Promise<Project> {
    const cached = this.projectStore.get(query);
    if (cached != null) return cached;
    const projects = await this.execRetrieve({ keys: [query] });
    checkForMultipleOrNoResults("Project", query, projects, true);
    this.projectStore.set(projects);
    return projects[0];
  }

  private mountSingle({ query, update, remove }: cache.MountParams<Key, Project>) {
    return [
      this.projectEvents.onSet((project) => {
        if (project.key === query) update(project);
      }),
      this.projectEvents.onDelete((key) => {
        if (key === query) remove(this.projectStore.getTombstone(key)?.corpse);
      }),
    ];
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Project[]> {
    if (isKeysOnly(query)) return await this.fetchKeys(query.keys as Key[]);
    const projects = await this.execRetrieve(query);
    this.projectStore.set(projects);
    return projects;
  }

  private mountRequest({
    query,
    update,
  }: cache.MountParams<RetrieveRequest, Project[]>) {
    const matches = requestFilter(query);
    return [
      this.projectEvents.onSet((project) => {
        update((prev) => {
          if (prev == null) return prev;
          const existing = prev.some((p) => p.key === project.key);
          if (!matches(project))
            return existing ? prev.filter((p) => p.key !== project.key) : prev;
          if (existing) return prev.map((p) => (p.key === project.key ? project : p));
          return [...prev, project];
        });
      }),
      this.projectEvents.onDelete((key) => {
        update((prev) => prev?.filter((p) => p.key !== key));
      }),
    ];
  }
}
