// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, destructor, primitive } from "@synnaxlabs/x";
import { z } from "zod";

import {
  type Key,
  keyZ,
  type New,
  ontologyID,
  type Role,
  roleZ,
} from "@/access/role/types.gen";
import { ontology } from "@/ontology";
import { query } from "@/query";
import { user } from "@/user";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export const SET_CHANNEL_NAME = "sy_role_set";
export const DELETE_CHANNEL_NAME = "sy_role_delete";

const retrieveRequestZ = z.object({
  keys: keyZ.array().optional(),
  limit: z.number().optional(),
  offset: z.number().optional(),
  internal: z.boolean().optional(),
});

const keyRetrieveRequestZ = z
  .object({ key: keyZ })
  .transform(({ key }) => ({ keys: [key] }));

const singleCreateParamsZ = roleZ.transform((r) => ({ roles: [r] }));
export type SingleCreateParams = z.input<typeof singleCreateParamsZ>;

export const multipleCreateParamsZ = roleZ.array().transform((roles) => ({ roles }));

export const createParamsZ = z.union([singleCreateParamsZ, multipleCreateParamsZ]);
export type CreateParams = z.input<typeof createParamsZ>;

const createResZ = z.object({ roles: roleZ.array() });
const retrieveResZ = z.object({ roles: roleZ.array().default(() => []) });

export type RetrieveSingleParams = z.input<typeof keyRetrieveRequestZ>;
export type RetrieveMultipleParams = z.input<typeof retrieveRequestZ>;

export const retrieveParamsZ = z.union([keyRetrieveRequestZ, retrieveRequestZ]);
export type RetrieveParams = z.input<typeof retrieveParamsZ>;

interface RetrieveRequest extends z.infer<typeof retrieveRequestZ> {}

const deleteResZ = z.object({});

const deleteParamsZ = keyZ
  .transform((key) => ({ keys: [key] }))
  .or(keyZ.array().transform((keys) => ({ keys })));
export type DeleteParams = z.input<typeof deleteParamsZ>;

const assignReqZ = z.object({
  user: user.keyZ,
  role: keyZ,
});
export type AssignParams = z.input<typeof assignReqZ>;

const assignResZ = z.object({});

const unassignReqZ = z.object({
  user: user.keyZ,
  role: keyZ,
});
export type UnassignParams = z.input<typeof unassignReqZ>;

const unassignResZ = z.object({});

/** Query fields only the server can evaluate. */
const SERVER_FIELDS = ["limit", "offset"] as const;

const normalizeRequest = (params: RetrieveMultipleParams): RetrieveRequest =>
  retrieveRequestZ.parse(params);

const isKeysOnly = (req: RetrieveRequest): req is RetrieveRequest & { keys: Key[] } =>
  primitive.isNonZero(req.keys) &&
  req.internal == null &&
  req.limit == null &&
  req.offset == null;

/**
 * Client-side matching for a request: key sets and the internal flag.
 * Server-computed shapes (pagination) never reach this filter; they refetch
 * instead.
 */
const requestFilter = (req: RetrieveRequest): ((r: Role) => boolean) => {
  const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
  return (r) => {
    if (keySet != null && !keySet.has(r.key)) return false;
    if (req.internal != null && r.internal !== req.internal) return false;
    return true;
  };
};

const assignmentRel = (role: Key, userKey: user.Key): ontology.Relationship => ({
  from: ontologyID(role),
  type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
  to: user.ontologyID(userKey),
});

const createTable = (cache: query.Cache): query.Table<Key, Role> => {
  const table = cache.createTable<Key, Role>({ name: "roles" });
  const set: query.ChannelListener<typeof roleZ> = {
    channel: SET_CHANNEL_NAME,
    schema: roleZ,
    onChange: table.set.bind(table),
  };
  const del: query.ChannelListener<typeof keyZ> = {
    channel: DELETE_CHANNEL_NAME,
    schema: keyZ,
    onChange: table.delete.bind(table),
  };
  cache.addListeners(table, set, del);
  return table;
};

