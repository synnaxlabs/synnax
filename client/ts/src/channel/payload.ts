// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Rate, UnparsedDataType, UnparsedRate } from "@synnaxlabs/x";
import { z } from "zod";

export type ChannelKey = number;
export type ChannelKeys = number[];
export type ChannelName = string;
export type ChannelNames = string[];
export type ChannelKeyOrName = ChannelKey | ChannelName;
export type ChannelKeysOrNames = ChannelKeys | ChannelNames;
export type ChannelParams =
  | ChannelKey
  | ChannelName
  | ChannelKeys
  | ChannelNames
  | Array<ChannelKey | ChannelName>;

export const channelPayload = z.object({
  key: z.number(),
  rate: Rate.z,
  dataType: DataType.z,
  name: z.string(),
  leaseholder: z.number().default(0).optional(),
  index: z.string().default("").optional(),
  isIndex: z.boolean().default(false).optional(),
});

export type ChannelPayload = z.infer<typeof channelPayload>;

export const unkeyedChannelPayload = channelPayload.extend({
  key: z.number().optional(),
});

export type UnkeyedChannelPayload = z.infer<typeof unkeyedChannelPayload>;

export interface UnparsedChannel {
  key?: number;
  name: string;
  dataType: UnparsedDataType;
  rate?: UnparsedRate;
  leaseholder?: number;
  index?: string;
  isIndex?: boolean;
}

export const parseChannels = (channels: UnparsedChannel[]): UnkeyedChannelPayload[] =>
  channels.map((channel) => ({
    name: channel.name,
    dataType: new DataType(channel.dataType),
    rate: new Rate(channel.rate ?? 0),
    leaseholder: channel.leaseholder,
    index: channel.index,
    isIndex: channel.isIndex,
  }));
