// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, device, type task } from "@synnaxlabs/client";
import { type Task } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import { type z } from "zod";

/** The values of a form over a task with the schemas S. */
export type Values<S extends task.Schemas = task.Schemas> = z.infer<Task.FormSchema<S>>;

/** The path to the values of any task form. */
export const formPath = deep.path<Values>();

export type Command = "start" | "stop";

// Deploy-time only: shape schemas keep device lax so drafts round-trip.
export const deviceKeyZ = device.keyZ.min(1, "Device is required");

// The channel shape every generated task config carries.
export interface Channel {
  disabled: boolean;
  key: string;
}

const enabledFirst = <C extends Channel>(channels: C[]): C[] =>
  channels.toSorted((a, b) => Number(a.disabled) - Number(b.disabled));

/**
 * @returns the first channel on each port, an enabled one if any. Channels on one port
 * share its data channel, and deploying the task creates it from this one.
 */
export const firstPerPort = <C extends Channel>(
  channels: C[],
  portOf: (channel: C) => string,
): C[] => {
  const seen = new Set<string>();
  return enabledFirst(channels).filter((c) => {
    const port = portOf(c);
    if (seen.has(port)) return false;
    seen.add(port);
    return true;
  });
};

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

/**
 * @returns the channel that an added channel copies: the channel with the given key, or
 * else the last channel. Returns undefined when there are no channels.
 */
export const channelToCopy = <C extends Channel>(
  channels: C[],
  copy?: string,
): C | undefined => channels.find(({ key }) => key === copy) ?? channels.at(-1);

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
      message: "Device is required",
      path: [i, "device"],
      input: channels,
    });
  });
};

export const validateEnabledChannel = ({
  value: channels,
  issues,
}: z.core.ParsePayload<Channel[]>) => {
  if (channels.some(({ disabled }) => !disabled)) return;
  issues.push({
    code: "custom",
    message: "At least one enabled channel is required",
    input: channels,
  });
};

export const validateReadChannels = (
  ctx: z.core.ParsePayload<
    { key: string; channel: channel.Key; name: string; disabled: boolean }[]
  >,
) => {
  validateChannels(ctx);
  const { value: channels, issues } = ctx;
  const channelToIndexMap = new Map<channel.Key, number>();
  const nameToIndexMap = new Map<string, number>();
  channels.forEach(({ channel, name, disabled }, i) => {
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
    // The Driver skips a disabled channel, so it can share a Synnax channel.
    if (channel === 0 || disabled) return;
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
    // A channel type with no Synnax channels has none of the channel fields.
    {
      key: string;
      cmdChannel?: channel.Key;
      stateChannel?: channel.Key;
      cmdChannelName?: string;
      stateChannelName?: string;
      disabled: boolean;
    }[]
  >,
) => {
  validateChannels(ctx);
  const { value: channels, issues } = ctx;
  const channelsToIndexMap = new Map<channel.Key, IndexAndType>();
  const nameToIndexMap = new Map<string, IndexAndType>();
  channels.forEach(
    (
      {
        cmdChannel = 0,
        stateChannel = 0,
        cmdChannelName = "",
        stateChannelName = "",
        disabled,
      },
      i,
    ) => {
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
      // The Driver skips a disabled channel, so it can share a Synnax channel.
      if (cmdChannel !== 0 && !disabled)
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
      if (stateChannel === 0 || disabled) return;
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
