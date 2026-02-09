// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type change } from "@synnaxlabs/x";
import { z } from "zod";

import { channel } from "@/channel";
import { type Key, keyZ } from "@/range/payload";

export const SET_CHANNEL_NAME = "sy_range_alias_set";
export const DELETE_CHANNEL_NAME = "sy_range_alias_delete";

export const resolveReqZ = z.object({ range: keyZ, aliases: z.string().array() });

export const resolveResZ = z.object({
  aliases: z.record(z.string(), channel.keyZ),
});

export const setReqZ = z.object({
  range: keyZ,
  aliases: z.record(channel.keyZ, z.string()),
});

export const setResZ = z.unknown();

export const deleteReqZ = z.object({ range: keyZ, channels: channel.keyZ.array() });

export const deleteResZ = z.unknown();

export const listReqZ = z.object({ range: keyZ });

export const listResZ = z.object({ aliases: z.record(z.string(), z.string()) });

export const retrieveReqZ = z.object({
  range: keyZ,
  channels: channel.keyZ.array(),
});

export const retrieveResZ = z.object({
  aliases: z.record(z.string(), z.string()),
});

export const aliasZ = z.object({
  alias: z.string().optional(),
  channel: channel.keyZ,
  range: keyZ,
});
export interface Alias extends z.infer<typeof aliasZ> {}

export type AliasChange = change.Change<string, Alias>;

const SEPARATOR = "---";

export const createKey = (alias: Pick<Alias, "range" | "channel">): string =>
  `${alias.range}${SEPARATOR}${alias.channel}`;

export interface DecodedDeleteAliasChange {
  range: Key;
  channel: channel.Key;
}

export const decodeDeleteChange = (deletedAlias: string): DecodedDeleteAliasChange => {
  const [range, channel] = deletedAlias.split(SEPARATOR);
  return { range, channel: Number(channel) };
};

/** @deprecated Use {@link SET_CHANNEL_NAME} instead. */
export const SET_ALIAS_CHANNEL_NAME = SET_CHANNEL_NAME;
/** @deprecated Use {@link DELETE_CHANNEL_NAME} instead. */
export const DELETE_ALIAS_CHANNEL_NAME = DELETE_CHANNEL_NAME;
/** @deprecated Use {@link createKey} instead. */
export const aliasKey = createKey;
/** @deprecated Use {@link decodeDeleteChange} instead. */
export const decodeDeleteAliasChange = decodeDeleteChange;
