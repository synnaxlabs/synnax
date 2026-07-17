// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, type destructor } from "@synnaxlabs/x";
import z from "zod";

import { cache } from "@/cache";
import { bindStore, STORE_KEY } from "@/group/store";
import { type Group, groupZ, type Key, keyZ, ontologyID } from "@/group/types.gen";
import { ontology } from "@/ontology";
import { idZ as ontologyIDZ } from "@/ontology/payload";

const resZ = z.object({ group: groupZ });

const createReqZ = z.object({
  parent: ontologyIDZ,
  key: keyZ.optional(),
  name: z.string(),
});

const renameReqZ = z.object({ key: keyZ, name: z.string() });

const deleteReqZ = z.object({ keys: z.array(keyZ) });

export interface CreateParams extends z.infer<typeof createReqZ> {}

const retrieveChildrenParamsZ = z.object({
  parent: ontologyIDZ,
  searchTerm: z.string().optional(),
  offset: z.int().optional(),
  limit: z.int().optional(),
});

export type RetrieveSingleParams = { key: Key };
export type RetrieveChildrenParams = z.input<typeof retrieveChildrenParamsZ>;
export type RetrieveParams = RetrieveSingleParams | RetrieveChildrenParams;

interface ChildrenRequest extends z.infer<typeof retrieveChildrenParamsZ> {}

const MOUNT_SCOPE = "group.mounts";

export class Client {
  client: UnaryClient;
  private readonly ontologyClient?: ontology.Client;
  private readonly engine_?: cache.Engine;
  private readonly queries_?: {
    single: cache.Queries<Key, Group>;
    children: cache.Queries<ChildrenRequest, Group[]>;
  };

  constructor(
    client: UnaryClient,
    ontologyClient?: ontology.Client,
    engine?: cache.Engine,
  ) {
    this.client = client;
    this.ontologyClient = ontologyClient;
    if (engine == null) return;
    bindStore(engine);
    this.engine_ = engine;
    const ensureStreaming = async () => await engine.ensureStreaming();
    this.queries_ = {
      single: new cache.Queries({
        name: "group",
        fetch: async (query) => await this.fetchSingle(query),
        mount: (params) => this.mountSingle(params),
        ensureStreaming,
      }),
      children: new cache.Queries({
        name: "groups",
        fetch: async (query) => await this.fetchChildren(query),
        mount: (params) => this.mountChildren(params),
        ensureStreaming,
      }),
    };
  }

  async create(params: CreateParams): Promise<Group> {
    const res = await this.client.send(
      "/ontology/create-group",
      params,
      createReqZ,
      resZ,
    );
    const writes = this.writes;
    if (writes != null) {
      writes.set(res.group.key, res.group);
      const rel: ontology.Relationship = {
        from: params.parent,
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
        to: ontologyID(res.group.key),
      };
      this.relationshipWrites?.set(ontology.relationshipToString(rel), rel);
    }
    return res.group;
  }

  async retrieve(params: RetrieveSingleParams): Promise<Group>;
  async retrieve(params: RetrieveChildrenParams): Promise<Group[]>;
  async retrieve(params: RetrieveParams): Promise<Group | Group[]> {
    if (this.queries_ == null) {
      if ("key" in params) return await this.execRetrieveSingle(params.key);
      return await this.execRetrieveChildren(retrieveChildrenParamsZ.parse(params));
    }
    if ("key" in params) return await this.queries_.single.retrieve(params.key);
    return await this.queries_.children.retrieve(retrieveChildrenParamsZ.parse(params));
  }

