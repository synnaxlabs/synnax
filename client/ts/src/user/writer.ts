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

import { type Key, keyZ, type New, newZ, type User, userZ } from "@/user/payload";

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

const CREATE_ENDPOINT = "/user/create";
const CHANGE_USERNAME_ENDPOINT = "/user/change-username";
const RENAME_ENDPOINT = "/user/rename";
const DELETE_ENDPOINT = "/user/delete";

export class Writer {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(users: New | New[]): Promise<User[]> {
    const res = await sendRequired<typeof createReqZ, typeof createResZ>(
      this.client,
      CREATE_ENDPOINT,
      { users: array.toArray(users) },
      createReqZ,
      createResZ,
    );
    return res.users;
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

  async rename(key: Key, firstName?: string, lastName?: string): Promise<void> {
    await sendRequired<typeof renameReqZ, typeof renameResZ>(
      this.client,
      RENAME_ENDPOINT,
      { key, firstName, lastName },
      renameReqZ,
      renameResZ,
    );
  }

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
