// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { array } from "@synnaxlabs/x";
import { z } from "zod";

import { type Key, keyZ } from "@/ranger/payload";

export const KV_SET_CHANNEL = "sy_range_kv_set";
export const KV_DELETE_CHANNEL = "sy_range_kv_delete";

export const kvPairZ = z.object({ range: keyZ, key: z.string(), value: z.string() });
export interface KVPair extends z.infer<typeof kvPairZ> {}

export const kvPairKey = ({ range, key }: Omit<KVPair, "value">) =>
  `${range}<--->${key}`;

const getReqZ = z.object({ range: keyZ, keys: z.string().array() });
export interface GetRequest extends z.infer<typeof getReqZ> {}

const getResZ = z.object({ pairs: array.nullishToEmpty(kvPairZ) });

const setReqZ = z.object({ range: keyZ, pairs: kvPairZ.array() });
export interface SetRequest extends z.infer<typeof setReqZ> {}

const deleteReqZ = z.object({ range: keyZ, keys: z.string().array() });
export interface DeleteRequest extends z.infer<typeof deleteReqZ> {}

export class KV {
  private readonly rangeKey: Key;
  private readonly client: UnaryClient;

  constructor(rng: Key, client: UnaryClient) {
    this.rangeKey = rng;
    this.client = client;
  }

  async get(key: string): Promise<string>;
  async get(keys: string[]): Promise<Record<string, string>>;
  async get(keys: string | string[]): Promise<string | Record<string, string>> {
    const res = await sendRequired(
      this.client,
      "/range/kv/get",
      { range: this.rangeKey, keys: array.toArray(keys) },
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
    if (typeof key == "string") pairs = [{ range: this.rangeKey, key, value }];
    else
      pairs = Object.entries(key).map(([k, v]) => ({
        range: this.rangeKey,
        key: k,
        value: v,
      }));

    await sendRequired(
      this.client,
      "/range/kv/set",
      { range: this.rangeKey, pairs },
      setReqZ,
      z.unknown(),
    );
  }

  async delete(key: string | string[]): Promise<void> {
    await sendRequired(
      this.client,
      "/range/kv/delete",
      { range: this.rangeKey, keys: array.toArray(key) },
      deleteReqZ,
      z.unknown(),
    );
  }
}
