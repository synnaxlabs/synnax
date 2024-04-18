// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.
import type { UnaryClient } from "@synnaxlabs/freighter";
import { debounce, toArray } from "@synnaxlabs/x";
import { Mutex } from "async-mutex";
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
  rangeKey: z.string().optional(),
  limit: z.number().optional(),
  offset: z.number().optional(),
});

type Request = z.input<typeof reqZ>;

const resZ = z.object({
  channels: payload.array(),
});

export interface Retriever {
  retrieve: (channels: Params, rangeKey?: string) => Promise<Payload[]>;
  search: (term: string, rangeKey?: string) => Promise<Payload[]>;
  page: (offset: number, limit: number, rangeKey?: string) => Promise<Payload[]>;
}

export class ClusterRetriever implements Retriever {
  private static readonly ENDPOINT = "/channel/retrieve";
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async search(term: string, rangeKey?: string): Promise<Payload[]> {
    return await this.execute({ search: term, rangeKey });
  }

  async retrieve(channels: Params, rangeKey?: string): Promise<Payload[]> {
    const { variant, normalized } = analyzeParams(channels);
    return await this.execute({ [variant]: normalized, rangeKey });
  }

  async page(offset: number, limit: number, rangeKey?: string): Promise<Payload[]> {
    return await this.execute({ offset, limit, rangeKey });
  }

  private async execute(request: Request): Promise<Payload[]> {
    const [res, err] = await this.client.send(
      ClusterRetriever.ENDPOINT,
      request,
      reqZ,
      resZ,
    );
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

  async search(term: string, rangeKey?: string): Promise<Payload[]> {
    return await this.wrapped.search(term, rangeKey);
  }

  async page(offset: number, limit: number, rangeKey?: string): Promise<Payload[]> {
    return await this.wrapped.page(offset, limit, rangeKey);
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
  let normal = (toArray(channels) as KeysOrNames).filter((c) => c !== 0);
  let variant: "names" | "keys" = "keys";
  if (typeof normal[0] === "string") {
    if (isNaN(parseInt(normal[0]))) variant = "names";
    else {
      normal = normal.map((v) => parseInt(v as string));
    }
  }
  return {
    single: !Array.isArray(channels),
    variant,
    normalized: normal,
    actual: channels,
  } as const as ParamAnalysisResult;
};

export interface PromiseFns<T> {
  resolve: (value: T) => void;
  reject: (reason?: any) => void;
}

// no interval
export class DebouncedBatchRetriever implements Retriever {
  private readonly mu = new Mutex();
  private readonly requests = new Map<Keys, PromiseFns<Payload[]>>();
  private readonly wrapped: Retriever;
  private readonly debouncedRun: () => void;

  constructor(wrapped: Retriever, deb: number) {
    this.wrapped = wrapped;
    this.debouncedRun = debounce(() => {
      void this.run();
    }, deb);
  }

  async search(term: string, rangeKey?: string): Promise<Payload[]> {
    return await this.wrapped.search(term, rangeKey);
  }

  async page(offset: number, limit: number, rangeKey?: string): Promise<Payload[]> {
    return await this.wrapped.page(offset, limit, rangeKey);
  }

  async retrieve(channels: Params): Promise<Payload[]> {
    const { normalized, variant } = analyzeParams(channels);
    // Bypass on name fetches for now.
    if (variant === "names") return await this.wrapped.retrieve(normalized);
    // eslint-disable-next-line @typescript-eslint/promise-function-async
    const a = new Promise<Payload[]>((resolve, reject) => {
      void this.mu.runExclusive(() => {
        this.requests.set(normalized, { resolve, reject });
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
  params: Params,
): Promise<Payload[]> => {
  const { normalized } = analyzeParams(params);
  const results = await r.retrieve(normalized);
  const notFound: KeyOrName[] = [];
  normalized.forEach((v) => {
    if (results.find((c) => c.name === v || c.key === v) == null) notFound.push(v);
  });
  if (notFound.length > 0)
    throw new QueryError(`Could not find channels: ${JSON.stringify(notFound)}`);
  return results;
};
