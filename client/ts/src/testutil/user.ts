// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";

import type { role } from "@/access/role";
import type Synnax from "@/client";
import { user } from "@/user";

/** The password of every test user that sets none of its own. */
export const TEST_USER_PASSWORD = "test";

export interface CreateTestUserParams {
  /** The role the user holds. */
  role: role.Key;
  /** Defaults to a random ID. */
  username?: string;
  /** Defaults to {@link TEST_USER_PASSWORD}. */
  password?: string;
  /** Defaults to the username. */
  name?: string;
}

export interface TestUser {
  user: user.User;
  username: string;
  password: string;
}

/** Creates a user that can log in with the returned username and password. */
export const createTestUser = async (
  client: Synnax,
  {
    role,
    username = id.create(),
    password = TEST_USER_PASSWORD,
    name = username,
  }: CreateTestUserParams,
): Promise<TestUser> => {
  const u = await client.users.create({ name, role });
  await client.auth.passwords.create({
    subject: user.resourceID(u.key),
    username,
    password,
  });
  return { user: u, username, password };
};
