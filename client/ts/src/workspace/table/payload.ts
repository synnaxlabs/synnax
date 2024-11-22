// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnknownRecord, unknownRecordZ } from "@synnaxlabs/x/record";
import { z } from "zod";

import { ontology } from "@/ontology";

export const keyZ = z.string().uuid();
export type Key = z.infer<typeof keyZ>;
export type Params = Key | Key[];

export const tableZ = z.object({
  key: z.string(),
  name: z.string(),
  data: unknownRecordZ.or(z.string().transform((s) => JSON.parse(s) as UnknownRecord)),
});

export const tableRemoteZ = z.object({
  key: z.string(),
  name: z.string(),
  data: z.string().transform((s) => JSON.parse(s) as UnknownRecord),
});

export type Table = z.infer<typeof tableZ>;

export const ONTOLOGY_TYPE: ontology.ResourceType = "table";

export const ontologyID = (key: Key): ontology.ID =>
  new ontology.ID({ type: ONTOLOGY_TYPE, key });
