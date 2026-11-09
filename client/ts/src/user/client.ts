// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, filter, type observe } from "@synnaxlabs/x";
import { z } from "zod";

import { type role } from "@/access/role";
import { type batch } from "@/batch";
import { type connection } from "@/connection";
import { MultipleFoundError, NotFoundError } from "@/errors";
import { query } from "@/query";
import { resource } from "@/resource";
import {
  type Action,
  dispatchReqZ,
  dispatchResZ,
  set,
  setUsername,
} from "@/user/actions.gen";
import { Current } from "@/user/current";
import {
  type Key,
  keyZ,
  type New,
  newZ,
  type RetrieveRequest,
  retrieveRequestZ,
  retrieveResponseZ,
  type User,
  userZ,
} from "@/user/types.gen";
import { type Lines, matches, retrieveInputZ, serverComputed } from "@/user/where.gen";

const retrieveMultiParamsZ = z.union([
  query.keysRequestZ(keyZ),
  filter.requestInputZ(
    retrieveRequestZ,
    retrieveInputZ(retrieveRequestZ.omit({ where: true }).shape),
  ),
]);

const keyRetrieveRequestZ = z
  .object({ key: keyZ })
  .transform(({ key }): RetrieveRequest => ({ where: { key: { eq: key } } }));

const usernameRetrieveRequestZ = z
  .object({ username: z.string() })
  .transform(({ username }): RetrieveRequest => ({
    where: { usernames: [username] },
  }));

export type KeyRetrieveRequest = z.input<typeof keyRetrieveRequestZ>;
export type UsernameRetrieveRequest = z.input<typeof usernameRetrieveRequestZ>;

const retrieveParamsZ = z.union([
  keyRetrieveRequestZ,
  usernameRetrieveRequestZ,
  retrieveRequestZ,
]);

export type RetrieveParams =
  KeyRetrieveRequest | UsernameRetrieveRequest | RetrieveRequest;

