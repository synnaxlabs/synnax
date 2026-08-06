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

import { type Params, type PrimitiveParams } from "@/channel/payload";
import { type Key, keyZ, type Name, type Payload, payloadZ } from "@/channel/types.gen";
import { QueryError } from "@/errors";
import { keyZ as rangeKeyZ } from "@/ranger/types.gen";
import {
  analyzeParams as analyzeParameters,
  type ParamAnalysisResult,
} from "@/util/retrieve";

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

export const analyzeParams = (
  channels: Params,
): ParamAnalysisResult<Key | Name, { number: "keys"; string: "names" }> => {
  if (Array.isArray(channels) && channels.length > 0 && typeof channels[0] === "object")
    channels = (channels as Payload[]).map((c) => c.key);
  else if (typeof channels === "object" && "key" in channels) channels = [channels.key];
  return analyzeParameters(channels as PrimitiveParams, {
    number: "keys",
    string: "names",
  });
};

export interface Retriever {
  retrieve: ((channels: Params, opts?: RetrieveOptions) => Promise<Payload[]>) &
    ((request: RetrieveRequest) => Promise<Payload[]>);
}

export class ClusterRetriever implements Retriever {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async retrieve(
    channels: Params | RetrieveRequest,
    options?: RetrieveOptions,
  ): Promise<Payload[]> {
    if (!Array.isArray(channels) && typeof channels === "object")
      return await this.execute(channels);
    const res = analyzeParams(channels);
    const { variant } = res;
    let { normalized } = res;
    if (variant === "keys" && (normalized as Key[]).indexOf(0) !== -1)
      normalized = (normalized as Key[]).filter((k) => k !== 0);
    if (normalized.length === 0) return [];
    return await this.execute({ [variant]: normalized, ...options });
  }

  private async execute(request: RetrieveRequest): Promise<Payload[]> {
    const res = await this.client.send(
      "/channel/retrieve",
      request,
      retrieveRequestZ,
      resZ,
    );
    return res.channels;
  }
}

export const retrieveRequired = async (
  r: Retriever,
  channels: Params,
): Promise<Payload[]> => {
  const { normalized } = analyzeParams(channels);
  const results = await r.retrieve(normalized);
  const notFound: (Key | Name)[] = [];
  normalized.forEach((v) => {
    if (results.find((c) => c.name === v || c.key === v) == null) notFound.push(v);
  });
  if (notFound.length > 0)
    throw new QueryError(`Could not find channels: ${JSON.stringify(notFound)}`);
  return results;
};
