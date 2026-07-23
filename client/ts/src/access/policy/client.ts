// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, primitive } from "@synnaxlabs/x";
import { z } from "zod";

import {
  type Key,
  keyZ,
  type New,
  ontologyID,
  type Policy,
  policyZ,
} from "@/access/policy/types.gen";
import { ontology } from "@/ontology";
import { query } from "@/query";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export const SET_CHANNEL_NAME = "sy_policy_set";
export const DELETE_CHANNEL_NAME = "sy_policy_delete";

const retrieveRequestZ = z.object({
  keys: keyZ.array().optional(),
  subjects: ontology.idZ.array().optional(),
  limit: z.number().optional(),
  offset: z.number().optional(),
  internal: z.boolean().optional(),
});

const keyRetrieveRequestZ = z
  .object({ key: keyZ })
  .transform(({ key }) => ({ keys: [key] }));

const listRetrieveParamsZ = z.union([
  z
    .object({ for: ontology.idZ })
    .transform(({ for: forId }) => ({ subjects: [forId] })),
  z
    .object({ for: ontology.idZ.array() })
    .transform(({ for: forIds }) => ({ subjects: forIds })),
  retrieveRequestZ,
]);

export type RetrieveSingleParams = z.input<typeof keyRetrieveRequestZ>;
export type RetrieveMultipleParams = z.input<typeof listRetrieveParamsZ>;

const retrieveParamsZ = z.union([keyRetrieveRequestZ, listRetrieveParamsZ]);

export type RetrieveParams = z.input<typeof retrieveParamsZ>;

interface RetrieveRequest extends z.infer<typeof retrieveRequestZ> {}

const retrieveResZ = z.object({ policies: policyZ.array().default(() => []) });

const singleCreateParamsZ = policyZ.transform((p) => ({ policies: [p] }));
export type SingleCreateParams = z.input<typeof singleCreateParamsZ>;

export const multipleCreateParamsZ = policyZ
  .array()
  .transform((policies) => ({ policies }));

export const createParamsZ = z.union([singleCreateParamsZ, multipleCreateParamsZ]);
export type CreateParams = z.input<typeof createParamsZ>;

const createResZ = z.object({ policies: policyZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const deleteResZ = z.object({});

/**
 * Query fields only the server can evaluate: policy records do not carry
 * their subjects, so subject membership cannot be checked locally.
 */
const SERVER_FIELDS = ["subjects", "limit", "offset"] as const;

const normalizeRequest = (params: RetrieveMultipleParams): RetrieveRequest =>
  listRetrieveParamsZ.parse(params);

const isKeysOnly = (req: RetrieveRequest): req is RetrieveRequest & { keys: Key[] } =>
  primitive.isNonZero(req.keys) &&
  req.subjects == null &&
  req.internal == null &&
  req.limit == null &&
  req.offset == null;

/**
 * Client-side matching for a request: key sets and the internal flag.
 * Server-computed shapes (subjects, pagination) never reach this filter; they
 * refetch instead.
 */
const requestFilter = (req: RetrieveRequest): ((p: Policy) => boolean) => {
  const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
  return (p) => {
    if (keySet != null && !keySet.has(p.key)) return false;
    if (req.internal != null && p.internal !== req.internal) return false;
    return true;
  };
};

const createTable = (cache: query.Cache): query.Table<Key, Policy> => {
  const table = cache.createTable<Key, Policy>({ name: "policies" });
  const set: query.ChannelListener<typeof policyZ> = {
    channel: SET_CHANNEL_NAME,
    schema: policyZ,
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
  RetrieveMultipleParams,
  Key,
  RetrieveRequest,
  Policy
> {
  private readonly client: UnaryClient;
  private readonly store: query.Table<Key, Policy>;
  private readonly ontology: ontology.Stores;

  constructor(
    client: UnaryClient,
    cache: query.Cache,
    ontologyStores: ontology.Stores,
  ) {
    const store = createTable(cache);
    super({
      single: cache.queries({
        name: "policy",
        table: store,
        fetch: async (query) => [await this.fetchSingle(query)].map((p) => p.key),
        compose: (records) => records[0],
        keyOf: (query) => query,
        single: true,
      }),
      request: cache.queries({
        name: "policies",
        table: store,
        fetch: async (query) => (await this.fetchRequest(query)).map((p) => p.key),
        compose: (records) => records,
        matches: (policy, query) => requestFilter(query)(policy),
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

  async create(policy: New): Promise<Policy>;
  async create(policies: New[]): Promise<Policy[]>;
  async create(policies: CreateParams): Promise<Policy | Policy[]> {
    const isMany = Array.isArray(policies);
    const res = await this.client.send(
      "/access/policy/create",
      policies,
      createParamsZ,
      createResZ,
    );
    this.store.set(res.policies);
    return isMany ? res.policies : res.policies[0];
  }

  async delete(key: Key, opts?: query.WriteOptions): Promise<void>;
  async delete(keys: Key[], opts?: query.WriteOptions): Promise<void>;
  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const ids = ontologyID(keysArr);
    const rollback = new query.Rollback();
    rollback.add(ontology.deleteCachedResources(this.ontology, ids));
    rollback.add(this.store.delete(keysArr));
    await opts.onOptimistic?.();
    await rollback.guard(
      async () =>
        await this.client.send(
          "/access/policy/delete",
          { keys: keysArr },
          deleteReqZ,
          deleteResZ,
        ),
    );
    this.store.delete(keysArr);
    this.ontology.relationships.delete((r) =>
      ids.some((id) => ontology.idsEqual(r.from, id) || ontology.idsEqual(r.to, id)),
    );
  }

  async rename(key: Key, name: string, opts: query.WriteOptions = {}): Promise<void> {
    const existing = await this.retrieve({ key });
    const rollback = new query.Rollback();
    rollback.add(query.partialUpdate(this.store, key, { name }));
    rollback.add(ontology.renameCachedResource(this.ontology, ontologyID(key), name));
    await opts.onOptimistic?.();
    await rollback.guard(async () => {
      await this.create({ ...existing, name });
    });
  }

  private async execRetrieve(params: RetrieveParams): Promise<Policy[]> {
    const res = await this.client.send(
      "/access/policy/retrieve",
      params,
      retrieveParamsZ,
      retrieveResZ,
    );
    return res.policies;
  }

  /**
   * Fetches the given keys, serving cached entries and fetching only the
   * misses. Preserves the caller's key order.
   */
  private async fetchKeys(keys: Key[]): Promise<Policy[]> {
    const results: Policy[] = [];
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
    return query.orderByKeys(keys, results, (p) => p.key);
  }

  private async fetchSingle(query: Key): Promise<Policy> {
    const cached = this.store.get(query);
    if (cached != null) return cached;
    const policies = await this.execRetrieve({ key: query });
    checkForMultipleOrNoResults("Policy", query, policies, true);
    this.store.set(policies);
    return policies[0];
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Policy[]> {
    if (isKeysOnly(query)) return await this.fetchKeys(query.keys);
    const policies = await this.execRetrieve(query);
    this.store.set(policies);
    return policies;
  }
}
