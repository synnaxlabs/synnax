// Copyright 2026 Synnax Labs, Inc.
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

import {
  deleteRequestZ,
  getRequestZ,
  getResponseZ,
  type Pair,
  setRequestZ,
} from "@/kv/payload";
import { range } from "@/range";

export const SET_CHANNEL = "sy_range_kv_set";
export const DELETE_CHANNEL = "sy_range_kv_delete";

/**
 * KV provides key-value storage operations scoped to a range.
 */
export class KV {
  private readonly rangeKey: range.Key;
  private readonly client: UnaryClient;

  constructor(rangeKey: range.Key, client: UnaryClient) {
    this.rangeKey = rangeKey;
    this.client = client;
  }

  /**
   * Get a single value by key.
   * @param key - The key to retrieve.
   * @returns The value associated with the key.
   */
  async get(key: string): Promise<string>;
  /**
   * Get multiple values by keys.
   * @param keys - The keys to retrieve.
   * @returns A record mapping keys to values.
   */
  async get(keys: string[]): Promise<Record<string, string>>;
  async get(keys: string | string[]): Promise<string | Record<string, string>> {
    const res = await sendRequired(
      this.client,
      "/kv/get",
      { range: this.rangeKey, keys: array.toArray(keys) },
      getRequestZ,
      getResponseZ,
    );
    if (typeof keys === "string") return res.pairs[0].value;
    return Object.fromEntries(res.pairs.map((pair) => [pair.key, pair.value]));
  }

  /**
   * List all key-value pairs for the range.
   * @returns A record of all keys and values.
   */
  async list(): Promise<Record<string, string>> {
    return this.get([]);
  }

  /**
   * Set a single key-value pair.
   * @param key - The key to set.
   * @param value - The value to set.
   */
  async set(key: string, value: string): Promise<void>;
  /**
   * Set multiple key-value pairs.
   * @param kv - A record of keys and values to set.
   */
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

    await sendRequired(
      this.client,
      "/kv/set",
      { range: this.rangeKey, pairs },
      setRequestZ,
      z.unknown(),
    );
  }

  /**
   * Delete one or more keys.
   * @param key - The key or keys to delete.
   */
  async delete(key: string | string[]): Promise<void> {
    await sendRequired(
      this.client,
      "/kv/delete",
      { range: this.rangeKey, keys: array.toArray(key) },
      deleteRequestZ,
      z.unknown(),
    );
  }
}

/**
 * Client provides access to the KV API.
 */
export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  /**
   * Get a KV instance scoped to a specific range.
   * @param rangeKey - The range key to scope the KV operations to.
   * @returns A KV instance for the specified range.
   */
  get(rangeKey: range.Key): KV {
    return new KV(rangeKey, this.client);
  }
}
