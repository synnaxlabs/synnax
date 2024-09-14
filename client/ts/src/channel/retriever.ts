// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.
import type { UnaryClient } from "@synnaxlabs/freighter";
import { debounce } from "@synnaxlabs/x/debounce";
import { DataType } from "@synnaxlabs/x/telem";
import { Mutex } from "async-mutex";
import { z } from "zod";

import {
  type Key,
  type KeyOrName,
  type Keys,
  type KeysOrNames,
  type Params,
  type Payload,
  payload,
} from "@/channel/payload";
import { QueryError } from "@/errors";
import { analyzeParams, type ParamAnalysisResult } from "@/util/retrieve";
import { nullableArrayZ } from "@/util/zod";

const reqZ = z.object({
  leaseholder: z.number().optional(),
  keys: z.number().array().optional(),
  names: z.string().array().optional(),
  search: z.string().optional(),
  rangeKey: z.string().optional(),
  limit: z.number().optional(),
  offset: z.number().optional(),
  dataTypes: DataType.z.array().optional(),
  notDataTypes: DataType.z.array().optional(),
  virtual: z.boolean().optional(),
  isIndex: z.boolean().optional(),
  internal: z.boolean().optional(),
});

type Request = z.input<typeof reqZ>;

export type RetrieveOptions = Omit<Request, "keys" | "names" | "search">;
export type PageOptions = Omit<RetrieveOptions, "offset" | "limit">;

const resZ = z.object({
  channels: nullableArrayZ(payload),
});

export const analyzeChannelParams = (
  channels: Params,
): ParamAnalysisResult<KeyOrName, { number: "keys"; string: "names" }> =>
  analyzeParams(channels, {
    number: "keys",
    string: "names",
  });

export interface Retriever {
  retrieve: (channels: Params, opts?: RetrieveOptions) => Promise<Payload[]>;
  search: (term: string, opts?: RetrieveOptions) => Promise<Payload[]>;
  page: (offset: number, limit: number, opts?: PageOptions) => Promise<Payload[]>;
}

export class ClusterRetriever implements Retriever {
  private static readonly ENDPOINT = "/channel/retrieve";
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async search(term: string, options?: RetrieveOptions): Promise<Payload[]> {
    return await this.execute({ search: term, ...options });
  }

  async retrieve(channels: Params, options?: RetrieveOptions): Promise<Payload[]> {
    const res = analyzeChannelParams(channels);
    const { variant } = res;
    let { normalized } = res;
    if (variant === "keys" && (normalized as Key[]).indexOf(0) !== -1)
      normalized = (normalized as Key[]).filter((k) => k !== 0);
    if (normalized.length === 0) return [];
    return await this.execute({ [variant]: normalized, ...options });
  }

  async page(offset: number, limit: number, options?: PageOptions): Promise<Payload[]> {
    return await this.execute({ offset, limit, ...options });
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
  private readonly namesToKeys: Map<string, Set<number>>;
  private readonly wrapped: Retriever;

  constructor(wrapped: Retriever) {
    this.cache = new Map();
    this.namesToKeys = new Map();
    this.wrapped = wrapped;
  }

  async search(term: string, options?: RetrieveOptions): Promise<Payload[]> {
    return await this.wrapped.search(term, options);
  }

  async page(offset: number, limit: number, options?: PageOptions): Promise<Payload[]> {
    return await this.wrapped.page(offset, limit, options);
  }

  async retrieve(channels: Params, options?: RetrieveOptions): Promise<Payload[]> {
    const { normalized } = analyzeParams<string | number>(channels, {
      string: "names",
      number: "keys",
    });
    const results: Payload[] = [];
    const toFetch: KeysOrNames = [];
    normalized.forEach((keyOrName) => {
      const c = this.get(keyOrName);
      if (c != null) results.push(...c);
      else toFetch.push(keyOrName as never);
    });
    if (toFetch.length === 0) return results;
    const fetched = await this.wrapped.retrieve(toFetch, options);
    this.set(fetched);
    return results.concat(fetched);
  }

  delete(channels: Params): void {
    const { variant, normalized } = analyzeChannelParams(channels);
    if (variant === "names")
      (normalized as string[]).forEach((name) => {
        const keys = this.namesToKeys.get(name);
        if (keys == null) return;
        keys.forEach((k) => this.cache.delete(k));
        this.namesToKeys.delete(name);
      });
    else
      (normalized as number[]).forEach((key) => {
        const channel = this.cache.get(key);
        if (channel == null) return;
        this.cache.delete(key);
        this.namesToKeys.delete(channel.name);
      });
  }

  rename(keys: Key[], names: string[]): void {
    keys.forEach((key, i) => {
      const name = names[i];
      const ch = this.cache.get(key);
      if (ch == null) return;
      this.cache.delete(key);
      const keys = this.namesToKeys.get(ch.name);
      if (keys != null) {
        keys.delete(key);
        if (keys.size === 0) this.namesToKeys.delete(ch.name);
      }
      ch.name = name;
      this.cache.set(key, ch);
      const newKeys = this.namesToKeys.get(name);
      if (newKeys == null) this.namesToKeys.set(name, new Set([key]));
      else newKeys.add(key);
    });
  }

  set(channels: Payload[]): void {
    channels.forEach((channel) => {
      this.cache.set(channel.key, channel);
      const keys = this.namesToKeys.get(channel.name);
      if (keys == null) this.namesToKeys.set(channel.name, new Set([channel.key]));
      else keys.add(channel.key);
    });
  }

  private get(channel: KeyOrName): Payload[] | undefined {
    if (typeof channel === "number") {
      const ch = this.cache.get(channel);
      if (ch == null) return undefined;
      return [ch];
    }
    const keys = this.namesToKeys.get(channel);
    if (keys == null) return undefined;
    const channels: Payload[] = [];
    keys.forEach((key) => {
      const ch = this.cache.get(key);
      if (ch != null) channels.push(ch);
    });
    if (channels.length === 0) return undefined;
    return channels;
  }
}

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

  async search(term: string, options?: RetrieveOptions): Promise<Payload[]> {
    return await this.wrapped.search(term, options);
  }

  async page(
    offset: number,
    limit: number,
    options?: RetrieveOptions,
  ): Promise<Payload[]> {
    return await this.wrapped.page(offset, limit, options);
  }

  async retrieve(channels: Params): Promise<Payload[]> {
    const { normalized, variant } = analyzeChannelParams(channels);
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
  const { normalized } = analyzeChannelParams(channels);
  const results = await r.retrieve(normalized);
  const notFound: KeyOrName[] = [];
  normalized.forEach((v) => {
    if (results.find((c) => c.name === v || c.key === v) == null) notFound.push(v);
  });
  if (notFound.length > 0)
    throw new QueryError(`Could not find channels: ${JSON.stringify(notFound)}`);
  return results;
};
