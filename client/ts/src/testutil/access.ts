// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";

import { policy } from "@/access/policy";
import { role } from "@/access/role";
import type Synnax from "@/client";
import { createTestClient } from "@/testutil/client";
import { user } from "@/user";

export const createTestClientWithPolicy = async (client: Synnax, pol: policy.New) => {
  const username = id.create();
  const u = await client.users.create({
    username,
    password: "test",
    firstName: "test",
    lastName: "test",
  });
  const p = await client.access.policies.create(pol);
  const r = await client.access.roles.create({
    name: "test",
    description: "test",
  });
  await client.ontology.addChildren(role.ontologyID(r.key), policy.ontologyID(p.key));
  await client.access.roles.assign({ user: user.ontologyID(u.key), role: r.key });
  const userClient = createTestClient({ username, password: "test" });
  return userClient;
};
