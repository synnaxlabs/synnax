// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { UnaryClient } from "@synnaxlabs/freighter";
import { toArray, AsyncTermSearcher } from "@synnaxlabs/x";
import { z } from "zod";

import { QueryError } from "..";

import {
  ChannelKey,
  ChannelKeyOrName,
  ChannelKeys,
  ChannelKeysOrNames,
  ChannelName,
  ChannelNames,
  ChannelParams,
  ChannelPayload,
  channelPayload,
} from "@/channel/payload";

const requestSchema = z.object({
  leaseholder: z.number().optional(),
  keys: z.number().array().optional(),
  names: z.string().array().optional(),
  search: z.string().optional(),
});

type Request = z.infer<typeof requestSchema>;

const resZ = z.object({
  channels: channelPayload.array(),
});

export interface ChannelRetriever {
  retrieve: (channels: ChannelParams) => Promise<ChannelPayload[]>;
  search: (term: string) => Promise<ChannelPayload[]>;
}

export class ClusterChannelRetriever implements ChannelRetriever {
  private static readonly ENDPOINT = "/channel/retrieve";
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async search(term: string): Promise<ChannelPayload[]> {
    return await this.execute({ search: term });
  }

  async retrieve(channels: ChannelParams): Promise<ChannelPayload[]> {
    const { variant, normalized } = analyzeChannelParams(channels);
    return await this.execute({ [variant]: normalized });
  }

  private async execute(request: Request): Promise<ChannelPayload[]> {
    const [res, err] = await this.client.send(
      ClusterChannelRetriever.ENDPOINT,
      request,
      resZ
    );
    if (err != null) throw err;
    return res.channels;
  }
}

export class CacheChannelRetriever implements ChannelRetriever {
  private readonly cache: Map<number, ChannelPayload>;
  private readonly namesToKeys: Map<string, number>;
  private readonly wrapped: ChannelRetriever;

  constructor(wrapped: ChannelRetriever) {
    this.cache = new Map();
    this.namesToKeys = new Map();
    this.wrapped = wrapped;
  }

  async search(term: string): Promise<ChannelPayload[]> {
    return await this.wrapped.search(term);
  }

  async retrieve(channels: ChannelParams): Promise<ChannelPayload[]> {
    const { normalized } = analyzeChannelParams(channels);
    const results: ChannelPayload[] = [];
    const toFetch: ChannelKeysOrNames = [];
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

  private updateCache(channels: ChannelPayload[]): void {
    channels.forEach((channel) => {
      this.cache.set(channel.key, channel);
      this.namesToKeys.set(channel.name, channel.key);
    });
  }

  private getFromCache(channel: ChannelKeyOrName): ChannelPayload | undefined {
    const key = typeof channel === "number" ? channel : this.namesToKeys.get(channel);
    if (key == null) return undefined;
    return this.cache.get(key);
  }
}

export type ParamAnalysisResult =
  | {
      single: true;
      variant: "names";
      normalized: ChannelNames;
      actual: ChannelName;
    }
  | {
      single: true;
      variant: "keys";
      normalized: ChannelKeys;
      actual: ChannelKey;
    }
  | {
      single: false;
      variant: "keys";
      normalized: ChannelKeys;
      actual: ChannelKeys;
    }
  | {
      single: false;
      variant: "names";
      normalized: ChannelNames;
      actual: ChannelNames;
    };

export const analyzeChannelParams = (channels: ChannelParams): ParamAnalysisResult => {
  const normalized = toArray(channels) as ChannelKeysOrNames;
  return {
    single: !Array.isArray(channels),
    variant: typeof normalized[0] === "number" ? "keys" : "names",
    normalized,
    actual: channels,
  } as const as ParamAnalysisResult;
};
