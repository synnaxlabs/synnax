// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { record } from "@synnaxlabs/x/record";
import { z } from "zod";

import { parseWithoutKeyConversion } from "@/util/parseWithoutKeyConversion";

export const keyZ = z.uuid();
export type Key = z.infer<typeof keyZ>;
export type Params = Key | Key[];

export const symbolZ = z.object({
  key: keyZ,
  name: z.string(),
  data: record.unknownZ.or(z.string().transform(parseWithoutKeyConversion)),
});
export interface Symbol extends z.infer<typeof symbolZ> {}

export const newZ = symbolZ
  .partial({ key: true })
  .transform((p) => ({ ...p, data: JSON.stringify(p.data) }));
export interface New extends z.input<typeof newZ> {}

export const remoteZ = symbolZ.extend({
  data: z.string().transform(parseWithoutKeyConversion),
});

// Symbol specification types for data field
export const regionZ = z.object({
  id: z.string(),
  name: z.string(),
  selector: z.string(),
  strokeColor: z.string().optional(),
  fillColor: z.string().optional(),
});

export interface Region extends z.infer<typeof regionZ> {}

export const stateZ = z.object({
  id: z.uuid(),
  color: z.string(),
  name: z.string(),
  regions: regionZ.array(),
});

export interface State extends z.infer<typeof stateZ> {}

export const specZ = z.object({
  id: z.uuid(),
  name: z.string(),
  svg: z.string(),
  states: stateZ.array(),
});

export interface Spec extends z.infer<typeof specZ> {}