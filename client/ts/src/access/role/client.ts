// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, type destructor, primitive } from "@synnaxlabs/x";
import { z } from "zod";

import { bindStore, STORE_KEY } from "@/access/role/store";
import {
  type Key,
  keyZ,
  type New,
  ontologyID,
  type Role,
  roleZ,
} from "@/access/role/types.gen";
import { cache } from "@/cache";
import { ontology } from "@/ontology";
import { user } from "@/user";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

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

const isKeysOnly = (req: RetrieveRequest): boolean =>
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

export class Client {
  private readonly client: UnaryClient;
  private readonly cache_: cache.Cache;
  private readonly answers_: {
    single: cache.Answers<Key, Role, Key, Role>;
    request: cache.Answers<RetrieveRequest, Role[], Key, Role>;
  };

  constructor(client: UnaryClient, engine: cache.Cache) {
    this.client = client;
    bindStore(engine);
    this.cache_ = engine;
    this.answers_ = {
      single: engine.answers({
        name: "role",
        table: this.roleStore,
        fetch: async (query) => [await this.fetchSingle(query)].map((r) => r.key),
        compose: (records) => records[0],
        keyOf: (query) => query,
        single: true,
      }),
      request: engine.answers({
        name: "roles",
        table: this.roleStore,
        fetch: async (query) => (await this.fetchRequest(query)).map((r) => r.key),
        compose: (records) => records,
        matches: (role, query) => requestFilter(query)(role),
        serverFields: SERVER_FIELDS,
      }),
    };
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
    this.writes.setMany(res.roles);
    return isMany ? res.roles : res.roles[0];
  }

  async retrieve(params: RetrieveSingleParams): Promise<Role>;
  async retrieve(params: RetrieveMultipleParams): Promise<Role[]>;
  async retrieve(params: RetrieveParams): Promise<Role | Role[]> {
    const isSingle = "key" in params;
    if (isSingle) return await this.answers_.single.retrieve(params.key);
    return await this.answers_.request.retrieve(normalizeRequest(params));
  }

  /**
   * Subscribes to changes in the cached answer to the given query. Single
   * queries deliver a role; every other shape delivers the matching roles.
   */
  onChange(
    params: RetrieveSingleParams,
    handler: cache.ChangeHandler<Role>,
  ): destructor.Destructor;
  onChange(
    params: RetrieveMultipleParams,
    handler: cache.ChangeHandler<Role[]>,
  ): destructor.Destructor;
  onChange(
    params: RetrieveParams,
    handler: cache.ChangeHandler<Role> | cache.ChangeHandler<Role[]>,
  ): destructor.Destructor {
    const answers = this.answers_;
    if ("key" in params)
      return answers.single.onChange(params.key, handler as cache.ChangeHandler<Role>);
    return answers.request.onChange(
      normalizeRequest(params),
      handler as cache.ChangeHandler<Role[]>,
    );
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached.
   */
  getCached(params: RetrieveSingleParams): cache.Cached<Role> | undefined;
  getCached(params: RetrieveMultipleParams): cache.Cached<Role[]> | undefined;
  getCached(
    params: RetrieveParams,
  ): cache.Cached<Role> | cache.Cached<Role[]> | undefined {
    const answers = this.answers_;
    if ("key" in params) return answers.single.getCached(params.key);
    return answers.request.getCached(normalizeRequest(params));
  }

  async delete(params: DeleteParams, opts: cache.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(params);
    const ids = ontologyID(keysArr);
    const rollback = new cache.Rollback();
    const writes = this.writes;
    rollback.add(ontology.deleteCachedResources(this.cache_, ids));
    rollback.add(writes.delete(keysArr));
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
    this.writes.delete(keysArr);
    this.relationshipWrites.delete((r) =>
      ids.some((id) => ontology.idsEqual(r.from, id) || ontology.idsEqual(r.to, id)),
    );
  }

  async rename(key: Key, name: string): Promise<void> {
    const existing = await this.retrieve({ key });
    await this.create({ ...existing, name });
    ontology.renameCachedResource(this.cache_, ontologyID(key), name);
  }

  async assign(params: AssignParams): Promise<void> {
    await this.client.send("/access/role/assign", params, assignReqZ, assignResZ);
    const rel = assignmentRel(params.role, params.user);
    this.relationshipWrites.set(ontology.relationshipToString(rel), rel);
  }

  async unassign(params: UnassignParams): Promise<void> {
    await this.client.send("/access/role/unassign", params, unassignReqZ, unassignResZ);
    this.relationshipWrites.delete(
      ontology.relationshipToString(assignmentRel(params.role, params.user)),
    );
  }

  private get writes(): cache.Table<Key, Role> {
    return this.cache_.table(STORE_KEY);
  }

  private get relationshipWrites(): cache.Table<string, ontology.Relationship> {
    return this.cache_.table(ontology.RELATIONSHIPS_STORE_KEY);
  }

  private get roleStore(): cache.Table<Key, Role> {
    return this.cache_.table(STORE_KEY);
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
      const cached = this.roleStore.get(key);
      if (cached != null) results.push(cached);
      else misses.push(key);
    }
    if (misses.length > 0) {
      const fetched = await this.execRetrieve({ keys: misses });
      this.roleStore.setMany(fetched);
      results.push(...fetched);
    }
    return cache.orderByKeys(keys, results, (r) => r.key);
  }

  private async fetchSingle(query: Key): Promise<Role> {
    const cached = this.roleStore.get(query);
    if (cached != null) return cached;
    const roles = await this.execRetrieve({ key: query });
    checkForMultipleOrNoResults("Role", query, roles, true);
    this.roleStore.setMany(roles);
    return roles[0];
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Role[]> {
    if (isKeysOnly(query)) return await this.fetchKeys(query.keys as Key[]);
    const roles = await this.execRetrieve(query);
    this.roleStore.setMany(roles);
    return roles;
  }
}
