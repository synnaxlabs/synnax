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
  type Policy,
  policyZ,
} from "@/access/policy/types.gen";
import { ontology } from "@/ontology";
import { query } from "@/query";

export const SET_CHANNEL_NAME = "sy_policy_set";
export const DELETE_CHANNEL_NAME = "sy_policy_delete";

const retrieveRequestZ = z.object({
  keys: keyZ.array().optional(),
  subjects: ontology.idZ.array().optional(),
  limit: z.number().optional(),
  offset: z.number().optional(),
  internal: z.boolean().optional(),
});

const listRetrieveParamsZ = z.union([
  z
    .object({ for: ontology.idZ })
    .transform(({ for: forId }) => ({ subjects: [forId] })),
  z
    .object({ for: ontology.idZ.array() })
    .transform(({ for: forIds }) => ({ subjects: forIds })),
  retrieveRequestZ,
]);

export type RetrieveSingleParams = { key: Key };
export type RetrieveMultipleParams = z.input<typeof listRetrieveParamsZ>;

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

export class Client extends query.Retriever<
  typeof listRetrieveParamsZ,
  Key,
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
    const store = cache.createTable<Key, Policy>({
      name: "policies",
      fetch: async (keys) => await this.execRetrieve({ keys }),
      listen: [
        query.createSetListener(SET_CHANNEL_NAME, policyZ),
        query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ),
      ],
    });
    super(cache, {
      name: "policy",
      table: store,
      request: {
        schema: listRetrieveParamsZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (policy, req) => requestFilter(req)(policy),
        serverFields: SERVER_FIELDS,
      },
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
    const drop = () => [
      ontology.deleteCachedResources(this.ontology, ids),
      this.store.delete(keysArr),
    ];
    const rollback = new destructor.Chain();
    rollback.add(...drop());
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
    drop();
    this.ontology.relationships.delete((r) =>
      ids.some((id) => ontology.idsEqual(r.from, id) || ontology.idsEqual(r.to, id)),
    );
  }

  async rename(key: Key, name: string, opts: query.WriteOptions = {}): Promise<void> {
    const existing = await this.retrieve({ key });
    const rename = () => [
      query.partialUpdate(this.store, key, { name }),
      ontology.renameCachedResource(this.ontology, ontologyID(key), name),
    ];
    const rollback = new destructor.Chain();
    rollback.add(...rename());
    await opts.onOptimistic?.();
    await rollback.guard(async () => {
      await this.create({ ...existing, name });
    });
    rename();
  }

  private async execRetrieve(params: RetrieveRequest): Promise<Policy[]> {
    const res = await this.client.send(
      "/access/policy/retrieve",
      params,
      retrieveRequestZ,
      retrieveResZ,
    );
    return res.policies;
  }
}
