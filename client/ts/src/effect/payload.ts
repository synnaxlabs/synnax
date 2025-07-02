// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/x";
import { z } from "zod/v4";

import { type ontology } from "@/ontology";
import { slate } from "@/slate";

export const keyZ = z.uuid();
export type Key = z.infer<typeof keyZ>;
export type Params = Key | Key[];

export const effectZ = z.object({
  key: keyZ,
  name: z.string(),
  enabled: z.boolean(),
  slate: slate.keyZ,
});
export interface Effect extends z.infer<typeof effectZ> {}

const effectDetailsZ = z.object({
  effect: keyZ,
});

export const statusZ = status.statusZ(effectDetailsZ);

export interface Status extends z.infer<typeof statusZ> {}

export const newZ = effectZ.partial({ key: true });
export interface New extends z.input<typeof newZ> {}

// We'll need to add this to ontology/payload.ts later
export const ONTOLOGY_TYPE = "effect";
export type OntologyType = typeof ONTOLOGY_TYPE;

export const ontologyID = (key: Key): ontology.ID => ({ type: ONTOLOGY_TYPE, key });

export const ontologyIDsFromEffects = (effects: Effect[]): ontology.ID[] =>
  effects.map((e) => ontologyID(e.key));
