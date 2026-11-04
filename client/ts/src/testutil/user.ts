// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";

import type Synnax from "@/client";
import { user } from "@/user";

export interface CreateTestUserParams {
  /** Defaults to a random ID. */
  username?: string;
  /** Defaults to "test". */
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
    username = id.create(),
    password = "test",
    name = username,
  }: CreateTestUserParams = {},
): Promise<TestUser> => {
  const u = await client.users.create({ name });
  await client.auth.createPassword({
    subject: user.ontologyID(u.key),
    username,
    password,
  });
  return { user: u, username, password };
};
