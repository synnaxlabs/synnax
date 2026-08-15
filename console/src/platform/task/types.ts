// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, device } from "@synnaxlabs/client";
import { z } from "zod";

export type Command = "start" | "stop";

// Deploy-time only: shape schemas keep device lax so drafts round-trip.
export const deviceKeyZ = device.keyZ.min(1, "Must specify a device");

// The channel shape every generated task config carries.
export interface Channel {
  disabled: boolean;
  key: string;
}

export const validateChannels = ({
  value: channels,
  issues,
}: z.core.ParsePayload<{ key: string }[]>) => {
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

export const nameZ = z.string().default("");

// Spread over a duplicated channel to reset its Synnax channel bindings.
export const READ_CHANNEL_OVERRIDE = { channel: 0, name: "" } as const;
export const WRITE_CHANNEL_OVERRIDE = {
  cmdChannel: 0,
  stateChannel: 0,
  cmdChannelName: "",
  stateChannelName: "",
} as const;

export const validateChannelDevices = ({
  value: channels,
  issues,
}: z.core.ParsePayload<{ device: device.Key }[]>) => {
  channels.forEach(({ device }, i) => {
    if (device !== "") return;
    issues.push({
      code: "custom",
      message: "Must specify a device",
      path: [i, "device"],
      input: channels,
    });
  });
};

export const validateReadChannels = (
  ctx: z.core.ParsePayload<{ key: string; channel: channel.Key; name: string }[]>,
) => {
  validateChannels(ctx);
  const { value: channels, issues } = ctx;
  const channelToIndexMap = new Map<channel.Key, number>();
  const nameToIndexMap = new Map<string, number>();
  channels.forEach(({ channel, name }, i) => {
    if (name !== "") {
      if (!nameToIndexMap.has(name)) {
        nameToIndexMap.set(name, i);
        return;
      }
      const index = nameToIndexMap.get(name) as number;
      const code = "custom";
      const message = `Name ${name} is used for multiple channels`;
      issues.push({ code, message, path: [index, "name"], input: channels });
      issues.push({ code, message, path: [i, "name"], input: channels });
    }
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

export type WriteChannelType = "cmd" | "state";

interface IndexAndType {
  index: number;
  type: WriteChannelType;
}

export const validateWriteChannels = (
  ctx: z.core.ParsePayload<
    {
      key: string;
      cmdChannel: channel.Key;
      stateChannel: channel.Key;
      cmdChannelName: string;
      stateChannelName: string;
    }[]
  >,
) => {
  validateChannels(ctx);
  const { value: channels, issues } = ctx;
  const channelsToIndexMap = new Map<channel.Key, IndexAndType>();
  const nameToIndexMap = new Map<string, IndexAndType>();
  channels.forEach(
    ({ cmdChannel, stateChannel, cmdChannelName, stateChannelName }, i) => {
      if (cmdChannelName !== "") {
        if (!nameToIndexMap.has(cmdChannelName)) {
          nameToIndexMap.set(cmdChannelName, { index: i, type: "cmd" });
          return;
        }
        const { index, type } = nameToIndexMap.get(cmdChannelName) as IndexAndType;
        const code = "custom";
        const message = `Name ${cmdChannelName} is used for multiple channels`;
        issues.push({
          code,
          message,
          path: [index, `${type}ChannelName`],
          input: channels,
        });
        issues.push({ code, message, path: [i, "cmdChannelName"], input: channels });
      }
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
      if (stateChannelName !== "") {
        if (!nameToIndexMap.has(stateChannelName)) {
          nameToIndexMap.set(stateChannelName, { index: i, type: "state" });
          return;
        }
        const { index, type } = nameToIndexMap.get(stateChannelName) as IndexAndType;
        const code = "custom";
        const message = `Name ${stateChannelName} is used for multiple channels`;
        issues.push({
          code,
          message,
          path: [index, `${type}ChannelName`],
          input: channels,
        });
        issues.push({ code, message, path: [i, "stateChannelName"], input: channels });
      }
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
    },
  );
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
