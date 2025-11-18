// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import z from "zod";

export const handleZ = z.object({
  node: z.string(),
  param: z.string(),
});
export interface Handle extends z.infer<typeof handleZ> {}

export const edgeZ = z.object({
  key: z.string(),
  source: handleZ,
  target: handleZ,
});
export interface Edge extends z.infer<typeof edgeZ> {}

export const nodeZ = z.object({
  key: z.string(),
  type: z.string(),
});
export interface Node extends z.infer<typeof nodeZ> {}
