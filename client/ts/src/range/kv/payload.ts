// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array } from "@synnaxlabs/x";
import { z } from "zod";

import { keyZ } from "@/range/payload";

export const SET_CHANNEL = "sy_range_kv_set";
export const DELETE_CHANNEL = "sy_range_kv_delete";

export const pairZ = z.object({ range: keyZ, key: z.string(), value: z.string() });
export interface Pair extends z.infer<typeof pairZ> {}

export const createPairKey = ({ range, key }: Omit<Pair, "value">) =>
  `${range}<--->${key}`;

export const getReqZ = z.object({ range: keyZ, keys: z.string().array() });
export interface GetRequest extends z.infer<typeof getReqZ> {}

export const getResZ = z.object({ pairs: array.nullableZ(pairZ) });

export const setReqZ = z.object({ range: keyZ, pairs: pairZ.array() });
export interface SetRequest extends z.infer<typeof setReqZ> {}

export const deleteReqZ = z.object({ range: keyZ, keys: z.string().array() });
export interface DeleteRequest extends z.infer<typeof deleteReqZ> {}
