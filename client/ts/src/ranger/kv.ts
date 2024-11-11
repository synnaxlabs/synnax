// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { isObject } from "@synnaxlabs/x/identity";
import { toArray } from "@synnaxlabs/x/toArray";
import { z } from "zod";

import { type framer } from "@/framer";
import { type Key, keyZ } from "@/ranger/payload";
import { signals } from "@/signals";
import { nullableArrayZ } from "@/util/zod";

const getReqZ = z.object({
  range: keyZ,
  keys: z.string().array(),
});

const kvPairZ = z.object({
  range: keyZ,
  key: z.string(),
  value: z.string(),
});

const getResZ = z.object({
  pairs: nullableArrayZ(kvPairZ),
});

export type GetRequest = z.infer<typeof getReqZ>;

const setReqZ = z.object({
  range: keyZ,
  pairs: kvPairZ.array(),
});

export type SetRequest = z.infer<typeof setReqZ>;

const deleteReqZ = z.object({
  range: keyZ,
  keys: z.string().array(),
});

export type DeleteRequest = z.infer<typeof deleteReqZ>;

export type KVPair = z.infer<typeof kvPairZ>;

export class KV {
  private static readonly GET_ENDPOINT = "/range/kv/get";
  private static readonly SET_ENDPOINT = "/range/kv/set";
  private static readonly DELETE_ENDPOINT = "/range/kv/delete";
  private readonly rangeKey: Key;
  private readonly client: UnaryClient;
  private readonly frameClient: framer.Client;

  constructor(rng: Key, client: UnaryClient, frameClient: framer.Client) {
    this.rangeKey = rng;
    this.client = client;
    this.frameClient = frameClient;
  }

  async get(key: string): Promise<string>;

  async get(keys: string[]): Promise<Record<string, string>>;

  async get(keys: string | string[]): Promise<string | Record<string, string>> {
    const res = await sendRequired(
      this.client,
      KV.GET_ENDPOINT,
      { range: this.rangeKey, keys: toArray(keys) },
      getReqZ,
      getResZ,
    );
    if (typeof keys === "string") return res.pairs[0].value;
    return Object.fromEntries(res.pairs.map((pair) => [pair.key, pair.value]));
  }

  async list(): Promise<Record<string, string>> {
    return this.get([]);
  }

  async set(key: string, value: string): Promise<void>;

  async set(kv: Record<string, string>): Promise<void>;

  async set(key: string | Record<string, string>, value: string = ""): Promise<void> {
    let pairs: KVPair[];
    if (isObject(key))
      pairs = Object.entries(key).map(([k, v]) => ({
        range: this.rangeKey,
        key: k,
        value: v,
      }));
    else pairs = [{ range: this.rangeKey, key, value }];
    await sendRequired(
      this.client,
      KV.SET_ENDPOINT,
      { range: this.rangeKey, pairs },
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

  async openTracker(): Promise<signals.Observable<string, KVPair>> {
    return await signals.openObservable<string, KVPair>(
      this.frameClient,
      "sy_range_kv_set",
      "sy_range_kv_delete",
      (variant, data) => {
        if (variant === "delete")
          return data.toStrings().map((combinedKey) => {
            const [range, key] = combinedKey.split("<--->", 2);
            return { variant, key: combinedKey, value: { range, key, value: "" } };
          });
        return data.parseJSON(kvPairZ).map((pair) => ({
          variant,
          key: `${pair.range}${pair.key}`,
          value: pair,
        }));
      },
    );
  }
}
