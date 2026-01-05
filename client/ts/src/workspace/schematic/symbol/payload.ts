// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { location, xy } from "@synnaxlabs/x";
import { z } from "zod";

export const keyZ = z.uuid();
export type Key = z.infer<typeof keyZ>;
export type Params = Key | Key[];

export const regionZ = z.object({
  key: z.string(),
  name: z.string(),
  selectors: z.string().array().default([]),
  strokeColor: z.string().optional(),
  fillColor: z.string().optional(),
});

export interface Region extends z.infer<typeof regionZ> {}

export const stateZ = z.object({
  key: z.string(),
  name: z.string(),
  regions: regionZ.array(),
});

export interface State extends z.infer<typeof stateZ> {}

export const handleZ = z.object({
  key: z.string(),
  position: xy.xy,
  orientation: location.outer,
});

export interface Handle extends z.infer<typeof handleZ> {}

const viewportZ = z.object({
  zoom: z.number().positive().default(1),
  position: xy.xy,
});

export const specZ = z.object({
  svg: z.string().min(1, "SVG is required"),
  states: stateZ.array(),
  variant: z.string().min(1, "Variant is required"),
  handles: handleZ.array(),
  scale: z.number().positive().default(1),
  scaleStroke: z.boolean().default(false),
  previewViewport: viewportZ.default({ zoom: 1, position: { x: 0, y: 0 } }),
});

export interface Spec extends z.infer<typeof specZ> {}

export const symbolZ = z.object({
  key: keyZ,
  version: z.literal(1).default(1),
  name: z.string().min(1, "Name is required"),
  data: specZ,
});

export const newZ = symbolZ.partial({ key: true });
export interface New extends z.input<typeof newZ> {}

export interface Symbol extends z.infer<typeof symbolZ> {}
