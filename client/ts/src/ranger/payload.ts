// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange } from "@synnaxlabs/x/telem";
import { z } from "zod";

import { label } from "@/label";
import { nullableArrayZ } from "@/util/zod";

export const keyZ = z.uuid();
export type Key = z.infer<typeof keyZ>;
export const nameZ = z.string().min(1);
export type Name = z.infer<typeof nameZ>;
export type Keys = Key[];
export type Names = Name[];
export type Params = Key | Name | Keys | Names;

export const stageZ = z.enum(["to_do", "in_progress", "completed"]);

export type Stage = z.infer<typeof stageZ>;

export const payloadZ = z.object({
  key: keyZ,
  name: nameZ,
  timeRange: TimeRange.z,
  stage: stageZ.optional().default("to_do"),
  color: z.string().optional(),
  labels: nullableArrayZ(label.labelZ).optional(),
  get parent(): z.ZodNullable<typeof payloadZ> {
    return payloadZ.nullable();
  },
});

export type Payload = z.infer<typeof payloadZ>;

export const newZ = payloadZ
  .omit({ parent: true, labels: true })
  .partial({ key: true });
export interface New extends z.input<typeof newZ> {}

export const ONTOLOGY_TYPE = "range";
export type OntologyType = typeof ONTOLOGY_TYPE;

export const ALIAS_ONTOLOGY_TYPE = "range-alias";
export type AliasOntologyType = typeof ALIAS_ONTOLOGY_TYPE;
