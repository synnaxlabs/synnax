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
  type Policy,
  policyZ,
} from "@/access/policy/types.gen";
import { role } from "@/access/role";
import { query } from "@/query";
import { resource } from "@/resource";
import { user } from "@/user";

export const SET_CHANNEL_NAME = "sy_policy_set";
export const DELETE_CHANNEL_NAME = "sy_policy_delete";

const retrieveRequestZ = z.object({
  keys: keyZ.array().optional(),
  subjects: resource.idZ.array().optional(),
  roles: role.keyZ.array().optional(),
  limit: z.number().optional(),
  offset: z.number().optional(),
  internal: z.boolean().optional(),
});

const listRetrieveParamsZ = z.union([
  z
    .object({ for: resource.idZ })
    .transform(({ for: forId }) => ({ subjects: [forId] })),
  z
    .object({ for: resource.idZ.array() })
    .transform(({ for: forIds }) => ({ subjects: forIds })),
  retrieveRequestZ,
]);
const retrieveMultiParamsZ = listRetrieveParamsZ.or(query.keyListZ(keyZ));

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
 * Query fields only the server can evaluate: the roles of a subject live on the
 * subject, so subject membership cannot be checked locally.
 */
const SERVER_FIELDS = ["subjects", "limit", "offset"] as const;

/**
 * Client-side matching for a request: key sets, roles, and the internal flag.
 * Server-computed shapes (subjects, pagination) never reach this filter; they refetch
 * instead.
 */
const requestFilter = (req: RetrieveRequest): ((p: Policy) => boolean) => {
  const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
  const roleSet = primitive.isNonZero(req.roles) ? new Set(req.roles) : undefined;
  return (p) => {
    if (keySet != null && !keySet.has(p.key)) return false;
    if (roleSet != null && !roleSet.has(p.role)) return false;
    if (req.internal != null && p.internal !== req.internal) return false;
    return true;
  };
};

export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
  /** Holds the roles of users, so a role change refetches the policies of a user. */
  users: user.Client;
}

const isForUser = (req: RetrieveRequest, key: user.Key): boolean =>
  req.subjects?.some(
    ({ type, key: k }) => type === user.TYPE_RESOURCE_ID.type && k === key,
  ) === true;

export class Client extends query.Retriever<typeof retrieveMultiParamsZ, Key, Policy> {
  private readonly cfg: ClientConfig;
  private readonly store: query.Table<Key, Policy>;

  constructor(cfg: ClientConfig) {
    const { cache } = cfg;
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
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (policy, req) => requestFilter(req)(policy),
        serverFields: SERVER_FIELDS,
        watch: [
          query.watch(cfg.users.store, ({ key }, req: RetrieveRequest) =>
            isForUser(req, key) ? "refetch" : null,
          ),
        ],
      },
    });
    this.cfg = cfg;
    this.store = store;
  }

  async create(policy: New): Promise<Policy>;
  async create(policies: New[]): Promise<Policy[]>;
  async create(policies: CreateParams): Promise<Policy | Policy[]> {
    const isMany = Array.isArray(policies);
    const res = await this.cfg.unary.send(
      "/access/policy/create",
      policies,
      createParamsZ,
      createResZ,
    );
    this.store.set(res.policies);
    return isMany ? res.policies : res.policies[0];
  }

  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const drop = () => [this.store.delete(keysArr)];
    await query.optimistic({
      rollbacks: drop(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/access/policy/delete",
          { keys: keysArr },
          deleteReqZ,
          deleteResZ,
        ),
    });
    drop();
  }

  async rename(key: Key, name: string, opts: query.WriteOptions = {}): Promise<void> {
    const existing = await this.retrieve(key);
    const rename = () => [query.partialUpdate(this.store, key, { name })];
    await query.optimistic({
      rollbacks: rename(),
      onOptimistic: opts.onOptimistic,
      commit: async () => {
        await this.create({ ...existing, name });
      },
    });
    rename();
  }

  private async execRetrieve(params: RetrieveRequest): Promise<Policy[]> {
    const res = await this.cfg.unary.send(
      "/access/policy/retrieve",
      params,
      retrieveRequestZ,
      retrieveResZ,
    );
    return res.policies;
  }
}
