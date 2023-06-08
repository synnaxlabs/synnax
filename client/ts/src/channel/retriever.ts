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
  ChannelKeyOrName,
  ChannelKeys,
  ChannelKeysOrNames,
  ChannelNames,
  ChannelParams,
  ChannelPayload,
  channelPayload,
} from "@/channel/payload";
import { QueryError, ValidationError } from "@/errors";
import { Transport } from "@/transport";

const requestSchema = z.object({
  leaseholder: z.number().optional(),
  keys: z.number().array().optional(),
  names: z.string().array().optional(),
});

type Request = z.infer<typeof requestSchema>;

const resZ = z.object({
  channels: channelPayload.array(),
});

export interface ChannelRetriever {
  retrieve: ((channels: ChannelKeyOrName) => Promise<ChannelPayload>) &
    ((channels: ChannelKeysOrNames) => Promise<ChannelPayload[]>);
  retrieveAll: () => Promise<ChannelPayload[]>;
}

export class ClusterChannelRetriever implements ChannelRetriever {
  private static readonly ENDPOINT = "/channel/retrieve";
  private readonly client: UnaryClient;

  constructor(transport: Transport) {
    this.client = transport.getClient();
  }

  private async execute(request: Request): Promise<ChannelPayload[]> {
    const [res, err] = await this.client.send(
      ClusterChannelRetriever.ENDPOINT,
      request,
      resZ
    );
    if (err != null) throw err;
    return res?.channels as ChannelPayload[];
  }

  async retrieve(channels: ChannelKeyOrName): Promise<ChannelPayload>;

  async retrieve(channels: ChannelKeysOrNames): Promise<ChannelPayload[]>;

  async retrieve(channels: ChannelParams): Promise<ChannelPayload | ChannelPayload[]> {
    const { single, variant, normalized } = analyzeChannelParams(channels);
    const res = await this.execute({ [variant]: normalized });
    if (!single) return res;
    if (res.length === 0) throw new QueryError("Channel not found");
    if (res.length > 1) throw new ValidationError("Multiple channels found");
    return res[0];
  }

  async retrieveAll(): Promise<ChannelPayload[]> {
    return await this.execute({});
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

  private get(channel: ChannelKeyOrName): ChannelPayload | undefined {
    const key = typeof channel === "number" ? channel : this.namesToKeys.get(channel);
    if (key == null) return undefined;
    return this.cache.get(key);
  }

  async retrieve(channels: ChannelKeyOrName): Promise<ChannelPayload>;

  async retrieve(channels: ChannelKeysOrNames): Promise<ChannelPayload[]>;

  async retrieve(channels: ChannelParams): Promise<ChannelPayload | ChannelPayload[]> {
    const { single, normalized } = analyzeChannelParams(channels);

    const results: ChannelPayload[] = [];
    const toFetch: ChannelKeysOrNames = [];
    normalized.forEach((name) => {
      const c = this.get(name);
      if (c != null) results.push(c);
      else toFetch.push(name as never);
    });

    if (toFetch.length > 0) {
      const fetched = await this.wrapped.retrieve(toFetch);
      fetched.forEach((channel) => {
        this.cache.set(channel.key, channel);
        this.namesToKeys.set(channel.name, channel.key);
      });
      results.push(...fetched);
    }
    if (!single) return results;
    if (results.length === 0) throw new QueryError(`Channel not found`);
    else if (results.length > 1) throw new QueryError("Multiple channels found");
    return results[0];
  }

  async retrieveAll(): Promise<ChannelPayload[]> {
    return await this.wrapped.retrieveAll();
  }
}

export type ParamAnalysisResult =
  | {
      single: boolean;
      variant: "keys";
      normalized: ChannelKeys;
    }
  | {
      single: boolean;
      variant: "names";
      normalized: ChannelNames;
    };

export const analyzeChannelParams = (channels: ChannelParams): ParamAnalysisResult => {
  const normalized = toArray(channels) as ChannelKeysOrNames;
  return {
    single: !Array.isArray(channels),
    variant: normalized[0] === "number" ? "keys" : "names",
    normalized,
  } as const as ParamAnalysisResult;
};
