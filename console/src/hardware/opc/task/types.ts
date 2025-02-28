// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";
import { z } from "zod";

import { Common } from "@/hardware/common";

export const PREFIX = "opc";

// Channels

// Read Channels

export const readChannelZ = z.object({
  key: z.string(),
  name: z.string(),
  channel: z.number(),
  nodeName: z.string(),
  nodeId: z.string(),
  enabled: z.boolean(),
  useAsIndex: z.boolean(),
  dataType: z.string(),
});
export interface ReadChannel extends z.infer<typeof readChannelZ> {}

// Write Channels

export const writeChannelZ = z.object({
  key: z.string(),
  name: z.string(),
  cmdChannel: z.number(),
  nodeName: z.string(),
  nodeId: z.string(),
  enabled: z.boolean(),
  dataType: z.string(),
});
export interface WriteChannel extends z.infer<typeof writeChannelZ> {}

// Tasks

// Read Task

const baseReadConfigZ = z.object({
  device: Common.Device.keyZ,
  sampleRate: z.number().positive().max(10000),
  channels: z.array(readChannelZ).superRefine(Common.Task.validateReadChannels),
  dataSaving: z.boolean().optional().default(true),
});

const nonArraySamplingConfig = baseReadConfigZ
  .and(
    z.object({
      arrayMode: z.literal(false),
      streamRate: z.number().positive().max(10000),
    }),
  )
  .refine(Common.Task.validateStreamRate);

const arraySamplingConfig = baseReadConfigZ
  .and(
    z.object({
      arrayMode: z.literal(true),
      arraySize: z.number().int().positive(),
    }),
  )
  .refine(({ sampleRate, arraySize }) => sampleRate >= arraySize, {
    message: "Sample rate must be greater than or equal to the array size",
    path: ["sampleRate"],
  });

export const newReadConfigZ = z.union([nonArraySamplingConfig, arraySamplingConfig]);
export type NewReadConfig = z.infer<typeof newReadConfigZ>;

export const readConfigZ = z
  .object({
    device: Common.Device.keyZ,
    sampleRate: z.number().positive().max(10000),
    streamRate: z.number().positive().max(10000).default(1),
    arrayMode: z.boolean(),
    arraySize: z.number().int().positive().default(1),
    channels: z
      .array(readChannelZ)
      .length(1, "Must specify at least one channel")
      .superRefine(Common.Task.validateReadChannels),
    dataSaving: z.boolean().optional().default(true),
  })
  .transform((cfg) => {
    if (cfg.arrayMode) cfg.streamRate = 1;
    else cfg.arraySize = 1;
    return cfg;
  })
  .refine(
    ({ arrayMode, sampleRate, streamRate }) => arrayMode || sampleRate >= streamRate,
    {
      message: "Stream rate must be less than or equal to the sample rate",
      path: ["streamRate"],
    },
  )
  .refine(
    ({ arrayMode, arraySize, sampleRate }) => !arrayMode || sampleRate >= arraySize,
    {
      message: "Sample rate must be greater than or equal to the array size",
      path: ["sampleRate"],
    },
  )
  .refine(({ arrayMode, streamRate }) => arrayMode || streamRate > 0, {
    message: "Stream rate must be greater than or equal to 1",
    path: ["streamRate"],
  })
  // Error if channel ahs been duplicated

  // Warning if node ID is duplicated
  .superRefine((cfg, ctx) => {
    const nodeIds = new Map<string, number>();
    cfg.channels.forEach(({ nodeId }) =>
      nodeIds.set(nodeId, (nodeIds.get(nodeId) ?? 0) + 1),
    );
    cfg.channels.forEach(({ nodeId }, i) => {
      if (nodeId.length === 0 || (nodeIds.get(nodeId) ?? 0) < 2) return;
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ["channels", i, "nodeId"],
        message: "This node ID has already been used elsewhere in the configuration",
        params: { variant: "warning" },
      });
    });
  });

