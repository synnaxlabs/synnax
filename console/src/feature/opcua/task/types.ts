// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, opcua, type task } from "@synnaxlabs/client";
import { map } from "@synnaxlabs/x";
import { z } from "zod";

import { Device } from "@/feature/opcua/device";
import { Task } from "@/platform/task";

export const PREFIX = "opc";

export interface ReadChannel extends opcua.ReadChannel {}
export interface WriteChannel extends opcua.WriteChannel {}
export type Channel = ReadChannel | WriteChannel;

const validateNodeIDs = ({
  value: channels,
  issues,
}: z.core.ParsePayload<Channel[]>) => {
  const counts = new Map<string, number>();
  channels.forEach((ch) => {
    const id = Device.nodeIdentity(ch);
    map.increment(counts, id);
  });
  channels.forEach((ch, i) => {
    if (ch.nodeId.length === 0 || (counts.get(Device.nodeIdentity(ch)) ?? 0) < 2)
      return;
    issues.push({
      continue: true,
      code: "custom",
      path: [i, "nodeId"],
      message:
        "field" in ch && ch.field !== ""
          ? "This field has already been used elsewhere in the configuration"
          : "This node ID has already been used elsewhere in the configuration",
      params: { variant: "warning" },
      input: channels,
    });
  });
};

export const READ_TYPE = `${PREFIX}_read`;

export interface ReadConfig extends opcua.ReadConfig {}

const readConfigZ = opcua.readConfigZ;

const validateIndexChannels = ({
  value: channels,
  issues,
}: z.core.ParsePayload<ReadChannel[]>) => {
  const indexChannelIndexes = channels
    .map(({ isIndex }, i) => (isIndex ? i : -1))
    .filter((i) => i !== -1);
  if (indexChannelIndexes.length <= 1) return;
  indexChannelIndexes.forEach((i) => {
    issues.push({
      continue: true,
      code: "custom",
      message: "Only one channel can be marked as an index channel",
      path: [i, "isIndex"],
      input: channels,
    });
  });
};

const ARRAY_SIZE_ERROR = "Array size must be a positive integer";

const deploySamplingZ = z.discriminatedUnion("type", [
  opcua.unarySamplingZ,
  opcua.arraySamplingZ.extend({
    size: z.uint32({ error: ARRAY_SIZE_ERROR }).min(1, { error: ARRAY_SIZE_ERROR }),
  }),
]);

export const deployReadConfigZ = opcua.readConfigZ
  .extend({
    device: Task.deviceKeyZ,
    sampleRate: z.number().positive().max(10000),
    sampling: deploySamplingZ,
    channels: opcua.readChannelZ
      .array()
      .check(Task.validateReadChannels)
      .check(Task.validateEnabledChannel)
      .check(validateNodeIDs)
      .check(validateIndexChannels),
  })
  .check((ctx) => {
    const { value, issues } = ctx;
    const { sampling, channels, sampleRate, streamRate } = value;
    if (sampling.type === "array") {
      channels.forEach(({ disabled, field }, i) => {
        if (disabled || field === "") return;
        issues.push({
          continue: true,
          code: "custom",
          message: "Array sampling cannot read structure fields",
          path: ["channels", i, "nodeId"],
          input: value,
        });
      });
      if (sampleRate < sampling.size)
        issues.push({
          continue: true,
          code: "custom",
          message: "Sample rate must be greater than or equal to the array size",
          path: ["sampleRate"],
          input: value,
        });
      return;
    }
    if (streamRate <= 0 || streamRate > 10000)
      issues.push({
        continue: true,
        code: "custom",
        message: "Stream rate must be between 0 and 10000",
        path: ["streamRate"],
        input: value,
      });
    else Task.validateStreamRate(ctx);
  });

export const READ_SCHEMAS = {
  type: z.literal(READ_TYPE),
  config: readConfigZ,
  statusData: z.unknown().optional(),
} as const satisfies task.Schemas;

export type ReadSchemas = typeof READ_SCHEMAS;

export const WRITE_TYPE = `${PREFIX}_write`;

const writeConfigZ = opcua.writeConfigZ;

export const deployWriteConfigZ = opcua.writeConfigZ.extend({
  device: Task.deviceKeyZ,
  channels: opcua.writeChannelZ
    .array()
    .check(Task.validateChannels)
    .check(Task.validateEnabledChannel)
    .check(({ value: channels, issues }) => {
      const channelsToIndexMap = new Map<channel.Key, number>();
      channels.forEach(({ cmdChannel }, i) => {
        if (cmdChannel === 0) return;
        if (!channelsToIndexMap.has(cmdChannel)) {
          channelsToIndexMap.set(cmdChannel, i);
          return;
        }
        const index = channelsToIndexMap.get(cmdChannel) as number;
        const code = "custom";
        const message = `Synnax channel with key ${cmdChannel} is used for multiple channels`;
        issues.push({
          continue: true,
          code,
          message,
          path: [index, "cmdChannel"],
          input: channels,
        });
        issues.push({
          continue: true,
          code,
          message,
          path: [i, "cmdChannel"],
          input: channels,
        });
      });
    })
    .check(validateNodeIDs),
});

export const WRITE_SCHEMAS = {
  type: z.literal(WRITE_TYPE),
  config: writeConfigZ,
  statusData: z.unknown().optional(),
} as const satisfies task.Schemas;

export type WriteSchemas = typeof WRITE_SCHEMAS;
