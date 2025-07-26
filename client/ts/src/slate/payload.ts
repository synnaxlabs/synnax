// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { record } from "@synnaxlabs/x/record";
import { z } from "zod/v4";

import { parseWithoutKeyConversion } from "@/util/parseWithoutKeyConversion";

export const nodeZ = z.object({
  key: z.string(),
  type: z.string(),
  config: record.unknownZ.or(z.string().transform(parseWithoutKeyConversion)),
});

export const handleZ = z.object({
  key: z.string(),
  node: z.string(),
});

export const edgeZ = z.object({
  source: handleZ,
  sink: handleZ,
});

export const graphZ = z.object({
  nodes: nodeZ.array(),
  edges: edgeZ.array(),
});

export const keyZ = z.uuid();
export type Key = z.infer<typeof keyZ>;
export type Params = Key | Key[];

export const slateZ = z.object({
  key: keyZ,
  graph: graphZ,
});
export interface Slate extends z.infer<typeof slateZ> {}

export const newZ = slateZ.partial({ key: true });
export interface New extends z.input<typeof newZ> {}

export const ONTOLOGY_TYPE = "slate";
export type OntologyType = typeof ONTOLOGY_TYPE;
