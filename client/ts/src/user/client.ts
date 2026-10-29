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
import { ontology } from "@/ontology";
import { query } from "@/query";
import { Current } from "@/user/current";
import { type Key, keyZ, type User, userZ } from "@/user/types.gen";

const retrieveRequestZ = z.object({ keys: keyZ.array().optional() });
const retrieveMultiParamsZ = retrieveRequestZ.or(query.keyListZ(keyZ));

const keyRetrieveRequestZ = z
  .object({
    key: keyZ,
  })
  .transform(({ key }) => ({ keys: [key] }));

export type KeyRetrieveRequest = z.input<typeof keyRetrieveRequestZ>;

const retrieveParamsZ = z.union([keyRetrieveRequestZ, retrieveRequestZ]);

export type RetrieveParams = z.input<typeof retrieveParamsZ>;

export interface RetrieveRequest extends z.infer<typeof retrieveRequestZ> {}

const retrieveResZ = z.object({ users: userZ.array().default(() => []) });

const createReqZ = z.object({ users: userZ.array() });
const createResZ = z.object({ users: userZ.array() });
const renameReqZ = z.object({ key: keyZ, name: z.string() });
const renameResZ = z.object({});
const deleteReqZ = z.object({ keys: keyZ.array() });
const deleteResZ = z.object({});

export const SET_CHANNEL_NAME = "sy_user_set";
export const DELETE_CHANNEL_NAME = "sy_user_delete";

const requestFilter = (req: RetrieveRequest): ((u: User) => boolean) => {
  const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
  return (u) => keySet == null || keySet.has(u.key);
};

export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
  ontology: ontology.Client;
  connection: connection.Client;
  /** Carries the user after each login, and undefined when the credentials change. */
  identity: observe.Observable<User | undefined>;
}

export class Client extends query.Retriever<typeof retrieveMultiParamsZ, Key, User> {
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
    super(cache, {
      name: "user",
      table: store,
      request: {
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (u, req) => requestFilter(req)(u),
      },
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

  async create(user: z.input<typeof userZ>): Promise<User>;
  async create(users: z.input<typeof userZ>[]): Promise<User[]>;
  async create(
    users: z.input<typeof userZ> | z.input<typeof userZ>[],
  ): Promise<User | User[]> {
    const isMany = Array.isArray(users);
    // Parsed once so a retried request keeps the keys it generated.
    const req = createReqZ.parse({ users: array.toArray(users) });
    const res = await this.cfg.unary.send("/user/create", req, createReqZ, createResZ);
    this.store.set(res.users);
    return isMany ? res.users : res.users[0];
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

export const ontologyID = ontology.createIDFactory<Key>("user");
export const TYPE_ONTOLOGY_ID = ontologyID("");
