// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { ontology } from "@/ontology";

export const keyZ = z.string().uuid();
export type Key = z.infer<typeof keyZ>;
export type Params = Key | Key[];

export const conditionZ = z.object({
  key: keyZ,
  type: z.string(),
  config: z.string(),
});
export interface Condition extends z.infer<typeof conditionZ> {}

export const newZ = conditionZ.partial({ key: true });
export interface New extends z.input<typeof newZ> {}

export const ONTOLOGY_TYPE = "condition";
export type OntologyType = typeof ONTOLOGY_TYPE;

export const ontologyID = (key: Key): ontology.ID =>
  new ontology.ID({ type: ONTOLOGY_TYPE, key });

export const ontologyIDsFromConditions = (conditions: Condition[]): ontology.ID[] =>
  conditions.map((c) => ontologyID(c.key));
