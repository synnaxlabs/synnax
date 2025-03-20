// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, type task } from "@synnaxlabs/client";
import { z } from "zod";

import { Common } from "@/hardware/common";
import { type Device } from "@/hardware/opc/device";

export const PREFIX = "opc";

const baseChannelZ = Common.Task.channelZ.extend({
  name: z.string(),
  nodeId: z.string(),
  nodeName: z.string(),
});

const readChannelZ = baseChannelZ.extend({
  channel: channel.keyZ,
  dataType: z.string(),
  useAsIndex: z.boolean(),
});
export interface ReadChannel extends z.infer<typeof readChannelZ> {}

const v0WriteChannelZ = baseChannelZ.extend({
  channel: channel.keyZ,
  dataType: z.string(),
});

const v1WriteChannelZ = v0WriteChannelZ
  .omit({ channel: true })
  .extend({ cmdChannel: channel.keyZ });

const writeChannelZ = v0WriteChannelZ
  .transform(({ channel, ...rest }) => ({ ...rest, cmdChannel: channel }))
  .or(v1WriteChannelZ);

export type WriteChannel = z.infer<typeof writeChannelZ>;

export type Channel = ReadChannel | WriteChannel;

const validateNodeIDs = (channels: Channel[], { addIssue }: z.RefinementCtx) => {
  const nodeIds = new Map<string, number>();
  channels.forEach(({ nodeId }) => nodeIds.set(nodeId, (nodeIds.get(nodeId) ?? 0) + 1));
  channels.forEach(({ nodeId }, i) => {
    if (nodeId.length === 0 || (nodeIds.get(nodeId) ?? 0) < 2) return;
    addIssue({
      code: z.ZodIssueCode.custom,
      path: ["channels", i, "nodeId"],
      message: "This node ID has already been used elsewhere in the configuration",
      params: { variant: "warning" },
    });
  });
};

interface BaseStateDetails {
  message?: string;
  running: boolean;
}

const baseReadConfigZ = Common.Task.baseConfigZ.extend({
  channels: z
    .array(readChannelZ)
    .superRefine(Common.Task.validateReadChannels)
    .superRefine(validateNodeIDs)
    .superRefine((channels, { addIssue }) => {
      // Get indexes of channels that are marked as index channels
      const indexChannelIndexes = channels
        .map(({ useAsIndex }, i) => (useAsIndex ? i : -1))
        .filter((i) => i !== -1);
      if (indexChannelIndexes.length === 0 || indexChannelIndexes.length === 1) return;
      const baseIssue = {
        code: z.ZodIssueCode.custom,
        message: "Only one channel can be marked as an index channel",
      };
      indexChannelIndexes.forEach((i) => {
        addIssue({ ...baseIssue, path: ["channels", i, "useAsIndex"] });
      });
    }),
  sampleRate: z.number().positive().max(10000),
});

const nonArraySamplingConfigZ = baseReadConfigZ
  .extend({
    arrayMode: z.literal(false),
    streamRate: z.number().positive().max(10000),
  })
  .superRefine(Common.Task.validateStreamRate);

const arraySamplingConfigZ = baseReadConfigZ
  .extend({
    arrayMode: z.literal(true),
    arraySize: z.number().int().positive(),
  })
  .refine(({ arraySize, sampleRate }) => sampleRate >= arraySize, {
    message: "Sample rate must be greater than or equal to the array size",
    path: ["sampleRate"],
  });

export const readConfigZ = z.union([nonArraySamplingConfigZ, arraySamplingConfigZ]);
export type ReadConfig = z.infer<typeof readConfigZ>;
const ZERO_READ_CONFIG: ReadConfig = {
  ...Common.Task.ZERO_BASE_CONFIG,
  arrayMode: false,
  channels: [],
  sampleRate: 50,
  streamRate: 25,
};

export interface ReadStateDetails extends BaseStateDetails {}
export interface ReadState extends task.State<ReadStateDetails> {}

export const READ_TYPE = `${PREFIX}_read`;
export type ReadType = typeof READ_TYPE;

export interface ReadPayload
  extends task.Payload<ReadConfig, ReadStateDetails, ReadType> {}
export const ZERO_READ_PAYLOAD: ReadPayload = {
  key: "",
  type: READ_TYPE,
  name: "OPC UA Read Task",
  config: ZERO_READ_CONFIG,
};

export interface ReadTask extends task.Task<ReadConfig, ReadStateDetails, ReadType> {}
export interface NewReadTask extends task.New<ReadConfig, ReadType> {}

export type ScanConfig = {};
export const ZERO_SCAN_CONFIG: ScanConfig = {};

export const SCAN_COMMAND_TYPE = "scan";

export interface ScannedNode {
  dataType: string;
  isArray: boolean;
  name: string;
  nodeClass: string;
  nodeId: string;
}
export interface ScanCommandResponse {
  channels: ScannedNode[];
  connection: Device.ConnectionConfig;
}

export const TEST_CONNECTION_COMMAND_TYPE = "test_connection";
export interface TestConnectionCommandResponse {
  message: string;
}

export type ScanStateDetails = ScanCommandResponse | TestConnectionCommandResponse;
export interface ScanState extends task.State<ScanStateDetails> {}

export interface TestConnectionCommandState
  extends task.State<TestConnectionCommandResponse> {}

export const SCAN_TYPE = `${PREFIX}_scan`;
export type ScanType = typeof SCAN_TYPE;

export interface ScanPayload
  extends task.Payload<ScanConfig, ScanStateDetails, ScanType> {}
export interface ScanTask extends task.Task<ScanConfig, ScanStateDetails, ScanType> {}
export interface NewScanTask extends task.New<ScanConfig, ScanType> {}

export const writeConfigZ = Common.Task.baseConfigZ.extend({
  channels: z
    .array(writeChannelZ)
    .superRefine(Common.Task.validateChannels)
    .superRefine((channels, { addIssue }) => {
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
        const baseIssue = {
          code: z.ZodIssueCode.custom,
          message: `Synnax channel with key ${cmdChannel} is used for multiple channels`,
        };
        addIssue({ ...baseIssue, path: [index, "cmdChannel"] });
        addIssue({ ...baseIssue, path: [i, "cmdChannel"] });
      });
    })
    .superRefine(validateNodeIDs),
});
export type WriteConfig = z.infer<typeof writeConfigZ>;
export const ZERO_WRITE_CONFIG: WriteConfig = {
  ...Common.Task.ZERO_BASE_CONFIG,
  channels: [],
};

export interface WriteStateDetails extends BaseStateDetails {}
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
