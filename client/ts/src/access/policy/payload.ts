// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { actionZ } from "@/access/payload";
import { ontology } from "@/ontology";
import { nullableArrayZ } from "@/util/zod";

export const keyZ = z.string().uuid();
export type Key = z.infer<typeof keyZ>;

export const newPolicyZ = z.object({
  key: keyZ.optional(),
  subjects: ontology.crudeIDZ.array().or(ontology.crudeIDZ),
  objects: ontology.crudeIDZ.array().or(ontology.crudeIDZ),
  actions: actionZ.array().or(actionZ),
});
export type NewPolicy = z.input<typeof newPolicyZ>;

export const policyZ = z.object({
  key: keyZ,
  subjects: nullableArrayZ(ontology.idZ),
  objects: nullableArrayZ(ontology.idZ),
  actions: nullableArrayZ(actionZ),
});
export type Policy = z.infer<typeof policyZ>;

export const ONTOLOGY_TYPE: ontology.ResourceType = "policy";

export const ontologyID = (key: Key): ontology.ID =>
  new ontology.ID({ type: ONTOLOGY_TYPE, key });

export const ALLOW_ALL_ONTOLOGY_TYPE: ontology.ResourceType = "allow_all";

export const ALLOW_ALL_ONTOLOGY_ID = new ontology.ID({
  type: ALLOW_ALL_ONTOLOGY_TYPE,
  key: "",
});
