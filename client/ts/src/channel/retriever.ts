// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { UnaryClient } from "@synnaxlabs/freighter";
import { z } from "zod";

import { ChannelKeyOrName, ChannelKeys, ChannelNames, ChannelParams, ChannelPayload, channelPayload } from "./payload";

import { ValidationError } from "@/errors";
import { Transport } from "@/transport";

const requestSchema = z.object({
  nodeKey: z.number().optional(),
  keys: z.number().array().optional(),
  names: z.string().array().optional(),
});

type Request = z.infer<typeof requestSchema>;

const responseSchema = z.object({
  channels: channelPayload.array(),
});

export interface ChannelRetriever {
  retrieve: ((keyOrName: ChannelKeyOrName) => Promise<ChannelPayload>) &
  ((...keysOrNames: Array<ChannelParams>) => Promise<ChannelPayload[]>);
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
      // eslint-disable-next-line @typescript-eslint/ban-ts-comment
      // @ts-expect-error
      responseSchema
    );
    if (err != null) throw err;
    return res?.channels as ChannelPayload[];
  }

  async retrieve(keyOrName: string | number): Promise<ChannelPayload>;

  async retrieve(...keysOrNames: Array<ChannelParams>): Promise<ChannelPayload[]>;

  async retrieve(
    ...keysOrNames: Array<ChannelParams>
  ): Promise<ChannelPayload | ChannelPayload[]> {
    const single = isSingle(keysOrNames);
    const [keys, names] = splitChannelparams(keysOrNames);
    const res = await this.execute({ keys, names });
    if (!single) return res;
    if (res.length === 0) throw new ValidationError("Channel not found");
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

  async retrieve(keyOrName: string | number): Promise<ChannelPayload>;

  async retrieve(...keysOrNames: Array<ChannelParams>): Promise<ChannelPayload[]>;

  async retrieve(
    ...keysOrNames: Array<ChannelParams>
  ): Promise<ChannelPayload | ChannelPayload[]> {
    const single = isSingle(keysOrNames);
    const [keys, names] = splitChannelparams(keysOrNames);


    const results: ChannelPayload[] = [];
    const toFetch: Array<string | number> = [];

    names.forEach((name) => {
      const key = this.namesToKeys.get(name);
      if (key == null) toFetch.push(name);
      else keys.push(key)
    });

    keys.forEach((key) => {
      const channel = this.cache.get(key);
      if (channel != null) results.push(channel);
      else toFetch.push(key);
    })


    if (toFetch.length > 0) {
      const fetched = await this.wrapped.retrieve(...toFetch);
      fetched.forEach((channel) => {
        this.cache.set(channel.key, channel);
        this.namesToKeys.set(channel.name, channel.key);
      });
      results.push(...fetched);
    }
    return single ? results[0] : results;
  }

  async retrieveAll(): Promise<ChannelPayload[]> {
    return await this.wrapped.retrieveAll();
  }
}

const splitChannelparams = (keysOrNames: Array<ChannelParams>): [number[], string[]] => {
  const keys: ChannelKeys = [];
  const names: ChannelNames = [];
  keysOrNames.flat().forEach((keyOrName) => {
    if (typeof keyOrName === "number") keys.push(keyOrName);
    else names.push(keyOrName);
  });
  return [keys, names];
}

const isSingle = (keysOrNames: Array<ChannelParams>): boolean => {
  return keysOrNames.length === 1 && (typeof keysOrNames[0] === "string" || typeof keysOrNames[0] === "number");
}
