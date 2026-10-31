// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, type observe, primitive } from "@synnaxlabs/x";
import { z } from "zod";

import { type connection } from "@/connection";
import { MultipleFoundError, NotFoundError } from "@/errors";
import { ontology } from "@/ontology";
import { query } from "@/query";
import { Current } from "@/user/current";
import { type Key, keyZ, type New, newZ, type User, userZ } from "@/user/types.gen";

const retrieveRequestZ = z.object({
  keys: keyZ.array().optional(),
  usernames: z.string().array().optional(),
});
const retrieveMultiParamsZ = retrieveRequestZ.or(query.keyListZ(keyZ));

const keyRetrieveRequestZ = z
  .object({
    key: keyZ,
  })
  .transform(({ key }) => ({ keys: [key] }));

const usernameRetrieveRequestZ = z
  .object({
    username: z.string(),
  })
  .transform(({ username }) => ({ usernames: [username] }));

export type KeyRetrieveRequest = z.input<typeof keyRetrieveRequestZ>;
export type UsernameRetrieveRequest = z.input<typeof usernameRetrieveRequestZ>;
export type UsernamesRetrieveRequest = { usernames: string[] };

const retrieveParamsZ = z.union([
  keyRetrieveRequestZ,
  usernameRetrieveRequestZ,
  retrieveRequestZ,
]);

export type RetrieveParams = z.input<typeof retrieveParamsZ>;

export interface RetrieveRequest extends z.infer<typeof retrieveRequestZ> {}

const retrieveResZ = z.object({ users: userZ.array().default(() => []) });

const createReqZ = z.object({ users: newZ.array() });
const createResZ = z.object({ users: userZ.array() });
const changeUsernameReqZ = z.object({ key: keyZ, username: z.string().min(1) });
const changeUsernameResZ = z.object({});
const renameReqZ = z.object({ key: keyZ, name: z.string() });
const renameResZ = z.object({});
const deleteReqZ = z.object({ keys: keyZ.array() });
const deleteResZ = z.object({});

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

const requestFilter = (req: RetrieveRequest): ((u: User) => boolean) => {
  const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
  return (u) => keySet == null || keySet.has(u.key);
};

// The username lives on the user's password, so only the Core can match it.
const SERVER_FIELDS = ["searchTerm", "limit", "offset", "usernames"] as const;

export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
  ontology: ontology.Client;
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
  private readonly cfg: ClientConfig;
  private readonly store: query.Table<Key, User>;

  constructor(cfg: ClientConfig) {
    const { cache } = cfg;
    const store = cache.createTable<Key, User>({
      name: "users",
      fetch: async (keys) => await this.execRetrieve({ keys }),
      listen: [
        query.createSetListener(SET_CHANNEL_NAME, userZ),
        query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ),
      ],
    });
    const single = cache.queries<SingleParams, User, Key, User>({
      name: "user",
      table: store,
      fetch: async (params) => {
        const users = await this.execRetrieve(params);
        checkSingle(params, users);
        store.ingest(users);
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
        matches: (u, req) => requestFilter(req)(u),
        serverFields: SERVER_FIELDS,
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
    await this.cfg.unary.send(
      "/user/change-username",
      { key, username: newUsername },
      changeUsernameReqZ,
      changeUsernameResZ,
    );
  }

  async rename(key: Key, name: string, opts: query.WriteOptions = {}): Promise<void> {
    await query.optimistic({
      rollbacks: [
        query.partialUpdate(this.store, key, { name }),
        this.cfg.ontology.cache.renameResource(ontologyID(key), name),
      ],
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/user/rename",
          { key, name },
          renameReqZ,
          renameResZ,
        ),
    });
  }

  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const drop = () => [
      this.cfg.ontology.cache.deleteResources(ontologyID(keysArr)),
      this.store.delete(keysArr),
    ];
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
      retrieveResZ,
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

export const ontologyID = ontology.createIDFactory<Key>("user");
export const TYPE_ONTOLOGY_ID = ontologyID("");
