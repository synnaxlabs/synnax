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

import { type channel } from "@/channel";
import {
  deleteReqZ,
  deleteResZ,
  listReqZ,
  listResZ,
  resolveReqZ,
  resolveResZ,
  retrieveReqZ,
  retrieveResZ,
  setReqZ,
  setResZ,
} from "@/range/alias/payload";
import { type Key } from "@/range/payload";

export class Client {
  private readonly cache = new Map<string, channel.Key>();
  private readonly client: UnaryClient;
  private readonly rangeKey: Key;

  constructor(rangeKey: Key, client: UnaryClient) {
    this.rangeKey = rangeKey;
    this.cache = new Map();
    this.client = client;
  }

  resolve(aliases: string): Promise<channel.Key>;

  resolve(aliases: string[]): Promise<Record<string, channel.Key>>;

  async resolve(
    aliases: string | string[],
  ): Promise<channel.Key | Record<string, channel.Key>> {
    const toFetch: string[] = [];
    const isSingle = typeof aliases === "string";
    const cached: Record<string, channel.Key> = {};
    if (isSingle) {
      const c = this.cache.get(aliases);
      if (c != null) return c;
      toFetch.push(aliases);
    } else
      aliases.forEach((alias) => {
        const c = this.cache.get(alias);
        if (c != null) cached[alias] = c;
        else toFetch.push(alias);
      });
    if (toFetch.length === 0) return cached;
    const res = await sendRequired<typeof resolveReqZ, typeof resolveResZ>(
      this.client,
      "/range/alias/resolve",
      { range: this.rangeKey, aliases: toFetch },
      resolveReqZ,
      resolveResZ,
    );
    Object.entries(res.aliases).forEach(([alias, key]) => this.cache.set(alias, key));
    return isSingle ? res.aliases[toFetch[0]] : { ...cached, ...res.aliases };
  }

  async set(aliases: Record<channel.Key, string>): Promise<void> {
    await sendRequired<typeof setReqZ, typeof setResZ>(
      this.client,
      "/range/alias/set",
      { range: this.rangeKey, aliases },
      setReqZ,
      setResZ,
    );
  }

  async list(): Promise<Record<channel.Key, string>> {
    return (
      await sendRequired<typeof listReqZ, typeof listResZ>(
        this.client,
        "/range/alias/list",
        { range: this.rangeKey },
        listReqZ,
        listResZ,
      )
    ).aliases;
  }

  async retrieve(alias: channel.Key): Promise<string>;
  async retrieve(aliases: channel.Key[]): Promise<Record<channel.Key, string>>;

  async retrieve(
    alias: channel.Key | channel.Key[],
  ): Promise<string | Record<channel.Key, string>> {
    const isSingle = typeof alias === "number";
    const res = await sendRequired<typeof retrieveReqZ, typeof retrieveResZ>(
      this.client,
      "/range/alias/retrieve",
      { range: this.rangeKey, channels: array.toArray(alias) },
      retrieveReqZ,
      retrieveResZ,
    );
    return isSingle ? res.aliases[alias] : res.aliases;
  }

  async delete(aliases: channel.Key | channel.Key[]): Promise<void> {
    await sendRequired<typeof deleteReqZ, typeof deleteResZ>(
      this.client,
      "/range/alias/delete",
      { range: this.rangeKey, channels: array.toArray(aliases) },
      deleteReqZ,
      deleteResZ,
    );
  }
}
