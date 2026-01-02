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

import {
  deleteRequestZ,
  listRequestZ,
  listResponseZ,
  resolveRequestZ,
  resolveResponseZ,
  retrieveRequestZ,
  retrieveResponseZ,
  setRequestZ,
} from "@/alias/payload";
import { channel } from "@/channel";
import { ranger } from "@/ranger";

export const SET_CHANNEL_NAME = "sy_range_alias_set";
export const DELETE_CHANNEL_NAME = "sy_range_alias_delete";

/**
 * Aliaser provides channel alias operations scoped to a range.
 */
export class Aliaser {
  private readonly cache = new Map<string, channel.Key>();
  private readonly client: UnaryClient;
  private readonly rangeKey: ranger.Key;

  constructor(rangeKey: ranger.Key, client: UnaryClient) {
    this.rangeKey = rangeKey;
    this.cache = new Map();
    this.client = client;
  }

  /**
   * Resolve a single alias to its channel key.
   * @param alias - The alias to resolve.
   * @returns The channel key for the alias.
   */
  resolve(alias: string): Promise<channel.Key>;
  /**
   * Resolve multiple aliases to their channel keys.
   * @param aliases - The aliases to resolve.
   * @returns A record mapping aliases to channel keys.
   */
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
    const res = await sendRequired<typeof resolveRequestZ, typeof resolveResponseZ>(
      this.client,
      "/alias/resolve",
      { range: this.rangeKey, aliases: toFetch },
      resolveRequestZ,
      resolveResponseZ,
    );
    Object.entries(res.aliases).forEach(([alias, key]) => this.cache.set(alias, key));
    return isSingle ? res.aliases[toFetch[0]] : { ...cached, ...res.aliases };
  }

  /**
   * Set aliases for channels.
   * @param aliases - A record mapping channel keys to their aliases.
   */
  async set(aliases: Record<channel.Key, string>): Promise<void> {
    await sendRequired(
      this.client,
      "/alias/set",
      { range: this.rangeKey, aliases },
      setRequestZ,
      z.unknown(),
    );
  }

  /**
   * List all aliases for the range.
   * @returns A record mapping channel keys to aliases.
   */
  async list(): Promise<Record<channel.Key, string>> {
    return (
      await sendRequired<typeof listRequestZ, typeof listResponseZ>(
        this.client,
        "/alias/list",
        { range: this.rangeKey },
        listRequestZ,
        listResponseZ,
      )
    ).aliases;
  }

  /**
   * Retrieve the alias for a single channel.
   * @param channel - The channel key.
   * @returns The alias for the channel.
   */
  async retrieve(channel: channel.Key): Promise<string>;
  /**
   * Retrieve aliases for multiple channels.
   * @param channels - The channel keys.
   * @returns A record mapping channel keys to aliases.
   */
  async retrieve(channels: channel.Key[]): Promise<Record<channel.Key, string>>;
  async retrieve(
    alias: channel.Key | channel.Key[],
  ): Promise<string | Record<channel.Key, string>> {
    const isSingle = typeof alias === "number";
    const res = await sendRequired<typeof retrieveRequestZ, typeof retrieveResponseZ>(
      this.client,
      "/alias/retrieve",
      { range: this.rangeKey, channels: array.toArray(alias) },
      retrieveRequestZ,
      retrieveResponseZ,
    );
    return isSingle ? res.aliases[alias] : res.aliases;
  }

  /**
   * Delete aliases for one or more channels.
   * @param channels - The channel key or keys to delete aliases for.
   */
  async delete(channels: channel.Key | channel.Key[]): Promise<void> {
    await sendRequired(
      this.client,
      "/alias/delete",
      { range: this.rangeKey, channels: array.toArray(channels) },
      deleteRequestZ,
      z.unknown(),
    );
  }
}

/**
 * Client provides access to the Alias API.
 */
export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  /**
   * Get an Aliaser instance scoped to a specific range.
   * @param rangeKey - The range key to scope the alias operations to.
   * @returns An Aliaser instance for the specified range.
   */
  get(rangeKey: ranger.Key): Aliaser {
    return new Aliaser(rangeKey, this.client);
  }
}
