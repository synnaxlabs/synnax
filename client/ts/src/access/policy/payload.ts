// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, zod } from "@synnaxlabs/x";
import { z } from "zod";

import { actionZ } from "@/access/types.gen";
import { ontology } from "@/ontology";

export const keyZ = z.uuid();
export type Key = z.infer<typeof keyZ>;

export const policyZ = z.object({
  key: keyZ,
  name: z.string(),
  objects: array.nullishToEmpty(ontology.idZ),
  actions: array.nullishToEmpty(actionZ),
  internal: z.boolean(),
});
export interface Policy extends z.infer<typeof policyZ> {}

export const newZ = z.object({
  key: keyZ.optional(),
  name: z.string(),
  objects: zod.toArray(ontology.idZ),
  actions: zod.toArray(actionZ),
});
export interface New extends z.input<typeof newZ> {}

export const ontologyID = ontology.createIDFactory<Key>("policy");
export const TYPE_ONTOLOGY_ID = ontologyID("");
