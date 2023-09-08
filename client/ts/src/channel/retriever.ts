// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { UnaryClient } from "@synnaxlabs/freighter";
import { toArray } from "@synnaxlabs/x";
import { z } from "zod";

import {
  type Key,
  type KeyOrName,
  type Keys,
  type KeysOrNames,
  type Name,
  type Names,
  type Params,
  type Payload,
  payload,
} from "@/channel/payload";
import { QueryError } from "@/errors";

const reqZ = z.object({
  leaseholder: z.number().optional(),
  keys: z.number().array().optional(),
  names: z.string().array().optional(),
  search: z.string().optional(),
});

type Request = z.infer<typeof reqZ>;

const resZ = z.object({
  channels: payload.array(),
});

export interface Retriever {
  retrieve: (channels: Params) => Promise<Payload[]>;
  search: (term: string) => Promise<Payload[]>;
}

export class ClusterRetriever implements Retriever {
  private static readonly ENDPOINT = "/channel/retrieve";
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async search(term: string): Promise<Payload[]> {
    return await this.execute({ search: term });
  }

  async retrieve(channels: Params): Promise<Payload[]> {
    const { variant, normalized } = analyzeParams(channels);
    return await this.execute({ [variant]: normalized });
  }

  private async execute(request: Request): Promise<Payload[]> {
    const [res, err] = await this.client.send(ClusterRetriever.ENDPOINT, request, resZ);
    if (err != null) throw err;
    return res.channels;
  }
}

export class CacheRetriever implements Retriever {
  private readonly cache: Map<number, Payload>;
  private readonly namesToKeys: Map<string, number>;
  private readonly wrapped: Retriever;

  constructor(wrapped: Retriever) {
    this.cache = new Map();
    this.namesToKeys = new Map();
    this.wrapped = wrapped;
  }

  async search(term: string): Promise<Payload[]> {
    return await this.wrapped.search(term);
  }

  async retrieve(channels: Params): Promise<Payload[]> {
    const { normalized } = analyzeParams(channels);
    const results: Payload[] = [];
    const toFetch: KeysOrNames = [];
    normalized.forEach((keyOrName) => {
      const c = this.getFromCache(keyOrName);
      if (c != null) results.push(c);
      else toFetch.push(keyOrName as never);
    });
    if (toFetch.length === 0) return results;
    const fetched = await this.wrapped.retrieve(toFetch);
    this.updateCache(fetched);
    return results.concat(fetched);
  }

  private updateCache(channels: Payload[]): void {
    channels.forEach((channel) => {
      this.cache.set(channel.key, channel);
      this.namesToKeys.set(channel.name, channel.key);
    });
  }

  private getFromCache(channel: KeyOrName): Payload | undefined {
    const key = typeof channel === "number" ? channel : this.namesToKeys.get(channel);
    if (key == null) return undefined;
    return this.cache.get(key);
  }
}

export type ParamAnalysisResult =
  | {
      single: true;
      variant: "names";
      normalized: Names;
      actual: Name;
    }
  | {
      single: true;
      variant: "keys";
      normalized: Keys;
      actual: Key;
    }
  | {
      single: false;
      variant: "keys";
      normalized: Keys;
      actual: Keys;
    }
  | {
      single: false;
      variant: "names";
      normalized: Names;
      actual: Names;
    };

export const analyzeParams = (channels: Params): ParamAnalysisResult => {
  const normal = toArray(channels) as KeysOrNames;
  if (normal.length === 0) throw new QueryError("No channels provided");
  return {
    single: !Array.isArray(channels),
    variant: typeof normal[0] === "number" ? "keys" : "names",
    normalized: normal,
    actual: channels,
  } as const as ParamAnalysisResult;
};