const createReqZ = z.object({ users: newZ.array() });
const createResZ = z.object({ users: userZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const deleteResZ = z.object({});

// The username lives on the user's password, so only the Core can match it.
const LINES: Lines = {
  usernames: null,
  roles: (u, roles) => roles.includes(u.role),
};

export const SET_CHANNEL_NAME = "sy_user_set";
export const DELETE_CHANNEL_NAME = "sy_user_delete";

type SingleParams = KeyRetrieveRequest | UsernameRetrieveRequest;

const singleParamsZ = z.union([
  z.strictObject({ key: keyZ }),
  z.strictObject({ username: z.string() }),
  keyZ.transform((key) => ({ key })),
]);

const singleIdentifier = (params: SingleParams): string =>
  "key" in params ? `key ${params.key}` : `username ${params.username}`;

export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
  batch: batch.Client;
  connection: connection.Client;
  /** Carries the user after each login, and undefined when the credentials change. */
  identity: observe.Observable<User | undefined>;
}

export class Client extends query.Retriever<
  typeof retrieveMultiParamsZ,
  Key,
  User,
  User,
  SingleParams,
  SingleParams
> {
  /** The signed-in user. */
  readonly current: Current;
  /** The users that the client holds, by key. */
  readonly store: query.Table<Key, User>;
  private readonly cfg: ClientConfig;

  constructor(cfg: ClientConfig) {
    const { cache } = cfg;
    const store = cache.createTable<Key, User>({
      name: "users",
      fetch: async (keys) => await this.execRetrieve(query.keysRequest(keys)),
      listen: [
        query.createSetListener(SET_CHANNEL_NAME, userZ),
        query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ),
      ],
    });
    const single = cache.queries<SingleParams, User, Key, User>({
      name: "user",
      table: store,
      fetch: async (params) => {
        // A user written after since keeps that write, since the fetch predates it.
        const since = store.stamp();
        const users = await this.execRetrieve(params);
        checkSingle(params, users);
        store.ingest(users, { since });
        return users.map((u) => u.key);
      },
      compose: (records) => records[0],
      keyOf: (params) => ("key" in params ? params.key : null),
      single: true,
    });
    super(cache, {
      name: "user",
      table: store,
      request: {
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (u, req) => matches(LINES, u, req.where),
        serverComputed: (req) => serverComputed(LINES, req.where),
      },
      single: { schema: singleParamsZ, space: single },
    });
    this.cfg = cfg;
    this.store = store;
    this.current = new Current({
      users: this,
      table: store,
      identity: cfg.identity,
      connection: cfg.connection,
    });
  }

  async create(user: New): Promise<User>;
  async create(users: New[]): Promise<User[]>;
  async create(users: New | New[]): Promise<User | User[]> {
    const isMany = Array.isArray(users);
    const res = await this.cfg.unary.send(
      "/user/create",
      { users: array.toArray(users) },
      createReqZ,
      createResZ,
    );
    this.store.set(res.users);
    return isMany ? res.users : res.users[0];
  }

  async changeUsername(key: Key, newUsername: string): Promise<void> {
    await this.dispatch(key, setUsername({ username: newUsername }));
  }

  async rename(key: Key, name: string, opts: query.WriteOptions = {}): Promise<void> {
    await query.optimistic({
      rollbacks: [query.partialUpdate(this.store, key, { name })],
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.dispatch(key, set({ path: ["name"], value: name })),
    });
  }

  /**
   * Gives the role to the users with the given keys in one transaction.
   * @throws {ValidationError} if a user is a root user or the role does not exist.
   * @throws {NotFoundError} if a user does not exist.
   * @throws {AccessDeniedError} if the caller cannot update the users, the role, or the
   * roles the users hold now.
   */
  async setRole(
    keys: Key | Key[],
    role: role.Key,
    opts: query.WriteOptions = {},
  ): Promise<void> {
    const keysArr = array.toArray(keys);
    const action = set({ path: ["role"], value: role });
    const requests = keysArr.map((key) => ({
      endpoint: "/user/dispatch",
      payload: { key, actions: [action] },
    }));
    await query.optimistic({
      rollbacks: keysArr.map((key) => query.partialUpdate(this.store, key, { role })),
      onOptimistic: opts.onOptimistic,
      // A batch prefixes its errors with the request index, so one user skips it.
      commit: async () =>
        keysArr.length === 1
          ? await this.dispatch(keysArr[0], action)
          : await this.cfg.batch.send(requests),
    });
  }

  /**
   * Applies the actions to the user in one transaction.
   * @throws {ValidationError} if the user cannot hold an action.
   * @throws {NotFoundError} if the user does not exist.
   * @throws {AccessDeniedError} if the caller cannot update the user.
   */
  async dispatch(key: Key, actions: Action | Action[]): Promise<void> {
    await this.cfg.unary.send(
      "/user/dispatch",
      { key, dispatchKey: "", actions: array.toArray(actions) },
      dispatchReqZ,
      dispatchResZ,
    );
  }

  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const drop = () => [this.store.delete(keysArr)];
    await query.optimistic({
      rollbacks: drop(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/user/delete",
          { keys: keysArr },
          deleteReqZ,
          deleteResZ,
        ),
    });
    drop();
  }

  private async execRetrieve(params: RetrieveParams): Promise<User[]> {
    const res = await this.cfg.unary.send(
      "/user/retrieve",
      params,
      retrieveParamsZ,
      retrieveResponseZ,
    );
    return res.users;
  }
}

const checkSingle = (params: SingleParams, users: User[]): void => {
  if (users.length === 0)
    throw new NotFoundError(`No user with ${singleIdentifier(params)} found`);
  if (users.length > 1)
    throw new MultipleFoundError(
      `Multiple users found with ${singleIdentifier(params)}`,
    );
};

export const resourceID = resource.createIDFactory<Key>("user");
export const TYPE_RESOURCE_ID = resourceID("");
