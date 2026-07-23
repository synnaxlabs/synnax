// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array } from "@synnaxlabs/x";
import z from "zod";

import { type Group, groupZ, type Key, keyZ, ontologyID } from "@/group/types.gen";
import { ontology } from "@/ontology";
import { idZ as ontologyIDZ } from "@/ontology/payload";
import { query } from "@/query";

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

const createTable = (cache: query.Cache): query.Table<Key, Group> => {
  const table = cache.createTable<Key, Group>({ name: "groups" });
  const set: query.ChannelListener<typeof groupZ> = {
    channel: SET_CHANNEL_NAME,
    schema: groupZ,
    onChange: (changed) => table.set(changed),
  };
  const del: query.ChannelListener<typeof keyZ> = {
    channel: DELETE_CHANNEL_NAME,
    schema: keyZ,
    onChange: (changed) => table.delete(changed),
  };
  cache.addListeners(table, set, del);
  return table;
};

export class Client extends query.Retriever<
  RetrieveSingleParams,
  RetrieveChildrenParams,
  Key,
  ChildrenRequest,
  Group
> {
  client: UnaryClient;
  private readonly ontologyClient?: ontology.Client;
  private readonly store: query.Table<Key, Group>;
  private readonly ontology: ontology.Stores;

  constructor(
    client: UnaryClient,
    ontologyClient: ontology.Client | undefined,
    cache: query.Cache,
    ontologyStores: ontology.Stores,
  ) {
    const store = createTable(cache);
    super({
      single: cache.queries({
        name: "group",
        table: store,
        fetch: async (query) => [await this.fetchSingle(query)].map((g) => g.key),
        compose: (records) => records[0],
        keyOf: (query) => query,
        single: true,
      }),
      request: cache.queries({
        name: "groups",
        table: store,
        fetch: async (query) => (await this.fetchChildren(query)).map((g) => g.key),
        compose: (records) => records,
        matches: (group, query) => this.isCachedChild(query.parent, group.key),
        serverFields: SERVER_FIELDS,
        watch: [
          query.watch(ontologyStores.relationships, (event, query: ChildrenRequest) => {
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
      isSingle: (params) => "key" in params,
      normalizeSingle: ({ key }) => key,
      normalizeRequest: (params) => retrieveChildrenParamsZ.parse(params),
    });
    this.client = client;
    this.ontologyClient = ontologyClient;
    this.store = store;
    this.ontology = ontologyStores;
  }

  async create(params: CreateParams): Promise<Group> {
    const res = await this.client.send(
      "/ontology/create-group",
      params,
      createReqZ,
      resZ,
    );
    this.store.set(res.group);
    const rel: ontology.Relationship = {
      from: params.parent,
      type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
      to: ontologyID(res.group.key),
    };
    this.ontology.relationships.set(ontology.relationshipToString(rel), rel);
    return res.group;
  }

  async rename(key: Key, name: string, opts: query.WriteOptions = {}): Promise<void> {
    const rollback = new query.Rollback();
    rollback.add(query.partialUpdate(this.store, key, { name }));
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
    query.partialUpdate(this.store, key, { name });
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
    this.store.set(group);
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
