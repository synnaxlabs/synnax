// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, id } from "@synnaxlabs/x";

import { policy } from "@/access/policy";
import { role } from "@/access/role";
import type Synnax from "@/client";
import { AccessDeniedError, NotFoundError } from "@/errors";
import { createTestClient } from "@/testutil/client";
import { user } from "@/user";

/**
 * Reports whether the error, or any error it wraps, is an access denial.
 * {@link AccessDeniedError.matches} reads only the error it is given, so a wrapped
 * denial hides its type behind the wrapper.
 */
const isDenial = (err: unknown): boolean =>
  AccessDeniedError.matches(err) ||
  (err instanceof Error && err.cause != null && isDenial(err.cause));

/**
 * Connects as a new user, tolerating the change-stream refusal a restricted subject
 * gets. The Core refuses that stream by design; logging the refusal prints after
 * teardown, which vitest counts as an unhandled error. The returned client already
 * holds the subject's policies, so a guard reading them answers on its first render
 * rather than reporting a pending read as a denial.
 */
const connectAs = async (username: string, key: user.Key) => {
  const client = createTestClient({
    username,
    password: "test",
    onInternalError: (err) => {
      if (!isDenial(err)) console.error(err);
    },
  });
  await client.access.policies.retrieveForSubject(user.ontologyID(key));
  return client;
};

const createUser = async (client: Synnax) => {
  const username = id.create();
  const u = await client.users.create({
    username,
    password: "test",
    firstName: "test",
    lastName: "test",
  });
  return { username, key: u.key };
};

/**
 * Creates a client authenticated as a new user holding a role that carries the
 * policies. Pass several to grant different actions on different objects, which one
 * policy cannot express.
 */
export const createTestClientWithPolicy = async (
  client: Synnax,
  pol: policy.New | policy.New[],
) => {
  const u = await createUser(client);
  const policies = await Promise.all(
    array.toArray(pol).map(async (p) => await client.access.policies.create(p)),
  );
  const r = await client.access.roles.create({
    name: "test",
    description: "test",
  });
  await client.ontology.addChildren(
    role.ontologyID(r.key),
    ...policies.map(({ key }) => policy.ontologyID(key)),
  );
  await client.access.roles.assign({ user: u.key, role: r.key });
  return await connectAs(u.username, u.key);
};

/** The roles the Core provisions on startup. */
export const BUILT_IN_ROLES = [
  "Owner",
  "Engineer",
  "Host",
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
  const u = await createUser(client);
  await client.access.roles.assign({ user: u.key, role: r.key });
  return await connectAs(u.username, u.key);
};

/**
 * Caches one client per built-in role, so a spec covering many roles pays the user
 * creation cost once each. Construct one per spec file; the clients close with it.
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
