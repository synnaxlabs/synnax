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

import { createOf, reduceAll } from "@/access/role/actions";
import {
  type Action,
  dispatchReqZ,
  dispatchResZ,
  scopedActionZ,
  set,
} from "@/access/role/actions.gen";
import {
  type Key,
  keyZ,
  type New,
  type RetrieveRequest,
  retrieveRequestZ,
  retrieveResponseZ,
  type Role,
  roleZ,
} from "@/access/role/types.gen";
import {
  compareCursors,
  cursor,
  LIST_FIELDS,
  matches,
  retrieveInputZ,
  serverComputed,
  shortcutZ,
} from "@/access/role/where.gen";
import { actions } from "@/actions";
import { query } from "@/query";

export const ACTION_CHANNEL_NAME = "sy_role_action_set";
export const DELETE_CHANNEL_NAME = "sy_role_delete";

const retrieveMultiParamsZ = filter.requestInputZ(
  retrieveRequestZ,
  retrieveInputZ(retrieveRequestZ.omit({ where: true }).shape),
);

const singleCreateParamsZ = roleZ.transform((r) => ({ roles: [r] }));
export type SingleCreateParams = z.input<typeof singleCreateParamsZ>;

export const multipleCreateParamsZ = roleZ.array().transform((roles) => ({ roles }));

export const createParamsZ = z.union([singleCreateParamsZ, multipleCreateParamsZ]);
export type CreateParams = z.input<typeof createParamsZ>;

const createResZ = z.object({ roles: roleZ.array() });

export type RetrieveSingleParams = { key: Key };
export type RetrieveMultipleParams = query.RequestInput<typeof retrieveMultiParamsZ>;
export type RetrieveParams = RetrieveSingleParams | RetrieveMultipleParams;

const deleteResZ = z.object({});

const deleteParamsZ = keyZ
  .transform((key) => ({ keys: [key] }))
  .or(keyZ.array().transform((keys) => ({ keys })));
export type DeleteParams = z.input<typeof deleteParamsZ>;

export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
}

export class Client extends query.Retriever<typeof retrieveMultiParamsZ, Key, Role> {
  private readonly cfg: ClientConfig;
  private readonly store: query.Table<Key, Role>;
  private readonly dispatcher: actions.Controller<Key, Role, Action>;

  constructor(cfg: ClientConfig) {
    const { cache } = cfg;
    const store = cache.createTable<Key, Role>({
      name: "roles",
      fetch: async (keys) => await this.execRetrieve(query.keysRequest(keys)),
      listen: [query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ)],
    });
    // Every write to a role reaches the client as actions.
    const dispatcher = new actions.Controller<Key, Role, Action>({
      store,
      listed: (action) => actions.changesField(action, LIST_FIELDS),
      onError: cache.onError,
      onEpoch: (callback) => cache.onEpoch(callback),
      onStreamDown: (callback) => cache.onStreamDown(callback),
      send: actions.sender(
        cfg.unary,
        "/access/role/dispatch",
        dispatchReqZ,
        dispatchResZ,
      ),
      reduce: reduceAll,
      createOf,
    });
    cache.listen(dispatcher.listener(ACTION_CHANNEL_NAME, scopedActionZ));
    super(cache, {
      name: "role",
      table: store,
      shortcut: shortcutZ,
      request: {
        list: { cursor, compare: compareCursors },
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (role, req) => matches(role, req.where),
        serverComputed: (req) => serverComputed(req.where),
      },
    });
    this.cfg = cfg;
    this.store = store;
    this.dispatcher = dispatcher;
  }

  async create(role: New): Promise<Role>;
  async create(roles: New[]): Promise<Role[]>;
  async create(roles: New | New[]): Promise<Role | Role[]> {
    const isMany = Array.isArray(roles);
    const res = await this.cfg.unary.send(
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
    const drop = () => [this.store.delete(keysArr)];
    await query.optimistic({
      rollbacks: drop(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/access/role/delete",
          params,
          deleteParamsZ,
          deleteResZ,
        ),
    });
    drop();
  }

  async rename(key: Key, name: string): Promise<void> {
    await this.dispatch(key, set({ path: ["name"], value: name }));
  }

  /**
   * Applies the actions to the cached role until the Core echoes them, and sends them
   * to the Core to apply in one transaction. Actions hold the snake_case form that the
   * Core stores. A refused send drops the actions from the cached role.
   * @throws {Error} if the cached role cannot hold an action, before any send.
   * @throws {ValidationError} if the role cannot hold an action.
   * @throws {NotFoundError} if the role does not exist.
   */
  async dispatch(key: Key, actions: Action | Action[]): Promise<void> {
    await this.dispatcher.send(key, array.toArray(actions));
  }

  private async execRetrieve(params: RetrieveRequest): Promise<Role[]> {
    const res = await this.cfg.unary.send(
      "/access/role/retrieve",
      params,
      retrieveRequestZ,
      retrieveResponseZ,
    );
    return res.roles;
  }
}