export type ReadConfig = z.infer<typeof readConfigZ>;

export const readStateDetails = z.object({
  running: z.boolean().optional().default(false),
  message: z.string().optional(),
});
export type ReadStateDetails = z.infer<typeof readStateDetails>;
export type ReadState = task.State<ReadStateDetails>;

export const READ_TYPE = `${PREFIX}_read`;
export type ReadType = typeof READ_TYPE;

export type ReadTask = task.Task<ReadConfig, ReadStateDetails, ReadType>;
export type ReadPayload = task.Payload<ReadConfig, ReadStateDetails, ReadType>;
export const ZERO_READ_PAYLOAD: ReadPayload = {
  key: "",
  type: READ_TYPE,
  name: "OPC UA Read Task",
  config: {
    device: "",
    sampleRate: 50,
    streamRate: 25,
    arrayMode: false,
    arraySize: 1,
    channels: [],
    dataSaving: true,
  },
};

// Scan Task

export const SCAN_NAME = "opc Scanner";
export const SCAN_COMMAND_TYPE = "scan";

export interface ScannedNode {
  nodeId: string;
  dataType: string;
  name: string;
  nodeClass: string;
  isArray: boolean;
}

export const TEST_CONNECTION_COMMAND = "test_connection";
export type TestConnectionCommandResponse = { message: string };

export interface TestConnectionCommandState
  extends task.State<TestConnectionCommandResponse> {}

export type ScanCommandResult = { channels: ScannedNode[] };

// Write Task

export const writeConfigZ = z
  .object({
    device: z.string().min(1, "Device must be specified"),
    channels: z.array(writeChannelZ),
    dataSaving: z.boolean().optional().default(true),
  })
  // Error if channel has been duplicated
  .superRefine((cfg, ctx) => {
    const channels = new Map<number, number>();
    cfg.channels.forEach(({ cmdChannel }) =>
      channels.set(cmdChannel, (channels.get(cmdChannel) ?? 0) + 1),
    );
    cfg.channels.forEach(({ cmdChannel }, i) => {
      if (cmdChannel === 0 || (channels.get(cmdChannel) ?? 0) < 2) return;
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ["channels", i, "channel"],
        message: "This channel has already been used elsewhere in the configuration",
      });
    });
  })
  // Warning if node ID is duplicated
  .superRefine((cfg, ctx) => {
    const nodeIds = new Map<string, number>();
    cfg.channels.forEach(({ nodeId }) =>
      nodeIds.set(nodeId, (nodeIds.get(nodeId) ?? 0) + 1),
    );
    cfg.channels.forEach(({ nodeId }, i) => {
      if (nodeId.length === 0 || (nodeIds.get(nodeId) ?? 0) < 2) return;
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ["channels", i, "nodeId"],
        message: "This node ID has already been used elsewhere in the configuration",
        params: { variant: "warning" },
      });
    });
  });
export interface WriteConfig extends z.infer<typeof writeConfigZ> {}
export const ZERO_WRITE_CONFIG: WriteConfig = {
  device: "",
  channels: [],
  dataSaving: true,
};

export interface WriteStateDetails {
  running: boolean;
  message?: string;
}
export interface WriteState extends task.State<WriteStateDetails> {}

export const WRITE_TYPE = `${PREFIX}_write`;
export type WriteType = typeof WRITE_TYPE;

export type WritePayload = task.Payload<WriteConfig, WriteStateDetails, WriteType>;
export const ZERO_WRITE_PAYLOAD: WritePayload = {
  key: "",
  type: WRITE_TYPE,
  name: "OPC UA Write Task",
  config: ZERO_WRITE_CONFIG,
};

export interface WriteTask
  extends task.Task<WriteConfig, WriteStateDetails, WriteType> {}
export interface NewWriteTask extends task.New<WriteConfig, WriteType> {}
