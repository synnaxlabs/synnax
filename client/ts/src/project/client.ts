// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, caseconv, destructor, primitive, record } from "@synnaxlabs/x";
import { z } from "zod";

import { ontology } from "@/ontology";
import {
  type Key,
  keyZ,
  type New,
  ontologyID,
  type Project,
  projectZ,
} from "@/project/types.gen";
import { query } from "@/query";

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

/**
 * Client-side matching for a request: key sets. Server-computed shapes
 * (search, limit/offset) never reach this filter; they refetch instead.
 */
const requestFilter = (req: RetrieveRequest): ((p: Project) => boolean) => {
  const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
  return (p) => keySet == null || keySet.has(p.key);
};

export interface ClientParams {
  unary: UnaryClient;
  cache: query.Cache;
  ontologyStores: ontology.Stores;
}

export class Client extends query.Retriever<typeof retrieveReqZ, Key, Project> {
  private readonly unary: UnaryClient;
  private readonly store: query.Table<Key, Project>;
  private readonly ontologyStores: ontology.Stores;

  constructor({ unary, cache, ontologyStores }: ClientParams) {
    const store = cache.createTable<Key, Project>({
      name: "projects",
      fetch: async (keys) => await this.execRetrieve({ keys }),
      listen: [
        query.createSetListener(SET_CHANNEL_NAME, projectZ),
        query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ),
      ],
    });
    super(cache, {
      name: "project",
      table: store,
      request: {
        schema: retrieveReqZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (project, req) => requestFilter(req)(project),
      },
    });
    this.unary = unary;
    this.ontologyStores = ontologyStores;
    this.store = store;
  }

  async create(project: New, opts?: query.WriteOptions<Project[]>): Promise<Project>;
  async create(
    projects: New[],
    opts?: query.WriteOptions<Project[]>,
  ): Promise<Project[]>;
  async create(
    projects: New | New[],
    opts: query.WriteOptions<Project[]> = {},
  ): Promise<Project | Project[]> {
    const isMany = Array.isArray(projects);
    const optimistic = array.toArray(projects).map((p) => projectZ.parse(p));
    const rollback = new destructor.Chain();
    rollback.add(this.store.set(optimistic));
    await opts.onOptimistic?.(optimistic);
    const res = await rollback.guard(
      async () =>
        await this.unary.send(
          "/project/create",
          { projects: optimistic },
          createReqZ,
          createResZ,
        ),
    );
    this.store.set(res.projects);
    return isMany ? res.projects : res.projects[0];
  }

  async rename(key: Key, name: string, opts: query.WriteOptions = {}): Promise<void> {
    const rename = () => [
      query.partialUpdate(this.store, key, { name }),
      ontology.renameCachedResource(this.ontologyStores, ontologyID(key), name),
    ];
    const rollback = new destructor.Chain();
    rollback.add(...rename());
    await opts.onOptimistic?.();
    await rollback.guard(
      async () =>
        await this.unary.send("/project/rename", { key, name }, renameReqZ, emptyResZ),
    );
    rename();
  }

  async setLayout(
    key: Key,
    layout: record.Unknown,
    opts: query.WriteOptions = {},
  ): Promise<void> {
    const rollback = new destructor.Chain();
    rollback.add(query.partialUpdate(this.store, key, { layout }));
    await opts.onOptimistic?.();
    await rollback.guard(
      async () =>
        await this.unary.send(
          "/project/set-layout",
          { key, layout },
          setLayoutReqZ,
          emptyResZ,
        ),
    );
    this.mergeThrough(key, { layout });
  }

  async delete(key: Key, opts?: query.WriteOptions): Promise<void>;
  async delete(keys: Key[], opts?: query.WriteOptions): Promise<void>;
  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const drop = () => [
      ontology.deleteCachedResources(this.ontologyStores, ontologyID(keysArr)),
      this.store.delete(keysArr),
    ];
    const rollback = new destructor.Chain();
    rollback.add(...drop());
    await opts.onOptimistic?.();
    await rollback.guard(
      async () =>
        await this.unary.send(
          "/project/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    );
    drop();
  }

  /** Subscribes to every project delete delivered to the cache. */
  onDelete(handler: (key: Key) => void): destructor.Destructor {
    return this.store.subscribe((event) => {
      if (event.variant === "delete") handler(event.key);
    });
  }

  // Undefined fields are dropped: the server keeps prior values for them.
  private mergeThrough(key: Key, changes: Partial<Project>): void {
    const prev = this.store.get(key);
    if (prev != null)
      this.store.set(key, { ...prev, ...record.purgeUndefined(changes) });
  }

  private async execRetrieve(req: RetrieveRequest): Promise<Project[]> {
    const res = await this.unary.send(
      "/project/retrieve",
      req,
      retrieveReqZ,
      retrieveResZ,
    );
    return res.projects;
  }
}
