// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { record, xy } from "@synnaxlabs/x";
import { z } from "zod";

export const keyZ = z.uuid();
export type Key = z.infer<typeof keyZ>;
export type Params = Key | Key[];

const nodeZ = z.object({
  key: z.string(),
  position: xy.xy,
  selected: z.boolean().optional(),
  zIndex: z.number().optional(),
  type: z.string().optional(),
});

export type Node = z.infer<typeof nodeZ>;

export const edgeZ = z.object({
  key: z.string(),
  source: z.string(),
  target: z.string(),
  id: z.string(),
  data: record.unknownZ.optional(),
  sourceHandle: z.string().nullable().optional(),
  targetHandle: z.string().nullable().optional(),
});

export type Edge = z.infer<typeof edgeZ>;

export const stateZ = z.object({
  version: z.string(),
  nodes: nodeZ.array(),
  edges: edgeZ.array(),
  props: z.record(z.string(), record.unknownZ),
});

export type State = z.infer<typeof stateZ>;

export const schematicZ = z.object({
  key: keyZ,
  name: z.string(),
  data: stateZ,
  snapshot: z.boolean(),
});
export interface Schematic extends z.infer<typeof schematicZ> {}

export const newZ = schematicZ
  .partial({ key: true, snapshot: true })
  .transform((p) => ({ ...p, data: JSON.stringify(p.data) }));
export interface New extends z.input<typeof newZ> {}
