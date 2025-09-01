// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel } from "@synnaxlabs/client";
import { z } from "zod";

import { Device } from "@/hardware/common/device";

export type Command = "start" | "stop";

export const channelZ = z.object({ enabled: z.boolean(), key: z.string() });
export interface Channel extends z.infer<typeof channelZ> {}
export const ZERO_CHANNEL: Channel = { enabled: true, key: "" };

export const validateChannels = ({
  value: channels,
  issues,
}: z.core.ParsePayload<Channel[]>) => {
  const keyToIndexMap = new Map<string, number>();
  channels.forEach(({ key }, i) => {
    if (!keyToIndexMap.has(key)) {
      keyToIndexMap.set(key, i);
      return;
    }
    const index = keyToIndexMap.get(key) as number;
    const code = "custom";
    const message = `Key ${key} is used for multiple channels`;
    issues.push({ code, message, path: [index, "key"], input: channels });
    issues.push({ code, message, path: [i, "key"], input: channels });
  });
};

export const readChannelZ = channelZ.extend({ channel: channel.keyZ });
export interface ReadChannel extends z.infer<typeof readChannelZ> {}

export const READ_CHANNEL_OVERRIDE: Pick<ReadChannel, "channel"> = { channel: 0 };

export const ZERO_READ_CHANNEL: ReadChannel = {
  ...ZERO_CHANNEL,
  ...READ_CHANNEL_OVERRIDE,
};

export const validateReadChannels = (ctx: z.core.ParsePayload<ReadChannel[]>) => {
  validateChannels(ctx);
  const { value: channels, issues } = ctx;
  const channelToIndexMap = new Map<channel.Key, number>();
  channels.forEach(({ channel }, i) => {
    if (channel === 0) return;
    if (!channelToIndexMap.has(channel)) {
      channelToIndexMap.set(channel, i);
      return;
    }
    const index = channelToIndexMap.get(channel) as number;
    const code = "custom";
    const message = `Synnax channel with key ${channel} is used for multiple channels`;
    issues.push({ code, message, path: [index, "channel"], input: channels });
    issues.push({ code, message, path: [i, "channel"], input: channels });
  });
};

export const writeChannelZ = channelZ.extend({
  cmdChannel: channel.keyZ,
  stateChannel: channel.keyZ,
});
export interface WriteChannel extends z.infer<typeof writeChannelZ> {}
export const WRITE_CHANNEL_OVERRIDE: Pick<WriteChannel, "cmdChannel" | "stateChannel"> =
  { cmdChannel: 0, stateChannel: 0 };
export const ZERO_WRITE_CHANNEL: WriteChannel = {
  ...ZERO_CHANNEL,
  ...WRITE_CHANNEL_OVERRIDE,
};

export type WriteChannelType = "cmd" | "state";

interface IndexAndType {
  index: number;
  type: WriteChannelType;
}

export const validateWriteChannels = (ctx: z.core.ParsePayload<WriteChannel[]>) => {
  validateChannels(ctx);
  const { value: channels, issues } = ctx;
  const channelsToIndexMap = new Map<channel.Key, IndexAndType>();
  channels.forEach(({ cmdChannel, stateChannel }, i) => {
    if (cmdChannel !== 0)
      if (channelsToIndexMap.has(cmdChannel)) {
        const { index, type } = channelsToIndexMap.get(cmdChannel) as IndexAndType;
        const code = "custom";
        const message = `Synnax channel with key ${cmdChannel} is used on multiple channels`;
        issues.push({
          code,
          message,
          path: [index, `${type}Channel`],
          input: channels,
        });
        issues.push({ code, message, path: [i, "cmdChannel"], input: channels });
      } else channelsToIndexMap.set(cmdChannel, { index: i, type: "cmd" });
    if (stateChannel === 0) return;
    if (channelsToIndexMap.has(stateChannel)) {
      const { index, type } = channelsToIndexMap.get(stateChannel) as IndexAndType;
      const code = "custom";
      const message = `Synnax channel with key ${stateChannel} is used for multiple channels`;
      issues.push({
        code,
        message,
        path: [index, `${type}Channel`],
        input: channels,
      });
      issues.push({ code, message, path: [i, "stateChannel"], input: channels });
    } else channelsToIndexMap.set(stateChannel, { index: i, type: "state" });
  });
};

export const baseConfigZ = z.object({
  autoStart: z.boolean().default(false),
  device: Device.keyZ,
});
export interface BaseConfig extends z.infer<typeof baseConfigZ> {}
export const ZERO_BASE_CONFIG: BaseConfig = {
  autoStart: false,
  device: "",
};

export const baseReadConfigZ = baseConfigZ.extend({
  dataSaving: z.boolean().default(true),
});
export interface BaseReadConfig extends z.infer<typeof baseReadConfigZ> {}
export const ZERO_BASE_READ_CONFIG: BaseReadConfig = {
  ...ZERO_BASE_CONFIG,
  dataSaving: true,
};

interface ConfigWithSampleRateAndStreamRate {
  sampleRate: number;
  streamRate: number;
}
export const validateStreamRate = (
  ctx: z.core.ParsePayload<ConfigWithSampleRateAndStreamRate>,
) => {
  const {
    value: { sampleRate, streamRate },
    issues,
  } = ctx;
  if (sampleRate < streamRate)
    issues.push({
      code: "custom",
      message: "Stream rate must be less than or equal to the sample rate",
      path: ["streamRate"],
      input: { sampleRate, streamRate },
    });
};
