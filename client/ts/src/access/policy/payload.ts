// Copyright 2025 Synnax Labs, Inc.
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

export const policyZ = z.object({
  key: keyZ,
  subjects: nullableArrayZ(ontology.idZ),
  objects: nullableArrayZ(ontology.idZ),
  actions: nullableArrayZ(actionZ),
});
export interface Policy extends z.infer<typeof policyZ> {}

export const newZ = z.object({
  key: keyZ.optional(),
  subjects: ontology.crudeIDZ.array().or(ontology.crudeIDZ),
  objects: ontology.crudeIDZ.array().or(ontology.crudeIDZ),
  actions: actionZ.array().or(actionZ),
});
export interface New extends z.input<typeof newZ> {}
