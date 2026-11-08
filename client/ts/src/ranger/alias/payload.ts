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
import { type resource } from "@/resource";

export const SET_CHANNEL_NAME = "sy_range_alias_set";
export const DELETE_CHANNEL_NAME = "sy_range_alias_delete";

export type AliasChange = change.Change<string, Alias>;

const SEPARATOR = "---";

export const createKey = (alias: Pick<Alias, "range" | "channel">): string =>
  `${alias.range}${SEPARATOR}${alias.channel}`;

export const resourceID = (rangeKey: Key, channelKey: channel.Key): resource.ID => ({
  type: "range-alias",
  key: createKey({ range: rangeKey, channel: channelKey }),
});

export const TYPE_RESOURCE_ID: resource.ID = { type: "range-alias", key: "" };

export interface DecodedDeleteAliasChange {
  range: Key;
  channel: channel.Key;
}

export const decodeDeleteChange = (deletedAlias: string): DecodedDeleteAliasChange => {
  const [range, channel] = deletedAlias.split(SEPARATOR);
  return { range, channel: Number(channel) };
};
