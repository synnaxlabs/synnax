// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array } from "@synnaxlabs/x";
import { z } from "zod";

import { ranger } from "@/ranger";

export const pairZ = z.object({
  range: ranger.keyZ,
  key: z.string(),
  value: z.string(),
});
export interface Pair extends z.infer<typeof pairZ> {}

export const pairKey = ({ range, key }: Omit<Pair, "value">): string =>
  `${range}<--->${key}`;

export const getRequestZ = z.object({
  range: ranger.keyZ,
  keys: z.string().array(),
});
export interface GetRequest extends z.infer<typeof getRequestZ> {}

export const getResponseZ = z.object({ pairs: array.nullishToEmpty(pairZ) });
export interface GetResponse extends z.infer<typeof getResponseZ> {}

export const setRequestZ = z.object({
  range: ranger.keyZ,
  pairs: pairZ.array(),
});
export interface SetRequest extends z.infer<typeof setRequestZ> {}

export const deleteRequestZ = z.object({
  range: ranger.keyZ,
  keys: z.string().array(),
});
export interface DeleteRequest extends z.infer<typeof deleteRequestZ> {}