export class Client extends query.Retriever<
  RetrieveSingleParams,
  RetrieveMultipleParams,
  Key,
  RetrieveRequest,
  Role
> {
  private readonly client: UnaryClient;
  private readonly store: query.Table<Key, Role>;
  private readonly ontology: ontology.Stores;

  constructor(
    client: UnaryClient,
    cache: query.Cache,
    ontologyStores: ontology.Stores,
  ) {
    const store = createTable(cache);
    super({
      single: cache.queries({
        name: "role",
        table: store,
        fetch: async (query) => [await this.fetchSingle(query)].map((r) => r.key),
        compose: (records) => records[0],
        keyOf: (query) => query,
        single: true,
      }),
      request: cache.queries({
        name: "roles",
        table: store,
        fetch: async (query) => (await this.fetchRequest(query)).map((r) => r.key),
        compose: (records) => records,
        matches: (role, query) => requestFilter(query)(role),
        serverFields: SERVER_FIELDS,
      }),
      isSingle: (params) => "key" in params,
      normalizeSingle: ({ key }) => key,
      normalizeRequest,
    });
    this.client = client;
    this.store = store;
    this.ontology = ontologyStores;
  }

  async create(role: New): Promise<Role>;
  async create(roles: New[]): Promise<Role[]>;
  async create(roles: New | New[]): Promise<Role | Role[]> {
    const isMany = Array.isArray(roles);
    const res = await this.client.send(
      "/access/role/create",
      roles,
      createParamsZ,
      createResZ,
    );
    this.store.set(res.roles);
    return isMany ? res.roles : res.roles[0];
  }

  async delete(params: DeleteParams, opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(params);
    const ids = ontologyID(keysArr);
    const rollback = new destructor.Chain();
    rollback.add(ontology.deleteCachedResources(this.ontology, ids));
    rollback.add(this.store.delete(keysArr));
    await opts.onOptimistic?.();
    await rollback.guard(
      async () =>
        await this.client.send(
          "/access/role/delete",
          params,
          deleteParamsZ,
          deleteResZ,
        ),
    );
    this.store.delete(keysArr);
    this.ontology.relationships.delete((r) =>
      ids.some((id) => ontology.idsEqual(r.from, id) || ontology.idsEqual(r.to, id)),
    );
  }

  async rename(key: Key, name: string): Promise<void> {
    const existing = await this.retrieve({ key });
    await this.create({ ...existing, name });
    ontology.renameCachedResource(this.ontology, ontologyID(key), name);
  }

  async assign(params: AssignParams): Promise<void> {
    await this.client.send("/access/role/assign", params, assignReqZ, assignResZ);
    const rel = assignmentRel(params.role, params.user);
    this.ontology.relationships.set(ontology.relationshipToString(rel), rel);
  }

  async unassign(params: UnassignParams): Promise<void> {
    await this.client.send("/access/role/unassign", params, unassignReqZ, unassignResZ);
    this.ontology.relationships.delete(
      ontology.relationshipToString(assignmentRel(params.role, params.user)),
    );
  }

  private async execRetrieve(params: RetrieveParams): Promise<Role[]> {
    const res = await this.client.send(
      "/access/role/retrieve",
      params,
      retrieveParamsZ,
      retrieveResZ,
    );
    return res.roles;
  }

  /**
   * Fetches the given keys, serving cached entries and fetching only the
   * misses. Preserves the caller's key order.
   */
  private async fetchKeys(keys: Key[]): Promise<Role[]> {
    const results: Role[] = [];
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
    return query.orderByKeys(keys, results, (r) => r.key);
  }

  private async fetchSingle(query: Key): Promise<Role> {
    const cached = this.store.get(query);
    if (cached != null) return cached;
    const roles = await this.execRetrieve({ key: query });
    checkForMultipleOrNoResults("Role", query, roles, true);
    this.store.set(roles);
    return roles[0];
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Role[]> {
    if (isKeysOnly(query)) return await this.fetchKeys(query.keys);
    const roles = await this.execRetrieve(query);
    this.store.set(roles);
    return roles;
  }
}
