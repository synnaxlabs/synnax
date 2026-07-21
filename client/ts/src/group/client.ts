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
import { type Group, groupZ, type Key, keyZ, ontologyID } from "@/group/types.gen";
import { ontology } from "@/ontology";
import { idZ as ontologyIDZ } from "@/ontology/payload";

export const SET_CHANNEL_NAME = "sy_group_set";
export const DELETE_CHANNEL_NAME = "sy_group_delete";

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

/** Query fields only the server can evaluate. */
const SERVER_FIELDS = ["searchTerm", "limit", "offset"] as const;

const isChildOf = (rel: ontology.Relationship, parent: ontology.ID): boolean =>
  ontology.matchRelationship(rel, {
    from: parent,
    type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
    to: { type: "group" },
  });

export class Client {
  client: UnaryClient;
  private readonly ontologyClient?: ontology.Client;
  private readonly store: cache.Table<Key, Group>;
  private readonly ontology: ontology.Stores;
  private readonly answers: {
    single: cache.Answers<Key, Group, Key, Group>;
    children: cache.Answers<ChildrenRequest, Group[], Key, Group>;
  };

  constructor(
    client: UnaryClient,
    ontologyClient: ontology.Client | undefined,
    engine: cache.Cache,
    ontologyStores: ontology.Stores,
  ) {
    this.client = client;
    this.ontologyClient = ontologyClient;
    this.store = this.createTable(engine);
    this.ontology = ontologyStores;
    this.answers = {
      single: engine.answers({
        name: "group",
        table: this.store,
        fetch: async (query) => [await this.fetchSingle(query)].map((g) => g.key),
        compose: (records) => records[0],
        keyOf: (query) => query,
        single: true,
      }),
      children: engine.answers({
        name: "groups",
        table: this.store,
        fetch: async (query) => (await this.fetchChildren(query)).map((g) => g.key),
        compose: (records) => records,
        matches: (group, query) => this.isCachedChild(query.parent, group.key),
        serverFields: SERVER_FIELDS,
        watch: [
          cache.watch(this.ontology.relationships, (event, query: ChildrenRequest) => {
            const rel =
              event.variant === "set"
                ? event.value
                : ontology.relationshipZ.parse(event.key);
            if (!isChildOf(rel, query.parent)) return null;
            return [rel.to.key];
          }),
        ],
        hydrate: async (keys) => {
          await Promise.all(keys.map(async (key) => await this.fetchSingle(key)));
        },
      }),
    };
  }

  private createTable(engine: cache.Cache): cache.Table<Key, Group> {
    const table = engine.createTable<Key, Group>({ name: "groups" });
    const set: cache.ChannelListener<typeof groupZ> = {
      channel: SET_CHANNEL_NAME,
      schema: groupZ,
      onChange: (changed) => table.set(changed.key, changed),
    };
    const del: cache.ChannelListener<typeof keyZ> = {
      channel: DELETE_CHANNEL_NAME,
      schema: keyZ,
      onChange: (changed) => table.delete(changed),
    };
    engine.addListeners(table, set, del);
    return table;
  }

  async create(params: CreateParams): Promise<Group> {
    const res = await this.client.send(
      "/ontology/create-group",
      params,
      createReqZ,
      resZ,
    );
    this.store.set(res.group.key, res.group);
    const rel: ontology.Relationship = {
      from: params.parent,
      type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
      to: ontologyID(res.group.key),
    };
    this.ontology.relationships.set(ontology.relationshipToString(rel), rel);
    return res.group;
  }

  async retrieve(params: RetrieveSingleParams): Promise<Group>;
  async retrieve(params: RetrieveChildrenParams): Promise<Group[]>;
  async retrieve(params: RetrieveParams): Promise<Group | Group[]> {
    if ("key" in params) return await this.answers.single.retrieve(params.key);
    return await this.answers.children.retrieve(retrieveChildrenParamsZ.parse(params));
  }

  /**
   * Subscribes to changes in the cached answer to the given query. Single
   * queries deliver a group; children queries deliver the matching groups.
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
    const { children, single } = this.answers;
    if ("key" in params)
      return single.onChange(params.key, handler as cache.ChangeHandler<Group>);
    return children.onChange(
      retrieveChildrenParamsZ.parse(params),
      handler as cache.ChangeHandler<Group[]>,
    );
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached.
   */
  getCached(params: RetrieveSingleParams): cache.Cached<Group> | undefined;
  getCached(params: RetrieveChildrenParams): cache.Cached<Group[]> | undefined;
  getCached(
    params: RetrieveParams,
  ): cache.Cached<Group> | cache.Cached<Group[]> | undefined {
    const { children, single } = this.answers;
    if ("key" in params) return single.getCached(params.key);
    return children.getCached(retrieveChildrenParamsZ.parse(params));
  }

  async rename(key: Key, name: string, opts: cache.WriteOptions = {}): Promise<void> {
    const rollback = new cache.Rollback();
    rollback.add(cache.partialUpdate(this.store, key, { name }));
    rollback.add(ontology.renameCachedResource(this.ontology, ontologyID(key), name));
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
    cache.partialUpdate(this.store, key, { name });
  }

  async delete(keys: Key | Key[]): Promise<void> {
    const keysArr = array.toArray(keys);
    await this.client.send(
      "/ontology/delete-group",
      { keys: keysArr },
      deleteReqZ,
      z.object({}),
    );
    this.store.delete(keysArr);
  }

  private isCachedChild(parent: ontology.ID, key: Key): boolean {
    return this.ontology.relationships.has(
      ontology.relationshipToString({
        from: parent,
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
        to: ontologyID(key),
      }),
    );
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
    const cached = this.store.get(query);
    if (cached != null) return cached;
    const group = await this.execRetrieveSingle(query);
    this.store.set(group.key, group);
    return group;
  }

  private async fetchChildren(query: ChildrenRequest): Promise<Group[]> {
    const groups = await this.execRetrieveChildren(query);
    this.store.set(groups);
    const rels = this.ontology.relationships;
    groups.forEach((g) => {
      const rel: ontology.Relationship = {
        from: query.parent,
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
        to: ontologyID(g.key),
      };
      rels.set(ontology.relationshipToString(rel), rel);
    });
    return groups;
  }
}
