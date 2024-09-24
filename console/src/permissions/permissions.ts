// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, policy, schematic, user } from "@synnaxlabs/client";
import { deep } from "@synnaxlabs/x";
import { z } from "zod";

const consolePolicyZ = z.enum(["admin", "schematic"]);
export type ConsolePolicy = z.infer<typeof consolePolicyZ>;

export const consolePolicySet = new Set<ConsolePolicy>(consolePolicyZ.options);

type PolicyWithoutSubjectsOrKey = Omit<policy.Policy, "subjects" | "key">;

const adminPolicy: PolicyWithoutSubjectsOrKey = {
  objects: [
    { type: user.ONTOLOGY_TYPE, key: "" },
    { type: policy.ONTOLOGY_TYPE, key: "" },
  ],
  actions: [access.ALL_ACTION],
};

const schematicPolicy: PolicyWithoutSubjectsOrKey = {
  objects: [{ type: schematic.ONTOLOGY_TYPE, key: "" }],
  actions: [access.ALL_ACTION],
};

export const consolePolicyKeysZ = z.object({
  schematic: z.string().optional(),
  admin: z.string().optional(),
});
type ConsolePolicyKeys = z.infer<typeof consolePolicyKeysZ>;

type ConsolePolicyRecord = Record<ConsolePolicy, PolicyWithoutSubjectsOrKey>;
export const consolePolicyRecord: ConsolePolicyRecord = {
  admin: adminPolicy,
  schematic: schematicPolicy,
};

export const permissionsZ = z.object({
  schematic: z.boolean(),
  admin: z.boolean(),
});
type Permissions = z.infer<typeof permissionsZ>;

export const convertKeysToPermissions = (keys: ConsolePolicyKeys): Permissions => ({
  schematic: keys.schematic != null,
  admin: keys.admin != null,
});

const policiesAreEqual = (a: policy.Policy, b: PolicyWithoutSubjectsOrKey): boolean =>
  deep.equal(a.objects, b.objects) && deep.equal(a.actions, b.actions);

export const convertPoliciesToKeys = (policies: policy.Policy[]): ConsolePolicyKeys => {
  const permissions: ConsolePolicyKeys = {};
  consolePolicySet.forEach((consolePolicy) => {
    const existingPolicy = policies.find((policy) =>
      policiesAreEqual(policy, consolePolicyRecord[consolePolicy]),
    );
    permissions[consolePolicy] = existingPolicy?.key;
  });
  return permissions;
};
