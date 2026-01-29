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
import { type Key as RangeKey, keyZ as rangeKeyZ } from "@/range/types.gen";

export const resolveRequestZ = z.object({
  range: rangeKeyZ,
  aliases: z.string().array(),
});
export interface ResolveRequest extends z.infer<typeof resolveRequestZ> {}

export const resolveResponseZ = z.object({
  aliases: z.record(z.string(), channel.keyZ),
});
export interface ResolveResponse extends z.infer<typeof resolveResponseZ> {}

export const setRequestZ = z.object({
  range: rangeKeyZ,
  aliases: z.record(channel.keyZ, z.string()),
});
export interface SetRequest extends z.infer<typeof setRequestZ> {}

export const deleteRequestZ = z.object({
  range: rangeKeyZ,
  channels: channel.keyZ.array(),
});
export interface DeleteRequest extends z.infer<typeof deleteRequestZ> {}

export const listRequestZ = z.object({ range: rangeKeyZ });
export interface ListRequest extends z.infer<typeof listRequestZ> {}

export const listResponseZ = z.object({
  aliases: z.record(z.string(), z.string()),
});
export interface ListResponse extends z.infer<typeof listResponseZ> {}

export const retrieveRequestZ = z.object({
  range: rangeKeyZ,
  channels: channel.keyZ.array(),
});
export interface RetrieveRequest extends z.infer<typeof retrieveRequestZ> {}

export const retrieveResponseZ = z.object({
  aliases: z.record(z.string(), z.string()),
});
export interface RetrieveResponse extends z.infer<typeof retrieveResponseZ> {}

export const aliasZ = z.object({
  alias: z.string().optional(),
  channel: channel.keyZ,
  range: rangeKeyZ,
});
export interface Alias extends z.infer<typeof aliasZ> {}

export type AliasChange = change.Change<string, Alias>;

const SEPARATOR = "---";

export const createKey = (alias: Pick<Alias, "range" | "channel">): string =>
  `${alias.range}${SEPARATOR}${alias.channel}`;

export interface DecodedDeleteAliasChange {
  range: RangeKey;
  channel: channel.Key;
}

export const decodeDeleteAliasChange = (
  deletedAlias: string,
): DecodedDeleteAliasChange => {
  const [range, channel] = deletedAlias.split(SEPARATOR);
  return { range, channel: Number(channel) };
};
