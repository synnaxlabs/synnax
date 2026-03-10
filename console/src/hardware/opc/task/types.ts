// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, type task } from "@synnaxlabs/client";
import { record } from "@synnaxlabs/x";
import { z } from "zod";

import { Common } from "@/hardware/common";
import { connectionConfigZ } from "@/hardware/opc/device/types";

export const PREFIX = "opc";

const baseChannelZ = Common.Task.channelZ.extend({
  channel: channel.keyZ,
  nodeId: z.string(),
  nodeName: z.string(),
  name: Common.Task.nameZ,
  dataType: z.string().default("float32"),
});

const inputChannelZ = baseChannelZ.extend({ useAsIndex: z.boolean() });

export interface InputChannel extends z.infer<typeof inputChannelZ> {}

const v0OutputChannelZ = baseChannelZ;

const v1OutputChannelZ = v0OutputChannelZ
  .omit({ channel: true })
  .extend({ cmdChannel: channel.keyZ });

const outputChannelZ = v0OutputChannelZ
  .transform(({ channel, ...rest }) => ({ ...rest, cmdChannel: channel }))
  .or(v1OutputChannelZ);

export type OutputChannel = z.infer<typeof outputChannelZ>;

export type Channel = InputChannel | OutputChannel;

const validateNodeIDs = ({
  value: channels,
  issues,
}: z.core.ParsePayload<Channel[]>) => {
  const nodeIds = new Map<string, number>();
  channels.forEach(({ nodeId }) => nodeIds.set(nodeId, (nodeIds.get(nodeId) ?? 0) + 1));
  channels.forEach(({ nodeId }, i) => {
    if (nodeId.length === 0 || (nodeIds.get(nodeId) ?? 0) < 2) return;
    issues.push({
      code: "custom",
      path: ["channels", i, "nodeId"],
      message: "This node ID has already been used elsewhere in the configuration",
      query: { variant: "warning" },
      input: channels,
    });
  });
};

export const READ_TYPE = `${PREFIX}_read`;

const baseReadConfigZ = Common.Task.baseReadConfigZ.extend({
  channels: z
    .array(inputChannelZ)
    .check(Common.Task.validateReadChannels)
    .check(validateNodeIDs)
    .check(({ value: channels, issues }) => {
      // Get indexes of channels that are marked as index channels
      const indexChannelIndexes = channels
        .map(({ useAsIndex }, i) => (useAsIndex ? i : -1))
        .filter((i) => i !== -1);
      if (indexChannelIndexes.length === 0 || indexChannelIndexes.length === 1) return;
      indexChannelIndexes.forEach((i) => {
        issues.push({
          code: "custom",
          message: "Only one channel can be marked as an index channel",
          path: ["channels", i, "useAsIndex"],
          input: channels,
        });
      });
    }),
  sampleRate: z.number().positive().max(10000),
});

const nonArraySamplingConfigZ = baseReadConfigZ
  .extend({ arrayMode: z.literal(false), streamRate: z.number().positive().max(10000) })
  .check(Common.Task.validateStreamRate);

const arraySamplingConfigZ = baseReadConfigZ
  .extend({ arrayMode: z.literal(true), arraySize: z.number().int().positive() })
  .refine(({ arraySize, sampleRate }) => sampleRate >= arraySize, {
    message: "Sample rate must be greater than or equal to the array size",
    path: ["sampleRate"],
  });

const readConfigZ = z.discriminatedUnion("arrayMode", [
  nonArraySamplingConfigZ,
  arraySamplingConfigZ,
]);

export type ReadConfig = z.infer<typeof readConfigZ>;

const ZERO_READ_CONFIG: ReadConfig = {
  ...Common.Task.ZERO_BASE_READ_CONFIG,
  arrayMode: false,
  channels: [],
  sampleRate: 50,
  streamRate: 25,
} as const satisfies ReadConfig;

export const READ_SCHEMAS = {
  typeSchema: z.literal(READ_TYPE),
  configSchema: readConfigZ,
  statusDataSchema: z.unknown(),
} as const satisfies task.Schemas;

export type ReadSchemas = typeof READ_SCHEMAS;

export interface ReadPayload extends task.Payload<ReadSchemas> {}

export const ZERO_READ_PAYLOAD = {
  key: "",
  type: "opc_read",
  name: "OPC UA Read Task",
  config: ZERO_READ_CONFIG,
} as const satisfies ReadPayload;

export const WRITE_TYPE = `${PREFIX}_write`;

const writeConfigZ = Common.Task.baseConfigZ.extend({
  channels: z
    .array(outputChannelZ)
    .check(Common.Task.validateChannels)
    .check(({ value: channels, issues }) => {
      // Have to have a separate validation here as OPC UA write channels do not have
      // a stateChannel key.
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
        issues.push({ code, message, path: [index, "cmdChannel"], input: channels });
        issues.push({ code, message, path: [i, "cmdChannel"], input: channels });
      });
    })
    .check(validateNodeIDs),
});

interface WriteConfig extends z.infer<typeof writeConfigZ> {}

export const ZERO_WRITE_CONFIG = {
  ...Common.Task.ZERO_BASE_CONFIG,
  channels: [],
} as const satisfies WriteConfig;

export const WRITE_SCHEMAS = {
  typeSchema: z.literal(WRITE_TYPE),
  configSchema: writeConfigZ,
  statusDataSchema: z.unknown(),
} as const satisfies task.Schemas;

export type WriteSchemas = typeof WRITE_SCHEMAS;

export interface WritePayload extends task.Payload<WriteSchemas> {}

export const ZERO_WRITE_PAYLOAD = {
  key: "",
  type: "opc_write",
  name: "OPC UA Write Task",
  config: ZERO_WRITE_CONFIG,
} as const satisfies WritePayload;

export const SCAN_TYPE = `${PREFIX}_scan`;

const scannedNodeZ = z
  .object({
    key: z.string().optional(),
    dataType: z.string(),
    isArray: z.boolean(),
    name: z.string(),
    nodeClass: z.string(),
    nodeId: z.string(),
  })
  .transform(({ key, ...rest }) => ({ ...rest, key: key ?? rest.nodeId }));

export interface ScannedNode extends z.infer<typeof scannedNodeZ> {}

const scanCommandResponseZ = z
  .object({ channels: z.array(scannedNodeZ), connection: connectionConfigZ })
  .or(z.null());

export const TEST_CONNECTION_COMMAND_TYPE = "test_connection";

export const BROWSE_COMMAND_TYPE = "browse";

export const SCAN_SCHEMAS = {
  typeSchema: z.literal(SCAN_TYPE),
  configSchema: record.nullishToEmpty(),
  statusDataSchema: scanCommandResponseZ,
} as const satisfies task.Schemas;
