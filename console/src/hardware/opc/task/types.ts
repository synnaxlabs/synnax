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

export const PREFIX = "opc";

// Reads
export const READ_TYPE = `${PREFIX}_read`;
export type ReadType = typeof READ_TYPE;

export type ReadChannelConfig = z.infer<typeof readChanZ>;

export const readStateDetails = z.object({
  running: z.boolean().optional().default(false),
  message: z.string().optional(),
});

export type ReadStateDetails = z.infer<typeof readStateDetails>;
export type ReadState = task.State<ReadStateDetails>;

export const readChanZ = z.object({
  key: z.string(),
  name: z.string(),
  channel: z.number(),
  nodeName: z.string(),
  nodeId: z.string(),
  enabled: z.boolean(),
  useAsIndex: z.boolean(),
  dataType: z.string(),
});

export const readConfigZ = z
  .object({
    device: z.string().min(1, "Device must be specified"),
    sampleRate: z.number().min(1).max(10000),
    streamRate: z.number(),
    arrayMode: z.boolean(),
    arraySize: z.number().min(1),
    channels: z.array(readChanZ),
    dataSaving: z.boolean().optional().default(true),
  })
  .refine(
    (cfg) => {
      if (cfg.arrayMode) return true;
      return cfg.sampleRate >= cfg.streamRate;
    },
    {
      message: "Sample rate must be greater than or equal to stream rate",
      path: ["sampleRate"],
    },
  )
  .refine(
    (cfg) => {
      if (!cfg.arrayMode) return true;
      return cfg.sampleRate >= cfg.arraySize;
    },
    {
      message: "Sample rate must be greater than or equal to the array size",
      path: ["sampleRate"],
    },
  )
  .refine(
    (cfg) => {
      if (cfg.arrayMode) return true;
      return cfg.streamRate > 0;
    },
    { message: "Stream rate must be greater than or equal to 1", path: ["streamRate"] },
  )
  // Error if channel ahs been duplicated
  .superRefine((cfg, ctx) => {
    const channels = new Map<number, number>();
    cfg.channels.forEach(({ channel }) =>
      channels.set(channel, (channels.get(channel) ?? 0) + 1),
    );
    cfg.channels.forEach(({ channel }, i) => {
      if (channel === 0 || (channels.get(channel) ?? 0) < 2) return;
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
  })
  .transform((cfg) => {
    if (!cfg.arrayMode) cfg.arraySize = 1;
    return cfg;
  });

export type ReadConfig = z.infer<typeof readConfigZ>;
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

export interface NodeId {
  namespaceIndex: number;
  identifierType: NodeIdType;
  identifier: string | number; // Strings for String, GUID, and ByteString types, number for Numeric
}

export const parseNodeId = (nodeIdStr: string): NodeId | null => {
  const regex = /NS=(\d+);(I|S|G|B)=(.+)/;
  const match = nodeIdStr.match(regex);

  if (match === null) return null;

  const namespaceIndex = parseInt(match[1]);
  const typeCode = match[2];
  const identifier = match[3];

  let identifierType: NodeIdType;

  switch (typeCode) {
    case "I":
      identifierType = "Numeric";
      return {
        namespaceIndex,
        identifierType,
        identifier: parseInt(identifier),
      };
    case "S":
      identifierType = "String";
      break;
    case "G":
      identifierType = "GUID";
      break;
    case "B":
      identifierType = "ByteString";
      break;
    default:
      return null;
  }

  return { namespaceIndex, identifierType, identifier };
};

export const nodeIdToString = (nodeId: NodeId): string => {
  const prefix = `NS=${nodeId.namespaceIndex};`;
  switch (nodeId.identifierType) {
    case "Numeric":
      return `${prefix}I=${nodeId.identifier}`;
    case "String":
    case "GUID":
    case "ByteString":
      return `${prefix}${nodeId.identifierType.charAt(0)}=${nodeId.identifier}`;
  }
};

// Writes
export const WRITE_TYPE = `${PREFIX}_write`;
export type WriteType = typeof WRITE_TYPE;

export type WriteChannelConfig = z.infer<typeof writeChanZ>;

export const writeStateDetails = z.object({
  running: z.boolean().optional().default(false),
  message: z.string().optional(),
});

export type WriteStateDetails = z.infer<typeof writeStateDetails>;
export type WriteState = task.State<WriteStateDetails>;

export const writeChanZ = z.object({
  key: z.string(),
  name: z.string(),
  cmdChannel: z.number(),
  nodeName: z.string(),
  nodeId: z.string(),
  enabled: z.boolean(),
  dataType: z.string(),
});

export const writeConfigZ = z
  .object({
    device: z.string().min(1, "Device must be specified"),
    channels: z.array(writeChanZ),
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

export type WriteConfig = z.infer<typeof writeConfigZ>;
export type WritePayload = task.Payload<WriteConfig, WriteStateDetails, WriteType>;
export const ZERO_WRITE_PAYLOAD: WritePayload = {
  key: "",
  type: WRITE_TYPE,
  name: "OPC UA Write Task",
  config: { device: "", channels: [], dataSaving: true },
};

type NodeIdType = "Numeric" | "String" | "GUID" | "ByteString";

// scan task

export const SCAN_NAME = "opc Scanner";
export const SCAN_COMMAND_NAME = "scan";

type ScannedNode = {
  nodeId: string;
  dataType: string;
  name: string;
  nodeClass: string;
  isArray: boolean;
};

export const TEST_CONNECTION_COMMAND = "test_connection";
export type TestConnectionCommandResponse = { message: string };

export interface TestConnectionCommandState
  extends task.State<TestConnectionCommandResponse> {}

export type ScanCommandResult = { channels: ScannedNode[] };
