// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { type CrudeTimeSpan, DataType, debounce, zod } from "@synnaxlabs/x";
import { Mutex } from "async-mutex";
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

export interface PromiseFns<T> {
  resolve: (value: T) => void;
  reject: (reason?: unknown) => void;
}

// no interval
export class DebouncedBatchRetriever implements Retriever {
  private readonly mu = new Mutex();
  private readonly requests = new Map<Key[], PromiseFns<Payload[]>>();
  private readonly wrapped: Retriever;
  private readonly debouncedRun: () => void;

  constructor(wrapped: Retriever, deb: CrudeTimeSpan) {
    this.wrapped = wrapped;
    this.debouncedRun = debounce(() => {
      void this.run();
    }, deb);
  }

  async retrieve(channels: Params | RetrieveRequest): Promise<Payload[]> {
    if (!Array.isArray(channels) && typeof channels === "object")
      return await this.wrapped.retrieve(channels);
    const { normalized, variant } = analyzeParams(channels);
    // Bypass on name fetches for now.
    if (variant === "names") return await this.wrapped.retrieve(normalized);

    const a = new Promise<Payload[]>((resolve, reject) => {
      void this.mu.runExclusive(() => {
        this.requests.set(normalized as Key[], { resolve, reject });
        this.debouncedRun();
      });
    });
    return await a;
  }

  async run(): Promise<void> {
    await this.mu.runExclusive(async () => {
      const allKeys = new Set<Key>();
      this.requests.forEach((_, keys) => keys.forEach((k) => allKeys.add(k)));
      try {
        const channels = await this.wrapped.retrieve(Array.from(allKeys));
        this.requests.forEach((fns, keys) =>
          fns.resolve(channels.filter((c) => keys.includes(c.key))),
        );
      } catch (e) {
        this.requests.forEach((fns) => fns.reject(e));
      } finally {
        this.requests.clear();
      }
    });
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
