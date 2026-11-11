// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, filter } from "@synnaxlabs/x";
import { z } from "zod";

import { createOf, reduceAll } from "@/access/policy/actions";
import {
  type Action,
  dispatchReqZ,
  dispatchResZ,
  scopedActionZ,
  set,
} from "@/access/policy/actions.gen";
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
  compareCursors,
  cursor,
  type Lines,
  LIST_FIELDS,
  matches,
  retrieveInputZ,
  serverComputed,
} from "@/access/policy/where.gen";
import { actions } from "@/actions";
import { query } from "@/query";
import { user } from "@/user";

export const ACTION_CHANNEL_NAME = "sy_policy_action_set";
export const DELETE_CHANNEL_NAME = "sy_policy_delete";

const retrieveMultiParamsZ = z.union([
  query.keysRequestZ(keyZ),
  filter.requestInputZ(
    retrieveRequestZ,
    retrieveInputZ(retrieveRequestZ.omit({ where: true }).shape),
  ),
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
  private readonly dispatcher: actions.Controller<Key, Policy, Action>;

  constructor(cfg: ClientConfig) {
    const { cache } = cfg;
    const store = cache.createTable<Key, Policy>({
      name: "policies",
      fetch: async (keys) => await this.execRetrieve(query.keysRequest(keys)),
      listen: [query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ)],
    });
    // Every write to a policy reaches the client as actions.
    const dispatcher = new actions.Controller<Key, Policy, Action>({
      store,
      listed: (action) => actions.changesField(action, LIST_FIELDS),
      onError: cache.onError,
      onEpoch: (callback) => cache.onEpoch(callback),
      onStreamDown: (callback) => cache.onStreamDown(callback),
      send: actions.sender(
        cfg.unary,
        "/access/policy/dispatch",
        dispatchReqZ,
        dispatchResZ,
      ),
      reduce: reduceAll,
      createOf,
    });
    cache.listen(dispatcher.listener(ACTION_CHANNEL_NAME, scopedActionZ));
    super(cache, {
      name: "policy",
      table: store,
      request: {
        list: { cursor, compare: compareCursors },
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (policy, req) => matches(LINES, policy, req.where),
        serverComputed: (req) => serverComputed(LINES, req.where),
        triggers: [
          query.trigger(cfg.users.store, ({ key }, req: RetrieveRequest) =>
            namesUser(req.where, key) ? "refetch" : null,
          ),
        ],
      },
    });
    this.cfg = cfg;
    this.store = store;
    this.dispatcher = dispatcher;
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

  async rename(key: Key, name: string): Promise<void> {
    await this.dispatch(key, set({ path: ["name"], value: name }));
  }

  /**
   * Applies the actions to the cached policy until the Core echoes them, and sends them
   * to the Core to apply in one transaction. Actions hold the snake_case form that the
   * Core stores. A refused send drops the actions from the cached policy.
   * @throws {Error} if the cached policy cannot hold an action, before any send.
   * @throws {ValidationError} if the policy cannot hold an action.
   * @throws {NotFoundError} if the policy does not exist.
   */
  async dispatch(key: Key, actions: Action | Action[]): Promise<void> {
    await this.dispatcher.send(key, array.toArray(actions));
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
