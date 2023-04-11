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

export const channelPayloadSchema = z.object({
  key: z.string(),
  rate: z.number().transform((n) => new Rate(n)),
  dataType: z.string().transform((s) => new DataType(s)),
  name: z.string(),
  nodeKey: z.number().default(0).optional(),
  index: z.string().default("").optional(),
  isIndex: z.boolean().default(false).optional(),
});

export type ChannelPayload = z.infer<typeof channelPayloadSchema>;

export const unkeyedChannelPayloadSchema = channelPayloadSchema.extend({
  key: z.string().optional(),
});

export type UnkeyedChannelPayload = z.infer<typeof unkeyedChannelPayloadSchema>;

export interface UnparsedChannel {
  key?: string;
  name: string;
  dataType: UnparsedDataType;
  rate?: UnparsedRate;
  nodeKey?: number;
  index?: string;
  isIndex?: boolean;
}

export const parseChannels = (channels: UnparsedChannel[]): UnkeyedChannelPayload[] =>
  channels.map((channel) => ({
    name: channel.name,
    dataType: new DataType(channel.dataType),
    rate: new Rate(channel.rate ?? 0),
    nodeKey: channel.nodeKey,
    index: channel.index,
    isIndex: channel.isIndex,
  }));
