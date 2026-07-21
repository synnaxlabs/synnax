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

import { bindStore, STORE_KEY } from "@/access/policy/store";
import {
  type Key,
  keyZ,
  type New,
  ontologyID,
  type Policy,
  policyZ,
} from "@/access/policy/types.gen";
import { cache } from "@/cache";
import { ontology } from "@/ontology";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

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

const isKeysOnly = (req: RetrieveRequest): boolean =>
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

export class Client {
  private readonly client: UnaryClient;
  private readonly cache_: cache.Cache;
  private readonly answers_: {
    single: cache.Answers<Key, Policy, Key, Policy>;
    request: cache.Answers<RetrieveRequest, Policy[], Key, Policy>;
  };

  constructor(client: UnaryClient, engine: cache.Cache) {
    this.client = client;
    bindStore(engine);
    this.cache_ = engine;
    this.answers_ = {
      single: engine.answers({
        name: "policy",
        table: this.policyStore,
        fetch: async (query) => [await this.fetchSingle(query)].map((p) => p.key),
        compose: (records) => records[0],
        keyOf: (query) => query,
        single: true,
      }),
      request: engine.answers({
        name: "policies",
        table: this.policyStore,
        fetch: async (query) => (await this.fetchRequest(query)).map((p) => p.key),
        compose: (records) => records,
        matches: (policy, query) => requestFilter(query)(policy),
        serverFields: SERVER_FIELDS,
      }),
    };
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
    this.writes.setMany(res.policies);
    return isMany ? res.policies : res.policies[0];
  }

  async retrieve(params: RetrieveSingleParams): Promise<Policy>;
  async retrieve(params: RetrieveMultipleParams): Promise<Policy[]>;
  async retrieve(params: RetrieveParams): Promise<Policy | Policy[]> {
    const isSingle = "key" in params;
    if (isSingle) return await this.answers_.single.retrieve(params.key);
    return await this.answers_.request.retrieve(normalizeRequest(params));
  }

  /**
   * Subscribes to changes in the cached answer to the given query. Single
   * queries deliver a policy; every other shape delivers the matching
   * policies.
   */
  onChange(
    params: RetrieveSingleParams,
    handler: cache.ChangeHandler<Policy>,
  ): destructor.Destructor;
  onChange(
    params: RetrieveMultipleParams,
    handler: cache.ChangeHandler<Policy[]>,
  ): destructor.Destructor;
  onChange(
    params: RetrieveParams,
    handler: cache.ChangeHandler<Policy> | cache.ChangeHandler<Policy[]>,
  ): destructor.Destructor {
    const answers = this.answers_;
    if ("key" in params)
      return answers.single.onChange(
        params.key,
        handler as cache.ChangeHandler<Policy>,
      );
    return answers.request.onChange(
      normalizeRequest(params),
      handler as cache.ChangeHandler<Policy[]>,
    );
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached.
   */
  getCached(params: RetrieveSingleParams): cache.Cached<Policy> | undefined;
  getCached(params: RetrieveMultipleParams): cache.Cached<Policy[]> | undefined;
  getCached(
    params: RetrieveParams,
  ): cache.Cached<Policy> | cache.Cached<Policy[]> | undefined {
    const answers = this.answers_;
    if ("key" in params) return answers.single.getCached(params.key);
    return answers.request.getCached(normalizeRequest(params));
  }

  async delete(key: Key, opts?: cache.WriteOptions): Promise<void>;
  async delete(keys: Key[], opts?: cache.WriteOptions): Promise<void>;
  async delete(keys: Key | Key[], opts: cache.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const ids = ontologyID(keysArr);
    const rollback = new cache.Rollback();
    const writes = this.writes;
    rollback.add(ontology.deleteCachedResources(this.cache_, ids));
    rollback.add(writes.delete(keysArr));
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
    this.writes.delete(keysArr);
    this.relationshipWrites.delete((r) =>
      ids.some((id) => ontology.idsEqual(r.from, id) || ontology.idsEqual(r.to, id)),
    );
  }

  async rename(key: Key, name: string, opts: cache.WriteOptions = {}): Promise<void> {
    const existing = await this.retrieve({ key });
    const rollback = new cache.Rollback();
    const writes = this.writes;
    rollback.add(cache.partialUpdate(writes, key, { name }));
    rollback.add(ontology.renameCachedResource(this.cache_, ontologyID(key), name));
    await opts.onOptimistic?.();
    await rollback.guard(async () => {
      await this.create({ ...existing, name });
    });
  }

  private get writes(): cache.Table<Key, Policy> {
    return this.cache_.table(STORE_KEY);
  }

  private get relationshipWrites(): cache.Table<string, ontology.Relationship> {
    return this.cache_.table(ontology.RELATIONSHIPS_STORE_KEY);
  }

  private get policyStore(): cache.Table<Key, Policy> {
    return this.cache_.table(STORE_KEY);
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
      const cached = this.policyStore.get(key);
      if (cached != null) results.push(cached);
      else misses.push(key);
    }
    if (misses.length > 0) {
      const fetched = await this.execRetrieve({ keys: misses });
      this.policyStore.setMany(fetched);
      results.push(...fetched);
    }
    return cache.orderByKeys(keys, results, (p) => p.key);
  }

  private async fetchSingle(query: Key): Promise<Policy> {
    const cached = this.policyStore.get(query);
    if (cached != null) return cached;
    const policies = await this.execRetrieve({ key: query });
    checkForMultipleOrNoResults("Policy", query, policies, true);
    this.policyStore.setMany(policies);
    return policies[0];
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Policy[]> {
    if (isKeysOnly(query)) return await this.fetchKeys(query.keys as Key[]);
    const policies = await this.execRetrieve(query);
    this.policyStore.setMany(policies);
    return policies;
  }
}
