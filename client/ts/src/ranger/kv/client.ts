// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array } from "@synnaxlabs/x";
import { z } from "zod";

import { type query } from "@/query";
import { createPairKey } from "@/ranger/kv/payload";
import { type Pair, pairZ } from "@/ranger/kv/types.gen";
import { type Key, keyZ } from "@/ranger/types.gen";

const getReqZ = z.object({ range: keyZ, keys: z.string().array() });
const getResZ = z.object({ pairs: pairZ.array().default(() => []) });
const setReqZ = z.object({ range: keyZ, pairs: pairZ.array() });
const deleteReqZ = z.object({ range: keyZ, keys: z.string().array() });

export class Client {
  private readonly rangeKey: Key;
  private readonly client: UnaryClient;
  private readonly pairs: query.Table<string, Pair>;

  constructor(rng: Key, client: UnaryClient, pairs: query.Table<string, Pair>) {
    this.rangeKey = rng;
    this.client = client;
    this.pairs = pairs;
  }

  async get(key: string): Promise<string>;
  async get(keys: string[]): Promise<Record<string, string>>;
  async get(keys: string | string[]): Promise<string | Record<string, string>> {
    const res = await this.client.send(
      "/range/kv/get",
      { range: this.rangeKey, keys: array.toArray(keys) },
      getReqZ,
      getResZ,
    );
    if (typeof keys === "string") return res.pairs[0].value;
    return Object.fromEntries(res.pairs.map((pair) => [pair.key, pair.value]));
  }

  async list(): Promise<Record<string, string>> {
    return await this.get([]);
  }

  async set(key: string, value: string): Promise<void>;
  async set(kv: Record<string, string>): Promise<void>;
  async set(key: string | Record<string, string>, value: string = ""): Promise<void> {
    let pairs: Pair[];
    if (typeof key == "string") pairs = [{ range: this.rangeKey, key, value }];
    else
      pairs = Object.entries(key).map(([k, v]) => ({
        range: this.rangeKey,
        key: k,
        value: v,
      }));

    await this.client.send(
      "/range/kv/set",
      { range: this.rangeKey, pairs },
      setReqZ,
      z.unknown(),
    );
    // Pair.key is the bare key; the table is keyed by createPairKey, so
    // keyed-object set would mis-key entries.
    pairs.forEach((p) => this.pairs.set(createPairKey(p), p));
  }

  async delete(key: string | string[]): Promise<void> {
    const keys = array.toArray(key);
    await this.client.send(
      "/range/kv/delete",
      { range: this.rangeKey, keys },
      deleteReqZ,
      z.unknown(),
    );
    this.pairs.delete(keys.map((k) => createPairKey({ range: this.rangeKey, key: k })));
  }
}
