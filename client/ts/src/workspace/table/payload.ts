// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { unknownRecordZ } from "@synnaxlabs/x/record";
import { z } from "zod";

import { parseWithoutKeyConversion } from "@/util/parseWithoutKeyConversion";

export const keyZ = z.string().uuid();
export type Key = z.infer<typeof keyZ>;
export type Params = Key | Key[];

export const tableZ = z.object({
  key: keyZ,
  name: z.string(),
  data: unknownRecordZ.or(z.string().transform(parseWithoutKeyConversion)),
});
export interface Table extends z.infer<typeof tableZ> {}

export const newZ = tableZ
  .partial({ key: true })
  .transform((p) => ({ ...p, data: JSON.stringify(p.data) }));
export interface New extends z.input<typeof newZ> {}

export const remoteZ = tableZ.extend({
  data: z.string().transform(parseWithoutKeyConversion),
});

export const ONTOLOGY_TYPE = "table";
export type OntologyType = typeof ONTOLOGY_TYPE;
