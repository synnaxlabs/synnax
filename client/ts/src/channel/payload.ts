// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Rate } from "@synnaxlabs/x";
import { z } from "zod";

export type ChannelKey = number;
export type ChannelKeys = number[];
export type ChannelName = string;
export type ChannelNames = string[];
export type ChannelKeyOrName = ChannelKey | ChannelName;
export type ChannelKeysOrNames = ChannelKeys | ChannelNames;
export type ChannelParams = ChannelKey | ChannelName | ChannelKeys | ChannelNames;

export const channelPayload = z.object({
  name: z.string(),
  key: z.number(),
  rate: Rate.z,
  dataType: DataType.z,
  leaseholder: z.number(),
  index: z.number(),
  isIndex: z.boolean(),
});

export type ChannelPayload = z.infer<typeof channelPayload>;

export const newChannelPayload = channelPayload.extend({
  key: z.number().optional(),
  leaseholder: z.number().optional(),
  index: z.number().optional(),
  rate: Rate.z.optional(),
  isIndex: z.boolean().optional(),
});

export type NewChannelPayload = z.input<typeof newChannelPayload>;

export const parseChannels = (channels: NewChannelPayload[]): NewChannelPayload[] =>
  channels.map((channel) => ({
    name: channel.name,
    dataType: new DataType(channel.dataType),
    rate: new Rate(channel.rate ?? 0),
    leaseholder: channel.leaseholder,
    index: channel.index,
    isIndex: channel.isIndex,
  }));
