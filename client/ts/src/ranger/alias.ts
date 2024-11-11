// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { type change } from "@synnaxlabs/x/change";
import { z } from "zod";

import { type channel } from "@/channel";
import { type Key as ChannelKey, keyZ as channelKeyZ } from "@/channel/payload";
import { type Client as FrameClient } from "@/framer/client";
import { type Key, keyZ } from "@/ranger/payload";
import { signals } from "@/signals";

export const ALIAS_SET_NAME = "sy_range_alias_set";
export const ALIAS_DELETE_NAME = "sy_range_alias_delete";

const resolveReqZ = z.object({
  range: keyZ,
  aliases: z.string().array(),
});

const resolveResZ = z.object({
  aliases: z.record(z.string(), channelKeyZ),
});

const setReqZ = z.object({
  range: keyZ,
  aliases: z.record(channelKeyZ.or(z.string()), z.string()),
});

const setResZ = z.unknown();

const deleteReqZ = z.object({
  range: keyZ,
  channels: channelKeyZ.array(),
});

const deleteResZ = z.unknown();

const listReqZ = z.object({
  range: keyZ,
});

const listResZ = z.object({
  aliases: z.record(z.string(), z.string()),
});

export class Aliaser {
  private static readonly SET_ENDPOINT = "/range/alias/set";
  private static readonly RESOLVE_ENDPOINT = "/range/alias/resolve";
  private static readonly LIST_ENDPOINT = "/range/alias/list";
  private static readonly DELETE_ENDPOINT = "/range/alias/delete";
  private readonly frameClient: FrameClient;
  private readonly cache = new Map<string, ChannelKey>();
  private readonly client: UnaryClient;
  private readonly rangeKey: Key;

  constructor(rangeKey: Key, frameClient: FrameClient, client: UnaryClient) {
    this.rangeKey = rangeKey;
    this.cache = new Map();
    this.client = client;
    this.frameClient = frameClient;
  }

  resolve(aliases: string): Promise<ChannelKey>;

  resolve(aliases: string[]): Promise<Record<string, ChannelKey>>;

  async resolve(
    aliases: string | string[],
  ): Promise<ChannelKey | Record<string, ChannelKey>> {
    const toFetch: string[] = [];
    const isSingle = typeof aliases === "string";
    const cached: Record<string, ChannelKey> = {};
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

  async set(aliases: Record<ChannelKey, string>): Promise<void> {
    await sendRequired<typeof setReqZ, typeof setResZ>(
      this.client,
      Aliaser.SET_ENDPOINT,
      { range: this.rangeKey, aliases },
      setReqZ,
      setResZ,
    );
  }

  async list(): Promise<Record<ChannelKey, string>> {
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

  async delete(aliases: ChannelKey[]): Promise<void> {
    await sendRequired<typeof deleteReqZ, typeof deleteResZ>(
      this.client,
      Aliaser.DELETE_ENDPOINT,
      { range: this.rangeKey, channels: aliases },
      deleteReqZ,
      deleteResZ,
    );
  }

  async openChangeTracker(): Promise<signals.Observable<string, Alias>> {
    return await signals.openObservable<string, Alias>(
      this.frameClient,
      ALIAS_SET_NAME,
      ALIAS_DELETE_NAME,
      decodeAliasChanges(this.rangeKey),
    );
  }
}

export interface Alias {
  range: Key;
  channel: channel.Key;
  alias: string;
}

export type AliasChange = change.Change<string, Alias>;

const aliasZ = z.object({
  range: keyZ,
  channel: channelKeyZ,
  alias: z.string(),
});

const aliasSeparator = "---";

const decodeAliasChanges =
  (rangeKey: Key): signals.Decoder<string, Alias> =>
  (variant, data) => {
    if (variant === "delete")
      return data
        .toStrings()
        .filter((k) => k.split(aliasSeparator)[0] === rangeKey)
        .map((alias) => ({
          variant,
          key: alias,
          value: undefined,
        }));

    return data.parseJSON(aliasZ).map((alias) => ({
      variant,
      key: alias.alias,
      value: alias,
    }));
  };
