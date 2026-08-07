// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { DataType, zod } from "@synnaxlabs/x";
import { z } from "zod";

import { analyzeParams, type Params } from "@/channel/payload";
import { type Key, keyZ, type Payload, payloadZ } from "@/channel/types.gen";
import { keyZ as rangeKeyZ } from "@/ranger/types.gen";

export const retrieveRequestZ = z.object({
  nodeKey: zod.uint12.optional(),
  keys: keyZ.array().optional(),
  names: z.string().array().optional(),
  searchTerm: z.string().optional(),
  rangeKey: rangeKeyZ.optional(),
  limit: z.int().optional(),
  offset: z.int().optional(),
  dataTypes: DataType.z.array().optional(),
  notDataTypes: DataType.z.array().optional(),
  virtual: z.boolean().optional(),
  isIndex: z.boolean().optional(),
  internal: z.boolean().optional(),
  legacyCalculated: z.boolean().optional(),
});
export type RetrieveRequest = z.input<typeof retrieveRequestZ>;

export type RetrieveOptions = Omit<RetrieveRequest, "keys" | "names" | "search">;
export type PageOptions = Omit<RetrieveOptions, "offset" | "limit">;

const resZ = z.object({ channels: payloadZ.array().default(() => []) });

/**
 * Fetches channel payloads from the cluster. Missing channels are omitted from
 * the result, never thrown.
 */
export const retrieve = async (
  client: UnaryClient,
  channels: Params | RetrieveRequest,
  options?: RetrieveOptions,
): Promise<Payload[]> => {
  if (!Array.isArray(channels) && typeof channels === "object" && !("key" in channels))
    return await execute(client, channels);
  const res = analyzeParams(channels);
  const { variant } = res;
  let { normalized } = res;
  if (variant === "keys" && (normalized as Key[]).indexOf(0) !== -1)
    normalized = (normalized as Key[]).filter((k) => k !== 0);
  if (normalized.length === 0) return [];
  return await execute(client, { [variant]: normalized, ...options });
};

const execute = async (
  client: UnaryClient,
  request: RetrieveRequest,
): Promise<Payload[]> => {
  const res = await client.send("/channel/retrieve", request, retrieveRequestZ, resZ);
  return res.channels;
};
