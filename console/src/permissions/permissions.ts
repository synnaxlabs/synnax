// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type access, type ontology, type Synnax } from "@synnaxlabs/client";
import { deep } from "@synnaxlabs/x";
import { z } from "zod";

const consolePolicyZ = z.enum(["admin", "schematic"]);

export type ConsolePolicy = z.infer<typeof consolePolicyZ>;

export const consolePolicySet = new Set<ConsolePolicy>(consolePolicyZ.options);

type PolicyWithoutSubjectsOrKey = Omit<access.Policy, "subjects" | "key">;

const adminTypes: ontology.ResourceType[] = ["cluster", "node", "policy", "user"];

const adminPolicy: PolicyWithoutSubjectsOrKey = {
  objects: adminTypes.map((type) => ({ type, key: "" })),
  actions: ["all"],
};

const schematicPolicy: PolicyWithoutSubjectsOrKey = {
  objects: [{ type: "schematic", key: "" }],
  actions: ["all"],
};

const baseTypes: ontology.ResourceType[] = [
  "channel",
  "device",
  "group",
  "label",
  "lineplot",
  "rack",
  "range",
  "range-alias",
  "task",
  "workspace",
];

const basePolicy: PolicyWithoutSubjectsOrKey = {
  objects: baseTypes.map((type) => ({ type, key: "" })),
  actions: ["all"],
};

const allTypes: ontology.ResourceType[] = [
  ...adminTypes,
  "schematic",
  ...baseTypes,
  "builtin",
];

const retrievePolicy: PolicyWithoutSubjectsOrKey = {
  objects: allTypes.map((type) => ({ type, key: "" })),
  actions: ["retrieve"],
};

const basicPolicies: PolicyWithoutSubjectsOrKey[] = [basePolicy, retrievePolicy];

type ConsolePolicyRecord = Record<ConsolePolicy, PolicyWithoutSubjectsOrKey>;

export const consolePolicyMap: ConsolePolicyRecord = {
  admin: adminPolicy,
  schematic: schematicPolicy,
};

export const allowAllPolicy: PolicyWithoutSubjectsOrKey = {
  objects: [{ type: "allow_all", key: "" }],
  actions: [],
};

//TODO: DRY
export const consolePolicyKeysZ = z.object({
  admin: z.string().optional(),
  schematic: z.string().optional(),
});

// if permission exists, return a key, otherwise return undefined
type ConsolePolicyKeys = z.infer<typeof consolePolicyKeysZ>;

//TODO: repetitive
export const permissionsZ = z.object({
  admin: z.boolean(),
  schematic: z.boolean(),
});

export type Permissions = z.infer<typeof permissionsZ>;

//TODO: repetitive
export const initialPermissions: Permissions = {
  admin: false,
  schematic: false,
};

//TODO: repetitive
export const convertKeysToPermissions = (keys: ConsolePolicyKeys): Permissions => ({
  admin: keys.admin != null,
  schematic: keys.schematic != null,
});

export const policiesAreEqual = (
  a: access.Policy,
  b: PolicyWithoutSubjectsOrKey,
): boolean => deep.equal(a.objects, b.objects) && deep.equal(a.actions, b.actions);

// this gets keys of permisisons for users
export const getConsolePolicyKeys = async (
  client: Synnax,
  userKey: string,
): Promise<ConsolePolicyKeys> => {
  const permissions: ConsolePolicyKeys = {};
  const policies = await client.access.retrieveFor({ type: "user", key: userKey });
  consolePolicySet.forEach((consolePolicy) => {
    const existingPolicy = policies.find((policy) =>
      policiesAreEqual(policy, consolePolicyMap[consolePolicy]),
    );
    permissions[consolePolicy] = existingPolicy?.key;
  });
  return permissions;
};

// this one checks if the current user is the root user
export const getIsRootUser = async (
  client: Synnax,
  userKey: string,
): Promise<boolean> => {
  const policies = await client.access.retrieveFor({ type: "user", key: userKey });
  return policies.some((policy) => policiesAreEqual(policy, allowAllPolicy));
};

// this one is used when registering a user
export const setBasePermissions = (client: Synnax, userKey: string): void => {
  const subjects = { type: "user" as const, key: userKey };
  const policiesToCreate = basicPolicies.map((policy) => ({ ...policy, subjects }));
  client.access.create(policiesToCreate);
};
