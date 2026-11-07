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
import { z } from "zod";

import {
  type Key,
  keyZ,
  type New,
  type Policy,
  policyZ,
  type RetrieveRequest,
  retrieveRequestZ,
  retrieveResponseZ,
  type Where,
} from "@/access/policy/types.gen";
import {
  type Lines,
  matches,
  retrieveInputZ,
  serverComputed,
} from "@/access/policy/where.gen";
import { query } from "@/query";
import { user } from "@/user";

export const SET_CHANNEL_NAME = "sy_policy_set";
export const DELETE_CHANNEL_NAME = "sy_policy_delete";

const retrieveMultiParamsZ = z.union([
  query.keysRequestZ(keyZ),
  retrieveRequestZ,
  retrieveInputZ(retrieveRequestZ.omit({ where: true }).shape),
]);

export type RetrieveSingleParams = { key: Key };
export type RetrieveMultipleParams = query.RequestInput<typeof retrieveMultiParamsZ>;

// The roles of a subject live on the subject, so only the Core can check subjects.
const LINES: Lines = {
  subjects: null,
  roles: ({ role }, roles) => roles.includes(role),
};

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

export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
  /** Holds the roles of users, so a role change refetches the policies of a user. */
  users: user.Client;
}

/** @returns whether any subjects line of where names the user with the given key. */
const namesUser = (where: Where | undefined, key: user.Key): boolean =>
  where != null &&
  (where.subjects?.some(
    ({ type, key: k }) => type === user.TYPE_RESOURCE_ID.type && k === key,
  ) === true ||
    where.and?.some((w) => namesUser(w, key)) === true ||
    where.or?.some((w) => namesUser(w, key)) === true ||
    namesUser(where.not, key));

export class Client extends query.Retriever<typeof retrieveMultiParamsZ, Key, Policy> {
  private readonly cfg: ClientConfig;
  private readonly store: query.Table<Key, Policy>;

  constructor(cfg: ClientConfig) {
    const { cache } = cfg;
    const store = cache.createTable<Key, Policy>({
      name: "policies",
      fetch: async (keys) => await this.execRetrieve(query.keysRequest(keys)),
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
        matches: (policy, req) => matches(LINES, policy, req.where),
        serverComputed: (req) => serverComputed(LINES, req.where),
        watch: [
          query.watch(cfg.users.store, ({ key }, req: RetrieveRequest) =>
            namesUser(req.where, key) ? "refetch" : null,
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
      retrieveResponseZ,
    );
    return res.policies;
  }
}
