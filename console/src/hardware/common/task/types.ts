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

export const channelZ = z.object({ enabled: z.boolean(), key: z.string() });
export interface Channel extends z.infer<typeof channelZ> {}
export const ZERO_CHANNEL: Channel = { enabled: true, key: "" };

export const validateChannels = (
  channels: Channel[],
  { addIssue }: z.RefinementCtx,
) => {
  if (channels.length === 0) {
    addIssue({
      code: z.ZodIssueCode.custom,
      message: "At least one channel must be specified",
    });
    return;
  }
  if (!channels.some(({ enabled }) => enabled))
    addIssue({
      code: z.ZodIssueCode.custom,
      message: "At least one channel must be enabled",
    });
  const keyToIndexMap = new Map<string, number>();
  channels.forEach(({ key }, i) => {
    if (!keyToIndexMap.has(key)) {
      keyToIndexMap.set(key, i);
      return;
    }
    const index = keyToIndexMap.get(key) as number;
    const issueBasics = {
      code: z.ZodIssueCode.custom,
      message: `Key ${key} is used for multiple channels`,
    };
    addIssue({ ...issueBasics, path: [index, "key"] });
    addIssue({ ...issueBasics, path: [i, "key"] });
  });
};

export const readChannelZ = channelZ.extend({ channel: channel.keyZ });
export interface ReadChannel extends z.infer<typeof readChannelZ> {}
export const ZERO_READ_CHANNEL: ReadChannel = { ...ZERO_CHANNEL, channel: 0 };

export const validateReadChannels = (channels: ReadChannel[], ctx: z.RefinementCtx) => {
  validateChannels(channels, ctx);
  const channelToIndexMap = new Map<channel.Key, number>();
  const { addIssue } = ctx;
  channels.forEach(({ channel }, i) => {
    if (channel === 0) return;
    if (!channelToIndexMap.has(channel)) {
      channelToIndexMap.set(channel, i);
      return;
    }
    const index = channelToIndexMap.get(channel) as number;
    const baseIssue = {
      code: z.ZodIssueCode.custom,
      message: `Synnax channel with key ${channel} is used for multiple channels`,
    };
    addIssue({ ...baseIssue, path: [index, "channel"] });
    addIssue({ ...baseIssue, path: [i, "channel"] });
  });
};

export const writeChannelZ = channelZ.extend({
  cmdChannel: channel.keyZ,
  stateChannel: channel.keyZ,
});
export interface WriteChannel extends z.infer<typeof writeChannelZ> {}
export const ZERO_WRITE_CHANNEL: WriteChannel = {
  ...ZERO_CHANNEL,
  cmdChannel: 0,
  stateChannel: 0,
};

interface IndexAndType {
  index: number;
  type: "cmd" | "state";
}

export const validateWriteChannels = (
  channels: WriteChannel[],
  ctx: z.RefinementCtx,
) => {
  validateChannels(channels, ctx);
  const channelsToIndexMap = new Map<channel.Key, IndexAndType>();
  const { addIssue } = ctx;
  channels.forEach(({ cmdChannel, stateChannel }, i) => {
    if (cmdChannel !== 0)
      if (channelsToIndexMap.has(cmdChannel)) {
        const { index, type } = channelsToIndexMap.get(cmdChannel) as IndexAndType;
        const baseIssue = {
          code: z.ZodIssueCode.custom,
          message: `Synnax channel with key ${cmdChannel} is used on multiple channels`,
        };
        addIssue({ ...baseIssue, path: [index, `${type}Channel`] });
        addIssue({ ...baseIssue, path: [i, "cmdChannel"] });
      } else channelsToIndexMap.set(cmdChannel, { index: i, type: "cmd" });
    if (stateChannel === 0) return;
    if (channelsToIndexMap.has(stateChannel)) {
      const { index, type } = channelsToIndexMap.get(stateChannel) as IndexAndType;
      const baseIssue = {
        code: z.ZodIssueCode.custom,
        message: `Synnax channel with key ${stateChannel} is used for multiple channels`,
      };
      addIssue({ ...baseIssue, path: [index, `${type}Channel`] });
      addIssue({ ...baseIssue, path: [i, "stateChannel"] });
    } else channelsToIndexMap.set(stateChannel, { index: i, type: "state" });
  });
};

interface ConfigWithSampleRateAndStreamRate {
  sampleRate: number;
  streamRate: number;
}

export const validateStreamRate = ({
  sampleRate,
  streamRate,
}: ConfigWithSampleRateAndStreamRate) =>
  [
    sampleRate >= streamRate,
    {
      path: ["streamRate"],
      message: "Stream rate must be less than or equal to the sample rate",
    },
  ] as const;
