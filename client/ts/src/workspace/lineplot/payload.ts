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

export const linePlotZ = z.object({
  key: z.string(),
  name: z.string(),
  data: unknownRecordZ.or(z.string().transform((s) => JSON.parse(s) as UnknownRecord)),
});

export type LinePlot = z.infer<typeof linePlotZ>;

export const LineplotOntologyType = "lineplot" as ontology.ResourceType;

export const ontologyID = (key: Key): ontology.ID =>
  new ontology.ID({ type: LineplotOntologyType, key: key });
