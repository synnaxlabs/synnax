// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, opc, type task } from "@synnaxlabs/client";
import { record } from "@synnaxlabs/x";
import { z } from "zod";

import { connectionConfigZ } from "@/feature/opc/device/types";
import { Task } from "@/platform/task";

export const PREFIX = "opc";

export interface InputChannel extends opc.InputChannel {}
export interface OutputChannel extends opc.OutputChannel {}
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

export interface ReadConfig extends opc.ReadConfig {}

export const readConfigZ = opc.readConfigZ;

const validateIndexChannels = ({
  value: channels,
  issues,
}: z.core.ParsePayload<InputChannel[]>) => {
  const indexChannelIndexes = channels
    .map(({ useAsIndex }, i) => (useAsIndex ? i : -1))
    .filter((i) => i !== -1);
  if (indexChannelIndexes.length <= 1) return;
  indexChannelIndexes.forEach((i) => {
    issues.push({
      code: "custom",
      message: "Only one channel can be marked as an index channel",
      path: ["channels", i, "useAsIndex"],
      input: channels,
    });
  });
};

export const deployReadConfigZ = opc.readConfigZ
  .extend({
    device: Task.deviceKeyZ,
    sampleRate: z.number().positive().max(10000),
    channels: opc.inputChannelZ
      .array()
      .check(Task.validateReadChannels)
      .check(validateNodeIDs)
      .check(validateIndexChannels),
  })
  .check((ctx) => {
    const { value, issues } = ctx;
    const { arrayMode, arraySize, sampleRate, streamRate } = value;
    if (arrayMode) {
      if (!Number.isInteger(arraySize) || arraySize <= 0)
        issues.push({
          code: "custom",
          message: "Array size must be a positive integer",
          path: ["arraySize"],
          input: value,
        });
      else if (sampleRate < arraySize)
        issues.push({
          code: "custom",
          message: "Sample rate must be greater than or equal to the array size",
          path: ["sampleRate"],
          input: value,
        });
      return;
    }
    if (streamRate <= 0 || streamRate > 10000)
      issues.push({
        code: "custom",
        message: "Stream rate must be between 0 and 10000",
        path: ["streamRate"],
        input: value,
      });
    else Task.validateStreamRate(ctx);
  });

const ZERO_READ_CONFIG = opc.readConfigZ.parse({ sampleRate: 50, streamRate: 25 });

export const READ_SCHEMAS = {
  type: z.literal(READ_TYPE),
  config: readConfigZ,
  statusData: z.unknown().optional(),
} as const satisfies task.Schemas;

export type ReadSchemas = typeof READ_SCHEMAS;

export interface ReadPayload extends task.Payload<ReadSchemas> {}

export const ZERO_READ_PAYLOAD: ReadPayload = {
  key: "",
  rack: 0,
  type: "opc_read",
  name: "OPC UA Read Task",
  config: ZERO_READ_CONFIG,
  configHash: "",
  internal: false,
  snapshot: false,
};

export const WRITE_TYPE = `${PREFIX}_write`;

export interface WriteConfig extends opc.WriteConfig {}

export const writeConfigZ = opc.writeConfigZ;

export const deployWriteConfigZ = opc.writeConfigZ.extend({
  device: Task.deviceKeyZ,
  channels: opc.outputChannelZ
    .array()
    .check(Task.validateChannels)
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
        issues.push({ code, message, path: [index, "cmdChannel"], input: channels });
        issues.push({ code, message, path: [i, "cmdChannel"], input: channels });
      });
    })
    .check(validateNodeIDs),
});

const ZERO_WRITE_CONFIG = opc.writeConfigZ.parse({});

export const WRITE_SCHEMAS = {
  type: z.literal(WRITE_TYPE),
  config: writeConfigZ,
  statusData: z.unknown().optional(),
} as const satisfies task.Schemas;

export type WriteSchemas = typeof WRITE_SCHEMAS;

export interface WritePayload extends task.Payload<WriteSchemas> {}

export const ZERO_WRITE_PAYLOAD: WritePayload = {
  key: "",
  rack: 0,
  type: "opc_write",
  name: "OPC UA Write Task",
  config: ZERO_WRITE_CONFIG,
  configHash: "",
  internal: false,
  snapshot: false,
};

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
  .nullish()
  .optional();

export const TEST_CONNECTION_COMMAND_TYPE = "test_connection";

export const BROWSE_COMMAND_TYPE = "browse";

export const SCAN_SCHEMAS = {
  type: z.literal(SCAN_TYPE),
  config: record.nullishToEmpty(),
  statusData: scanCommandResponseZ,
} as const satisfies task.Schemas;
