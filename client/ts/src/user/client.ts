// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { toArray } from "@synnaxlabs/x";

import { MultipleFoundError, NotFoundError } from "@/errors";
import { type Key, type NewUser, type User } from "@/user/payload";
import { Retriever } from "@/user/retriever";
import { Writer } from "@/user/writer";

export class Client {
  private readonly reader: Retriever;
  private readonly writer: Writer;

  constructor(client: UnaryClient) {
    this.writer = new Writer(client);
    this.reader = new Retriever(client);
  }

  async create(user: NewUser): Promise<User>;

  async create(users: NewUser[]): Promise<User[]>;

  async create(users: NewUser | NewUser[]): Promise<User | User[]> {
    const isMany = Array.isArray(users);
    const res = await this.writer.create(users);
    return isMany ? res : res[0];
  }

  async changeUsername(key: Key, newUsername: string): Promise<void> {
    await this.writer.changeUsername(key, newUsername);
  }

  async retrieve(key: Key): Promise<User>;

  async retrieve(keys: Key[]): Promise<User[]>;

  async retrieve(keys: Key | Key[]): Promise<User | User[]> {
    const isMany = Array.isArray(keys);
    const res = await this.reader.retrieve({ keys: toArray(keys) });
    if (isMany) return res;
    if (res.length === 0) throw new NotFoundError(`No user with key ${keys} found`);
    if (res.length > 1)
      throw new MultipleFoundError(`Multiple users found with key ${keys}`);
    return res[0];
  }

  async retrieveByName(username: string): Promise<User>;

  async retrieveByName(usernames: string[]): Promise<User[]>;

  async retrieveByName(usernames: string | string[]): Promise<User | User[]> {
    const isMany = Array.isArray(usernames);
    const res = await this.reader.retrieve({ usernames: toArray(usernames) });
    if (isMany) return res;
    if (res.length === 0)
      throw new NotFoundError(`No user with username ${usernames} found`);
    if (res.length > 1)
      throw new MultipleFoundError(`Multiple users found with username ${usernames}`);
    return res[0];
  }

  async changeName(key: Key, firstName?: string, lastName?: string): Promise<void> {
    await this.writer.changeName(key, firstName, lastName);
  }

  async delete(key: Key): Promise<void>;

  async delete(keys: Key[]): Promise<void>;

  async delete(keys: Key | Key[]): Promise<void> {
    await this.writer.delete(keys);
  }
}
