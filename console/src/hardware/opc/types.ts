// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task, type device } from "@synnaxlabs/client";
import { z } from "zod";

export const connectionConfigZ = z.object({
  endpoint: z.string(),
  username: z.string().optional(),
  password: z.string().optional(),
});

export type ConnectionConfig = z.infer<typeof connectionConfigZ>;

export const deviceNodeProperties = z.object({
  dataType: z.string(),
  name: z.string(),
  nodeId: z.string(),
});

export type DeviceNodeProperties = z.infer<typeof deviceNodeProperties>;

export const devicePropertiesZ = z.object({
  connection: connectionConfigZ,
  channels: deviceNodeProperties.array(),
});

export type DeviceProperties = z.infer<typeof devicePropertiesZ>;

export type ReadTaskChannelConfig = z.infer<typeof readTaskChannelConfigZ>;

export const readTaskStateDetails = z.object({
  running: z.boolean(),
});

export type ReadTaskStateDetails = z.infer<typeof readTaskStateDetails>;

export type ReadTaskState = task.State<ReadTaskStateDetails>;

export const readTaskChannelConfigZ = z.object({
  key: z.string(),
  channel: z.number().min(1, "Channel must be specified"),
  nodeId: z.string().min(1, "Node ID must be specified"),
  enabled: z.boolean(),
});

export const readTaskConfigZ = z
  .object({
    device: z.string().min(1, "Device must be specified"),
    sampleRate: z.number().min(0).max(1000),
    streamRate: z.number().min(0).max(200),
    channels: z.array(readTaskChannelConfigZ),
  })
  .refine((cfg) => cfg.sampleRate >= cfg.streamRate, {
    message: "Sample rate must be greater than or equal to stream rate",
    path: ["sampleRate"],
  })
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
        params: {
          variant: "warning",
        },
      });
    });
  });

export type ReadTaskConfig = z.infer<typeof readTaskConfigZ>;

export type Device = device.Device<DeviceProperties>;

type NodeIdType = "Numeric" | "String" | "GUID" | "ByteString";

export interface NodeId {
  namespaceIndex: number;
  identifierType: NodeIdType;
  identifier: string | number; // Strings for String, GUID, and ByteString types, number for Numeric
}

export const parseNodeId = (nodeIdStr: string): NodeId | null => {
  const regex = /NS=(\d+);(I|S|G|B)=(.+)/;
  const match = nodeIdStr.match(regex);

  if (match === null) return null;

  const namespaceIndex = parseInt(match[1], 10);
  const typeCode = match[2];
  const identifier = match[3];

  let identifierType: NodeIdType;

  switch (typeCode) {
    case "I":
      identifierType = "Numeric";
      return {
        namespaceIndex,
        identifierType,
        identifier: parseInt(identifier, 10),
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

  return {
    namespaceIndex,
    identifierType,
    identifier,
  };
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
