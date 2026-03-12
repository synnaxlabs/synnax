// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel } from "@synnaxlabs/client";
import { Log } from "@synnaxlabs/pluto";
import { z } from "zod";

const VERSION = "0.0.0";

export const { channelConfigZ } = Log;
export type ChannelConfig = z.infer<typeof Log.channelConfigZ>;

export const ZERO_CHANNEL_CONFIG: ChannelConfig = { color: "", precision: -1 };

export const stateZ = z.object({
  key: z.string(),
  version: z.literal(VERSION),
  channels: channel.keyZ.array(),
  remoteCreated: z.boolean(),
  timestampPrecision: z.number().min(0).max(3).default(0),
  channelConfigs: z.record(z.string(), channelConfigZ).default({}),
});

export type State = z.infer<typeof stateZ>;

export const ZERO_STATE: State = {
  key: "",
  version: VERSION,
  channels: [],
  remoteCreated: false,
  timestampPrecision: 0,
  channelConfigs: {},
};

export const sliceStateZ = z.object({
  version: z.literal(VERSION),
  logs: z.record(z.string(), stateZ),
});

export type SliceState = z.infer<typeof sliceStateZ>;

export const ZERO_SLICE_STATE: SliceState = { version: VERSION, logs: {} };
