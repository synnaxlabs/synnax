// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array } from "@synnaxlabs/x";
import { z } from "zod";

import { actionZ } from "@/access/payload";
import { ontology } from "@/ontology";

export const keyZ = z.uuid();
export type Key = z.infer<typeof keyZ>;

const effectZ = z.enum(["allow", "deny"]);
export type Effect = z.infer<typeof effectZ>;

export const policyZ = z.object({
  key: keyZ,
  name: z.string(),
  effect: effectZ,
  objects: array.nullableZ(ontology.idZ),
  actions: array.nullableZ(actionZ),
});
export interface Policy extends z.infer<typeof policyZ> {}

export const newZ = z.object({
  key: keyZ.optional(),
  name: z.string(),
  effect: effectZ,
  objects: ontology.idZ.array().or(ontology.idZ.transform((id) => [id])),
  actions: actionZ.array().or(actionZ.transform((action) => [action])),
});
export interface New extends z.input<typeof newZ> {}
