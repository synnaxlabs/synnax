// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, type destructor, primitive, record } from "@synnaxlabs/x";
import { z } from "zod";

import { cache } from "@/cache";
import { MultipleFoundError, NotFoundError } from "@/errors";
import { ontology } from "@/ontology";
import { type Key, keyZ, type New, newZ, type User, userZ } from "@/user/types.gen";

const retrieveRequestZ = z.object({
  keys: keyZ.array().optional(),
  usernames: z.string().array().optional(),
});

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

const usernamesRetrieveRequestZ = z
  .object({
    usernames: z.string().array(),
  })
  .transform(({ usernames }) => ({ usernames }));

export type KeyRetrieveRequest = z.input<typeof keyRetrieveRequestZ>;
export type UsernameRetrieveRequest = z.input<typeof usernameRetrieveRequestZ>;
export type UsernamesRetrieveRequest = z.input<typeof usernamesRetrieveRequestZ>;

const retrieveParamsZ = z.union([
  keyRetrieveRequestZ,
  usernameRetrieveRequestZ,
  usernamesRetrieveRequestZ,
  retrieveRequestZ,
]);

export type RetrieveParams = z.input<typeof retrieveParamsZ>;

export interface RetrieveRequest extends z.infer<typeof retrieveRequestZ> {}

const retrieveResZ = z.object({ users: userZ.array().default(() => []) });

const createReqZ = z.object({ users: newZ.array() });
const createResZ = z.object({ users: userZ.array() });
const changeUsernameReqZ = z.object({ key: keyZ, username: z.string().min(1) });
const changeUsernameResZ = z.object({});
const renameReqZ = z.object({
  key: keyZ,
  firstName: z.string().optional(),
  lastName: z.string().optional(),
});
const renameResZ = z.object({});
const deleteReqZ = z.object({ keys: keyZ.array() });
const deleteResZ = z.object({});

export const SET_CHANNEL_NAME = "sy_user_set";
export const DELETE_CHANNEL_NAME = "sy_user_delete";

type SingleParams = KeyRetrieveRequest | UsernameRetrieveRequest;

const isSingleParams = (params: RetrieveParams): params is SingleParams =>
  "key" in params || "username" in params;

const singleIdentifier = (params: SingleParams): string =>
  "key" in params ? `key ${params.key}` : `username ${params.username}`;

const normalizeRequest = (
  params: UsernamesRetrieveRequest | RetrieveRequest,
): RetrieveRequest =>
  "usernames" in params && !("keys" in params)
    ? { usernames: params.usernames }
    : params;

const isKeysOnly = (req: RetrieveRequest): boolean =>
  primitive.isNonZero(req.keys) && req.usernames == null;

const requestFilter = (req: RetrieveRequest): ((u: User) => boolean) => {
  const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
  const usernameSet = primitive.isNonZero(req.usernames)
    ? new Set(req.usernames)
    : undefined;
  return (u) => {
    if (keySet != null && !keySet.has(u.key)) return false;
    if (usernameSet != null && !usernameSet.has(u.username)) return false;
    return true;
  };
};

export class Client {
  private readonly client: UnaryClient;
  private readonly store: cache.Table<Key, User>;
  private readonly ontology: ontology.Stores;
  private readonly answers: {
    single: cache.Answers<SingleParams, User, Key, User>;
    request: cache.Answers<RetrieveRequest, User[], Key, User>;
  };

  constructor(
    client: UnaryClient,
    engine: cache.Cache,
    ontologyStores: ontology.Stores,
  ) {
    this.client = client;
    this.store = engine.createTable<Key, User>({ name: "users" });
    this.ontology = ontologyStores;
    this.answers = {
      single: engine.answers({
        name: "user",
        table: this.store,
        fetch: async (query) => [await this.fetchSingle(query)].map((u) => u.key),
        compose: (records) => records[0],
        keyOf: (query) => ("key" in query ? query.key : null),
        matches: (u, query) =>
          "key" in query ? u.key === query.key : u.username === query.username,
        single: true,
      }),
      request: engine.answers({
        name: "users",
        table: this.store,
        fetch: async (query) => (await this.fetchRequest(query)).map((u) => u.key),
        compose: (records) => records,
        matches: (u, query) => requestFilter(query)(u),
      }),
    };
  }

  async create(user: New): Promise<User>;
  async create(users: New[]): Promise<User[]>;
  async create(users: New | New[]): Promise<User | User[]> {
    const isMany = Array.isArray(users);
    const res = await this.client.send(
      "/user/create",
      { users: array.toArray(users) },
      createReqZ,
      createResZ,
    );
    this.store.set(res.users);
    return isMany ? res.users : res.users[0];
  }

