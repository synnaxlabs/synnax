// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type change } from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { type Alias } from "@/ranger/alias/types.gen";
import { type Key } from "@/ranger/types.gen";

export const SET_CHANNEL_NAME = "sy_range_alias_set";
export const DELETE_CHANNEL_NAME = "sy_range_alias_delete";

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
