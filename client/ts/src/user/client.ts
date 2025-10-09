// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { array } from "@synnaxlabs/x";
import { z } from "zod";

import { MultipleFoundError, NotFoundError } from "@/errors";
import { type ontology } from "@/ontology";
import { type Key, keyZ, type New, newZ, type User, userZ } from "@/user/payload";

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

const retrieveArgsZ = z.union([
  keyRetrieveRequestZ,
  usernameRetrieveRequestZ,
  usernamesRetrieveRequestZ,
  retrieveRequestZ,
]);

export type RetrieveArgs = z.input<typeof retrieveArgsZ>;

export interface RetrieveRequest extends z.infer<typeof retrieveRequestZ> {}

const retrieveResZ = z.object({ users: array.nullableZ(userZ) });

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

const RETRIEVE_ENDPOINT = "/user/retrieve";
const CREATE_ENDPOINT = "/user/create";
const CHANGE_USERNAME_ENDPOINT = "/user/change-username";
const RENAME_ENDPOINT = "/user/rename";
const DELETE_ENDPOINT = "/user/delete";

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(user: New): Promise<User>;
  async create(users: New[]): Promise<User[]>;
  async create(users: New | New[]): Promise<User | User[]> {
    const isMany = Array.isArray(users);
    const res = await sendRequired<typeof createReqZ, typeof createResZ>(
      this.client,
      CREATE_ENDPOINT,
      { users: array.toArray(users) },
      createReqZ,
      createResZ,
    );
    return isMany ? res.users : res.users[0];
  }

  async changeUsername(key: Key, newUsername: string): Promise<void> {
    await sendRequired<typeof changeUsernameReqZ, typeof changeUsernameResZ>(
      this.client,
      CHANGE_USERNAME_ENDPOINT,
      { key, username: newUsername },
      changeUsernameReqZ,
      changeUsernameResZ,
    );
  }

  async retrieve(args: KeyRetrieveRequest): Promise<User>;
  async retrieve(args: UsernameRetrieveRequest): Promise<User>;
  async retrieve(args: RetrieveArgs): Promise<User[]>;
  async retrieve(args: RetrieveArgs): Promise<User | User[]> {
    const isSingle = "key" in args || "username" in args;
    const res = await sendRequired<typeof retrieveArgsZ, typeof retrieveResZ>(
      this.client,
      RETRIEVE_ENDPOINT,
      args,
      retrieveArgsZ,
      retrieveResZ,
    );

    if (!isSingle) return res.users;

    if (res.users.length === 0) {
      const identifier =
        "key" in args ? `key ${args.key}` : `username ${args.username}`;
      throw new NotFoundError(`No user with ${identifier} found`);
    }
    if (res.users.length > 1) {
      const identifier =
        "key" in args ? `key ${args.key}` : `username ${args.username}`;
      throw new MultipleFoundError(`Multiple users found with ${identifier}`);
    }
    return res.users[0];
  }

  async rename(key: Key, firstName?: string, lastName?: string): Promise<void> {
    await sendRequired<typeof renameReqZ, typeof renameResZ>(
      this.client,
      RENAME_ENDPOINT,
      { key, firstName, lastName },
      renameReqZ,
      renameResZ,
    );
  }

  async delete(key: Key): Promise<void>;
  async delete(keys: Key[]): Promise<void>;
  async delete(keys: Key | Key[]): Promise<void> {
    await sendRequired<typeof deleteReqZ, typeof deleteResZ>(
      this.client,
      DELETE_ENDPOINT,
      { keys: array.toArray(keys) },
      deleteReqZ,
      deleteResZ,
    );
  }
}

export const ontologyID = (key: Key): ontology.ID => ({ type: "user", key });
