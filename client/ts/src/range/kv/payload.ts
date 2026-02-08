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

export const KV_SET_CHANNEL = "sy_range_kv_set";
export const KV_DELETE_CHANNEL = "sy_range_kv_delete";

export const kvPairZ = z.object({ range: keyZ, key: z.string(), value: z.string() });
export interface KVPair extends z.infer<typeof kvPairZ> {}

export const kvPairKey = ({ range, key }: Omit<KVPair, "value">) =>
  `${range}<--->${key}`;

export const getReqZ = z.object({ range: keyZ, keys: z.string().array() });
export interface GetRequest extends z.infer<typeof getReqZ> {}

export const getResZ = z.object({ pairs: array.nullableZ(kvPairZ) });

export const setReqZ = z.object({ range: keyZ, pairs: kvPairZ.array() });
export interface SetRequest extends z.infer<typeof setReqZ> {}

export const deleteReqZ = z.object({ range: keyZ, keys: z.string().array() });
export interface DeleteRequest extends z.infer<typeof deleteReqZ> {}
