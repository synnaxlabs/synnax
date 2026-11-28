// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, id } from "@synnaxlabs/x";

import type { policy } from "@/access/policy";
import type { role } from "@/access/role";
import type Synnax from "@/client";
import type { SynnaxParams } from "@/client";
import { NotFoundError } from "@/errors";
import { createFileTestClient, createTestClient } from "@/testutil/client";
import { createTestUser, type TestUser } from "@/testutil/user";
import { user } from "@/user";

/**
 * Creates a client that logs in as an existing user. It tolerates denials, as
 * {@link createTestClient} does.
 */
export const createTestClientAs = (username: string, password: string): Synnax =>
  createTestClient({ username, password });

/**
 * Connects as a new user and primes its policy cache. Unless onInternalError is given,
 * it tolerates denials, as {@link createTestClient} does.
 */
const connectAs = async (
  { user: { key }, username, password }: TestUser,
  onInternalError?: (err: Error) => void,
  create = createTestClient,
) => {
  const client = create({ username, password, onInternalError });
  await client.access.policies.retrieve({ subjects: [user.resourceID(key)] });
  return client;
};

/** Creates a role with no policies, for a user that can do nothing. */
export const createEmptyRole = async (client: Synnax): Promise<role.Role> =>
  await client.access.roles.create({ name: id.create(), description: "test" });

/** A policy for the role that a test client's user holds. */
export type TestPolicy = Omit<policy.New, "role">;

/**
 * Creates a client authenticated as a new user holding a role that carries the
 * policies. Pass several to grant different actions on different objects, which one
 * policy cannot express.
 * @param onInternalError - Receives every internal error of the new client, denials
 * included. Omit it to tolerate denials.
 * @param params - Further parameters of the new client.
 */
export const createTestClientWithPolicy = async (
  client: Synnax,
  pol: TestPolicy | TestPolicy[],
  onInternalError?: (err: Error) => void,
  params?: Partial<SynnaxParams>,
) => {
  const r = await createEmptyRole(client);
  await client.access.policies.create(
    array.toArray(pol).map((p) => ({ ...p, role: r.key })),
  );
  const u = await createTestUser(client, { role: r.key });
  return await connectAs(u, onInternalError ?? params?.onInternalError, (p) =>
    createTestClient({ ...params, ...p }),
  );
};

/** The roles the Core provisions on startup. */
export const BUILT_IN_ROLES = [
  "Owner",
  "Engineer",
  "Driver",
  "Operator",
  "Viewer",
] as const;

export type BuiltInRole = (typeof BUILT_IN_ROLES)[number];

/**
 * Creates a client authenticated as a new user holding one of the Core's built-in
 * roles, for exercising a real role's permission set instead of a synthetic policy.
 * @param name - The built-in role to assign.
 * @throws {NotFoundError} if the Core provisions no role under the name.
 */
export const createTestClientWithRole = async (
  client: Synnax,
  name: BuiltInRole,
): Promise<Synnax> => await connectAsRole(client, name, createTestClient);

const connectAsRole = async (
  client: Synnax,
  name: BuiltInRole,
  create: typeof createTestClient,
): Promise<Synnax> => {
  const roles = await client.access.roles.retrieve({});
  const r = roles.find((r) => r.name === name);
  if (r == null) throw new NotFoundError(`no built-in role named ${name}`);
  const u = await createTestUser(client, { role: r.key });
  return await connectAs(u, undefined, create);
};

/**
 * Caches one client per built-in role, so a spec covering many roles pays the user
 * creation cost once each. Construct one per spec file.
 */
export class RoleClients {
  private readonly client: Synnax;
  private readonly byRole = new Map<BuiltInRole, Promise<Synnax>>();

  constructor(client: Synnax) {
    this.client = client;
  }

  /** Returns a client authenticated as a user holding the role. */
  async get(name: BuiltInRole): Promise<Synnax> {
    const existing = this.byRole.get(name);
    if (existing != null) return await existing;
    // The cache outlives the test that fills it, so its clients do too.
    const created = connectAsRole(this.client, name, createFileTestClient);
    this.byRole.set(name, created);
    return await created;
  }
}