  async changeUsername(key: Key, newUsername: string): Promise<void> {
    await this.client.send(
      "/user/change-username",
      { key, username: newUsername },
      changeUsernameReqZ,
      changeUsernameResZ,
    );
    this.mergeThrough(key, { username: newUsername });
    ontology.renameCachedResource(this.ontology, ontologyID(key), newUsername);
  }

  async retrieve(params: KeyRetrieveRequest): Promise<User>;
  async retrieve(params: UsernameRetrieveRequest): Promise<User>;
  async retrieve(params: RetrieveParams): Promise<User[]>;
  async retrieve(params: RetrieveParams): Promise<User | User[]> {
    if (isSingleParams(params)) return await this.answers.single.retrieve(params);
    return await this.answers.request.retrieve(normalizeRequest(params));
  }

  /**
   * Subscribes to changes in the cached answer to the given query. Single
   * queries deliver a user; every other shape delivers the matching users.
   */
  onChange(
    params: SingleParams,
    handler: cache.ChangeHandler<User>,
  ): destructor.Destructor;
  onChange(
    params: UsernamesRetrieveRequest | RetrieveRequest,
    handler: cache.ChangeHandler<User[]>,
  ): destructor.Destructor;
  onChange(
    params: RetrieveParams,
    handler: cache.ChangeHandler<User> | cache.ChangeHandler<User[]>,
  ): destructor.Destructor {
    const { request, single } = this.answers;
    if (isSingleParams(params))
      return single.onChange(params, handler as cache.ChangeHandler<User>);
    return request.onChange(
      normalizeRequest(params),
      handler as cache.ChangeHandler<User[]>,
    );
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached.
   */
  getCached(params: SingleParams): cache.Cached<User> | undefined;
  getCached(
    params: UsernamesRetrieveRequest | RetrieveRequest,
  ): cache.Cached<User[]> | undefined;
  getCached(
    params: RetrieveParams,
  ): cache.Cached<User> | cache.Cached<User[]> | undefined {
    const { request, single } = this.answers;
    if (isSingleParams(params)) return single.getCached(params);
    return request.getCached(normalizeRequest(params));
  }

  async rename(key: Key, firstName?: string, lastName?: string): Promise<void> {
    await this.client.send(
      "/user/rename",
      { key, firstName, lastName },
      renameReqZ,
      renameResZ,
    );
    this.mergeThrough(key, { firstName, lastName });
  }

  async delete(key: Key, opts?: cache.WriteOptions): Promise<void>;
  async delete(keys: Key[], opts?: cache.WriteOptions): Promise<void>;
  async delete(keys: Key | Key[], opts: cache.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const rollback = new cache.Rollback();
    rollback.add(ontology.deleteCachedResources(this.ontology, ontologyID(keysArr)));
    await opts.onOptimistic?.();
    await rollback.guard(
      async () =>
        await this.client.send(
          "/user/delete",
          { keys: keysArr },
          deleteReqZ,
          deleteResZ,
        ),
    );
    this.store.delete(keysArr);
  }

  // Undefined fields are dropped: the server keeps prior values for them.
  private mergeThrough(key: Key, changes: Partial<User>): void {
    const prev = this.store.get(key);
    if (prev != null)
      this.store.set(key, { ...prev, ...record.purgeUndefined(changes) });
  }

  private async execRetrieve(params: RetrieveParams): Promise<User[]> {
    const res = await this.client.send(
      "/user/retrieve",
      params,
      retrieveParamsZ,
      retrieveResZ,
    );
    return res.users;
  }

  /**
   * Fetches the given keys, serving cached entries and fetching only the
   * misses. Preserves the caller's key order.
   */
  private async fetchKeys(keys: Key[]): Promise<User[]> {
    const results: User[] = [];
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
    return cache.orderByKeys(keys, results, (u) => u.key);
  }

  private async fetchSingle(query: SingleParams): Promise<User> {
    if ("key" in query) {
      const cached = this.store.get(query.key);
      if (cached != null) return cached;
    } else {
      const [cached] = this.store.get((u) => u.username === query.username);
      if (cached != null) return cached;
    }
    const users = await this.execRetrieve(query);
    checkSingle(query, users);
    this.store.set(users);
    return users[0];
  }

  private async fetchRequest(query: RetrieveRequest): Promise<User[]> {
    if (isKeysOnly(query)) return await this.fetchKeys(query.keys as Key[]);
    const users = await this.execRetrieve(query);
    this.store.set(users);
    return users;
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
