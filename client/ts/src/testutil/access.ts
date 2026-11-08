// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, errors, id } from "@synnaxlabs/x";

import { type policy } from "@/access/policy";
import { type role } from "@/access/role";
import type Synnax from "@/client";
import { AccessDeniedError, NotFoundError } from "@/errors";
import { createTestClient } from "@/testutil/client";
import { user } from "@/user";

/** Reports whether the error, or any error in its cause chain, is an access denial. */
const isDenial = (err: unknown): boolean =>
  AccessDeniedError.matches(err) ||
  (err instanceof Error && err.cause != null && isDenial(err.cause));

const logAllButDenials = (err: Error): void => {
  if (!isDenial(err)) console.error(err);
};

/**
 * Creates a client that logs in as an existing user. Denials are tolerated: the Core
 * refuses restricted subjects by design, and the change-stream refusal logs at any time,
 * even after teardown, which vitest counts as an unhandled error.
 */
export const createTestClientAs = (username: string, password: string): Synnax =>
  createTestClient({ username, password, onInternalError: logAllButDenials });

/**
 * Connects as a new user and primes its policy cache. Denials are tolerated by default,
 * as in {@link createTestClientAs}.
 */
const connectAs = async (
  username: string,
  key: user.Key,
  onInternalError: (err: Error) => void = logAllButDenials,
) => {
  const client = createTestClient({ username, password: "test", onInternalError });
  try {
    await client.access.policies.retrieve({
      where: { subjects: [user.resourceID(key)] },
    });
  } catch (err) {
    if (!isDenial(err)) throw errors.fromUnknown(err);
  }
  return client;
};

const createUser = async (client: Synnax, roles: role.Key[]) => {
  const username = id.create();
  const u = await client.users.create({
    username,
    password: "test",
    name: "test",
    roles,
  });
  return { username, key: u.key };
};

/** A policy for the role that a test client's user holds. */
export type TestPolicy = Omit<policy.New, "role">;

/**
 * Creates a client authenticated as a new user holding a role that carries the
 * policies. Pass several to grant different actions on different objects, which one
 * policy cannot express.
 * @param onInternalError - Receives every internal error of the new client, denials
 * included. Omit it to tolerate denials.
 */
export const createTestClientWithPolicy = async (
  client: Synnax,
  pol: TestPolicy | TestPolicy[],
  onInternalError?: (err: Error) => void,
) => {
  const r = await client.access.roles.create({
    name: "test",
    description: "test",
  });
  await client.access.policies.create(
    array.toArray(pol).map((p) => ({ ...p, role: r.key })),
  );
  const u = await createUser(client, [r.key]);
  return await connectAs(u.username, u.key, onInternalError);
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
): Promise<Synnax> => {
  const roles = await client.access.roles.retrieve({});
  const r = roles.find((r) => r.name === name);
  if (r == null) throw new NotFoundError(`no built-in role named ${name}`);
  const u = await createUser(client, [r.key]);
  return await connectAs(u.username, u.key);
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
    const created = createTestClientWithRole(this.client, name);
    this.byRole.set(name, created);
    return await created;
  }
}
