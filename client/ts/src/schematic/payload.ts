// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { graph, record, xy } from "@synnaxlabs/x";
import { z } from "zod";

export const keyZ = z.uuid();
export type Key = z.infer<typeof keyZ>;
export type Params = Key | Key[];

export const nodeZ = graph.nodeZ.extend({
  position: xy.xy,
  zIndex: z.number().optional(),
});
export type Node = z.infer<typeof nodeZ>;

export const edgeZ = graph.edgeZ;
export type Edge = z.infer<typeof edgeZ>;

export const schematicZ = z.object({
  key: keyZ,
  name: z.string(),
  version: z.number(),
  snapshot: z.boolean(),
  nodes: nodeZ.array(),
  edges: edgeZ.array(),
  props: z.record(z.string(), record.unknownZ),
});
export interface Schematic extends z.infer<typeof schematicZ> {}

export const newZ = schematicZ.partial({
  key: true,
  snapshot: true,
  version: true,
  nodes: true,
  edges: true,
  props: true,
});
export type New = z.input<typeof newZ>;
