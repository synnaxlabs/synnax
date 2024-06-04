// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired,type UnaryClient } from "@synnaxlabs/freighter";
import { isObject } from "@synnaxlabs/x/identity";
import { toArray } from "@synnaxlabs/x/toArray";
import { z } from "zod";

import { type Key, keyZ } from "@/ranger/payload";

const getReqZ = z.object({
  range: keyZ,
  keys: z.string().array(),
});

const getResZ = z.object({
  pairs: z.record(z.string(), z.string()),
});

export type GetRequest = z.infer<typeof getReqZ>;

const setReqZ = z.object({
  range: keyZ,
  pairs: z.record(z.string(), z.string()),
});

export type SetRequest = z.infer<typeof setReqZ>;

const deleteReqZ = z.object({
  range: keyZ,
  keys: z.string().array(),
});

export type DeleteRequest = z.infer<typeof deleteReqZ>;

export class KV {
  private static readonly GET_ENDPOINT = "/range/kv/get";
  private static readonly SET_ENDPOINT = "/range/kv/set";
  private static readonly DELETE_ENDPOINT = "/range/kv/delete";
  private readonly rangeKey: Key;
  private readonly client: UnaryClient;

  constructor(rng: Key, client: UnaryClient) {
    this.rangeKey = rng;
    this.client = client;
  }

  async get(key: string): Promise<string>;

  async get(keys: string[]): Promise<Record<string, string>>;

  async get(keys: string | string[]): Promise<string | Record<string, string>> {
    const [res, err] = await this.client.send(
      KV.GET_ENDPOINT,
      { range: this.rangeKey, keys: toArray(keys) },
      getReqZ,
      getResZ,
    );
    if (err != null) throw err;
    return Array.isArray(keys) ? res.pairs : res.pairs[keys];
  }

  async list(): Promise<Record<string, string>> {
    return this.get([]);
  }

  async set(key: string, value: string): Promise<void>;

  async set(kv: Record<string, string>): Promise<void>;

  async set(key: string | Record<string, string>, value: string = ""): Promise<void> {
    await sendRequired(
      this.client,
      KV.SET_ENDPOINT,
      {
        range: this.rangeKey,
        pairs: isObject(key) ? key : { [key]: value },
      },
      setReqZ,
      z.unknown(),
    );
  }

  async delete(key: string | string[]): Promise<void> {
    await sendRequired(
      this.client,
      KV.DELETE_ENDPOINT,
      { range: this.rangeKey, keys: toArray(key) },
      deleteReqZ,
      z.unknown(),
    );
  }
}
