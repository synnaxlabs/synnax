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

import { label } from "@/label";
import { type ontology } from "@/ontology";
import { slate } from "@/slate";
import { nullableArrayZ } from "@/util/zod";

export const keyZ = z.uuid();
export type Key = z.infer<typeof keyZ>;
export type Params = Key | Key[];

const effectDetailsZ = z.object({ effect: keyZ });

export const statusZ = status.statusZ(effectDetailsZ);

export interface Status extends z.infer<typeof statusZ> {}

export const effectZ = z.object({
  key: keyZ,
  name: z.string(),
  enabled: z.boolean(),
  slate: slate.keyZ,
  labels: nullableArrayZ(label.labelZ).optional(),
  status: statusZ.optional(),
});
export interface Effect extends z.infer<typeof effectZ> {}

export const newZ = effectZ.partial({ key: true });
export interface New extends z.input<typeof newZ> {}

export const ontologyID = (key: Key): ontology.ID => ({ type: "effect", key });

export const ontologyIDsFromEffects = (effects: Effect[]): ontology.ID[] =>
  effects.map((e) => ontologyID(e.key));
