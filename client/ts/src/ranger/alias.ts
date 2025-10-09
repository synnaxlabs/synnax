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
import { type change } from "@synnaxlabs/x/change";
import { z } from "zod";

import { channel } from "@/channel";
import { type framer } from "@/framer";
import { type Key, keyZ } from "@/ranger/payload";

export const SET_ALIAS_CHANNEL_NAME = "sy_range_alias_set";
export const DELETE_ALIAS_CHANNEL_NAME = "sy_range_alias_delete";

const resolveReqZ = z.object({ range: keyZ, aliases: z.string().array() });

const resolveResZ = z.object({ aliases: z.record(z.string(), channel.keyZ) });

const setReqZ = z.object({ range: keyZ, aliases: z.record(channel.keyZ, z.string()) });

const setResZ = z.unknown();

const deleteReqZ = z.object({ range: keyZ, channels: channel.keyZ.array() });

const deleteResZ = z.unknown();

const listReqZ = z.object({ range: keyZ });

const listResZ = z.object({ aliases: z.record(z.string(), z.string()) });

const retrieveReqZ = z.object({ range: keyZ, channels: channel.keyZ.array() });

const retrieveResZ = z.object({ aliases: z.record(z.string(), z.string()) });

export class Aliaser {
  private static readonly SET_ENDPOINT = "/range/alias/set";
  private static readonly RESOLVE_ENDPOINT = "/range/alias/resolve";
  private static readonly LIST_ENDPOINT = "/range/alias/list";
  private static readonly RETRIEVE_ENDPOINT = "/range/alias/retrieve";
  private static readonly DELETE_ENDPOINT = "/range/alias/delete";
  private readonly frameClient: framer.Client;
  private readonly cache = new Map<string, channel.Key>();
  private readonly client: UnaryClient;
  private readonly rangeKey: Key;

  constructor(rangeKey: Key, frameClient: framer.Client, client: UnaryClient) {
    this.rangeKey = rangeKey;
    this.cache = new Map();
    this.client = client;
    this.frameClient = frameClient;
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
      Aliaser.RESOLVE_ENDPOINT,
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
      Aliaser.SET_ENDPOINT,
      { range: this.rangeKey, aliases },
      setReqZ,
      setResZ,
    );
  }

  async list(): Promise<Record<channel.Key, string>> {
    return (
      await sendRequired<typeof listReqZ, typeof listResZ>(
        this.client,
        Aliaser.LIST_ENDPOINT,
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
      Aliaser.RETRIEVE_ENDPOINT,
      { range: this.rangeKey, channels: array.toArray(alias) },
      retrieveReqZ,
      retrieveResZ,
    );
    return isSingle ? res.aliases[alias] : res.aliases;
  }

  async delete(aliases: channel.Key | channel.Key[]): Promise<void> {
    await sendRequired<typeof deleteReqZ, typeof deleteResZ>(
      this.client,
      Aliaser.DELETE_ENDPOINT,
      { range: this.rangeKey, channels: array.toArray(aliases) },
      deleteReqZ,
      deleteResZ,
    );
  }
}

export const aliasZ = z.object({
  alias: z.string().optional(),
  channel: channel.keyZ,
  range: keyZ,
});
export interface Alias extends z.infer<typeof aliasZ> {}

export type AliasChange = change.Change<string, Alias>;

const SEPARATOR = "---";

export const aliasKey = (alias: Pick<Alias, "range" | "channel">): string =>
  `${alias.range}${SEPARATOR}${alias.channel}`;

export interface DecodedDeleteAliasChange {
  range: Key;
  channel: channel.Key;
}

export const decodeDeleteAliasChange = (
  deletedAlias: string,
): DecodedDeleteAliasChange => {
  const [range, channel] = deletedAlias.split(SEPARATOR);
  return { range, channel: Number(channel) };
};
