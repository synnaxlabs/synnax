// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { ontology } from "@/ontology";
import { crudeIDZ, idZ } from "@/ontology/payload";
import { nullableArrayZ } from "@/util/zod";

export const keyZ = z.string().uuid();
export type Key = z.infer<typeof keyZ>;
export type Params = Key | Key[];

export const actionZ = z.union([
  z.literal("all"),
  z.literal("create"),
  z.literal("delete"),
  z.literal("retrieve"),
  z.literal("rename"),
]);
export type Action = z.infer<typeof actionZ>;

export const newPolicyZ = z.object({
  key: keyZ.optional().catch(undefined),
  subjects: crudeIDZ.array().or(crudeIDZ.transform((v) => [v])),
  objects: crudeIDZ.array().or(crudeIDZ.transform((v) => [v])),
  actions: actionZ.array().or(actionZ.transform((v) => [v])),
});
export type NewPolicy = z.input<typeof newPolicyZ>;

export const policyZ = z.object({
  key: keyZ,
  subjects: nullableArrayZ(idZ),
  objects: nullableArrayZ(idZ),
  actions: nullableArrayZ(actionZ),
});
export type Policy = z.infer<typeof policyZ>;

export const OntologyType = "policy" as ontology.ResourceType;

export const ontologyID = (key: Key): ontology.ID =>
  new ontology.ID({ type: OntologyType, key });