  /**
   * Subscribes to changes in the cached answer to the given query. Single
   * queries deliver a group; children queries deliver the matching groups.
   * @throws when the cache was disabled at client construction.
   */
  onChange(
    params: RetrieveSingleParams,
    handler: cache.ChangeHandler<Group>,
  ): destructor.Destructor;
  onChange(
    params: RetrieveChildrenParams,
    handler: cache.ChangeHandler<Group[]>,
  ): destructor.Destructor;
  onChange(
    params: RetrieveParams,
    handler: cache.ChangeHandler<Group> | cache.ChangeHandler<Group[]>,
  ): destructor.Destructor {
    const queries = this.requireQueries();
    if ("key" in params)
      return queries.single.onChange(params.key, handler as cache.ChangeHandler<Group>);
    return queries.children.onChange(
      retrieveChildrenParamsZ.parse(params),
      handler as cache.ChangeHandler<Group[]>,
    );
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached.
   * @throws when the cache was disabled at client construction.
   */
  getCached(params: RetrieveSingleParams): cache.Cached<Group> | undefined;
  getCached(params: RetrieveChildrenParams): cache.Cached<Group[]> | undefined;
  getCached(
    params: RetrieveParams,
  ): cache.Cached<Group> | cache.Cached<Group[]> | undefined {
    const queries = this.requireQueries();
    if ("key" in params) return queries.single.getCached(params.key);
    return queries.children.getCached(retrieveChildrenParamsZ.parse(params));
  }

  async rename(key: Key, name: string, opts: cache.WriteOptions = {}): Promise<void> {
    const rollback = new cache.Rollback();
    const writes = this.writes;
    if (this.engine_ != null && writes != null) {
      rollback.add(cache.partialUpdate(writes, key, { name }));
      rollback.add(ontology.renameCachedResource(this.engine_, ontologyID(key), name));
    }
    await opts.onOptimistic?.();
    await rollback.guard(
      async () =>
        await this.client.send(
          "/ontology/rename-group",
          { key, name },
          renameReqZ,
          z.object({}),
        ),
    );
    // Re-applied after success: a stale streamer echo may have clobbered the
    // optimistic write while the send was in flight.
    if (writes != null) cache.partialUpdate(writes, key, { name });
  }

  async delete(keys: Key | Key[]): Promise<void> {
    const keysArr = array.toArray(keys);
    await this.client.send(
      "/ontology/delete-group",
      { keys: keysArr },
      deleteReqZ,
      z.object({}),
    );
    this.writes?.delete(keysArr);
  }

  private get writes(): cache.UnaryStore<Key, Group> | undefined {
    return this.engine_?.store(STORE_KEY);
  }

  private get relationshipWrites():
    cache.UnaryStore<string, ontology.Relationship> | undefined {
    return this.engine_?.store(ontology.RELATIONSHIPS_STORE_KEY);
  }

  private get groupStore(): cache.UnaryStore<Key, Group> {
    return this.requireEngine().store(STORE_KEY);
  }

  // Query mounts subscribe in their own scope: stores suppress notifications
  // to listeners in the writer's scope, and the streamer writes in the default
  // scope, which would silence default-scope subscriptions entirely.
  private get groupEvents(): cache.UnaryStore<Key, Group> {
    return this.requireEngine().store(STORE_KEY, MOUNT_SCOPE);
  }

  private get relationshipEvents(): cache.UnaryStore<string, ontology.Relationship> {
    return this.requireEngine().store(ontology.RELATIONSHIPS_STORE_KEY, MOUNT_SCOPE);
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

  private requireOntology(): ontology.Client {
    if (this.ontologyClient == null)
      throw new Error("ontology client is not available on this client");
    return this.ontologyClient;
  }

  private async execRetrieveSingle(key: Key): Promise<Group> {
    const res = await this.requireOntology().retrieve(ontologyID(key));
    return groupZ.parse(res.data);
  }

  private async execRetrieveChildren(query: ChildrenRequest): Promise<Group[]> {
    const { parent, ...options } = query;
    const res = await this.requireOntology().retrieveChildren(parent, {
      ...options,
      types: ["group"],
    });
    return res.map((r) => groupZ.parse(r.data));
  }

  private async fetchSingle(query: Key): Promise<Group> {
    const cached = this.groupStore.get(query);
    if (cached != null) return cached;
    const group = await this.execRetrieveSingle(query);
    this.groupStore.set(group.key, group);
    return group;
  }

  private mountSingle({ query, update, remove }: cache.MountParams<Key, Group>) {
    return [
      this.groupEvents.onSet((group) => {
        if (group.key === query) update(group);
      }),
      this.groupEvents.onDelete((key) => {
        if (key === query) remove(this.groupStore.getTombstone(key)?.corpse);
      }),
    ];
  }

  private async fetchChildren(query: ChildrenRequest): Promise<Group[]> {
    const groups = await this.execRetrieveChildren(query);
    this.groupStore.set(groups);
    const rels = this.relationshipWrites;
    groups.forEach((g) => {
      const rel: ontology.Relationship = {
        from: query.parent,
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
        to: ontologyID(g.key),
      };
      rels?.set(ontology.relationshipToString(rel), rel);
    });
    return groups;
  }

  private mountChildren({
    query,
    update,
  }: cache.MountParams<ChildrenRequest, Group[]>) {
    const isChild = (rel: ontology.Relationship) =>
      ontology.matchRelationship(rel, {
        from: query.parent,
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
        to: { type: "group" },
      });
    return [
      this.groupEvents.onSet((group) => {
        update((prev) => {
          if (prev == null || !prev.some((g) => g.key === group.key)) return prev;
          return prev.map((g) => (g.key === group.key ? group : g));
        });
      }),
      this.groupEvents.onDelete((key) => {
        update((prev) => prev?.filter((g) => g.key !== key));
      }),
      this.relationshipEvents.onSet((rel) => {
        if (!isChild(rel)) return;
        void this.fetchSingle(rel.to.key).then((group) => {
          update((prev) => {
            if (prev == null) return prev;
            if (prev.some((g) => g.key === group.key))
              return prev.map((g) => (g.key === group.key ? group : g));
            return [...prev, group];
          });
        });
      }),
      this.relationshipEvents.onDelete((relKey) => {
        const rel = ontology.relationshipZ.parse(relKey);
        if (isChild(rel)) update((prev) => prev?.filter((g) => g.key !== rel.to.key));
      }),
    ];
  }
}
