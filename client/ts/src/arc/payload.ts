// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { xy } from "@synnaxlabs/x";
import { record } from "@synnaxlabs/x/record";
import { z } from "zod/v4";

import { type Status } from "@/status/payload";
import { parseWithoutKeyConversion } from "@/util/parseWithoutKeyConversion";

export const irNodeZ = z.object({
  key: z.string(),
  type: z.string(),
  config: record.unknownZ.or(z.string().transform(parseWithoutKeyConversion)),
  source: z.string().optional(),
});

export const graphNodeZ = irNodeZ.extend({
  position: xy.xy,
});

export const handleZ = z.object({ param: z.string(), node: z.string() });

export const edgeZ = z.object({ source: handleZ, target: handleZ });

export const irZ = z.object({
  nodes: irNodeZ.array(),
  edges: edgeZ.array(),
});

export const graphZ = z.object({
  nodes: graphNodeZ.array(),
  edges: edgeZ.array(),
});

export const textZ = z.object({ raw: z.string() });

export interface IR extends z.infer<typeof irZ> {}
export interface Graph extends z.infer<typeof graphZ> {}
export interface Text extends z.infer<typeof textZ> {}

export const keyZ = z.uuid();
export type Key = z.infer<typeof keyZ>;
export type Params = Key | Key[];

export const arcZ = z.object({
  key: keyZ,
  name: z.string(),
  graph: graphZ,
  text: textZ,
});

export interface Arc extends z.infer<typeof arcZ> {
  status?: Status;
}

export const newZ = arcZ.partial({ key: true });
export interface New extends z.input<typeof newZ> {}

export const ONTOLOGY_TYPE = "arc";
export type OntologyType = typeof ONTOLOGY_TYPE;
