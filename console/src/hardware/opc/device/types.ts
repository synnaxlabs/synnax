// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, task } from "@synnaxlabs/client";
import { UnknownRecord } from "@synnaxlabs/x";
import { z } from "zod";

export const securityModeZ = z.union([
  z.literal("None"),
  z.literal("Sign"),
  z.literal("SignAndEncrypt"),
]);

export type SecurityMode = z.infer<typeof securityModeZ>;

export const securityPolicyZ = z.union([
  z.literal("None"),
  z.literal("Basic128Rsa15"),
  z.literal("Basic256"),
  z.literal("Basic256Sha256"),
  z.literal("Aes128_Sha256_RsaOaep"),
  z.literal("Aes256_Sha256_RsaPss"),
]);

export type SecurityPolicy = z.infer<typeof securityPolicyZ>;

export const connectionConfigZ = z.object({
  endpoint: z.string(),
  username: z.string().optional(),
  password: z.string().optional(),
  security_mode: securityModeZ,
  security_policy: securityPolicyZ,
  client_certificate: z.string().optional(),
  client_private_key: z.string().optional(),
  server_certificate: z.string().optional(),
});

export type ConnectionConfig = z.infer<typeof connectionConfigZ>;

export const nodeProperties = z.object({
  dataType: z.string(),
  name: z.string(),
  nodeId: z.string(),
  nodeClass: z.string(),
  isArray: z.string(),
  synnaxChannel: z.number().optional(),
});

export type NodeProperties = z.infer<typeof nodeProperties>;

export const propertiesZ = z.object({
  connection: connectionConfigZ,
  channels: nodeProperties.array().optional(),
});

export type Properties = z.infer<typeof propertiesZ>;

export type Device = device.Device<Properties>;

export const scannerScanCommandResult = z.object({
  channels: nodeProperties.array(),
});

export type ScannerScanCommandResult = z.infer<typeof scannerScanCommandResult>;

export const channelConfigZ = nodeProperties
  .extend({
    key: z.string(),
    isIndex: z.boolean().optional(),
  })
  .superRefine((data, ctx) => {
    if (!data.isIndex && !data.nodeId)
      ctx.addIssue({
        code: "custom",
        path: ["nodeId"],
        message: "Data channels must have a node ID",
      });
    return true;
  })
  .transform((data) => {
    return data;
  })
  .superRefine((data, ctx) => {
    if (data.isIndex && data.dataType !== "timestamp")
      ctx.addIssue({
        code: "custom",
        path: ["dataType"],
        message: "Index channels must have a data type of timestamp",
      });
    return true;
  });

export type ChannelConfig = z.infer<typeof channelConfigZ>;

export const groupConfigZ = z
  .object({
    key: z.string(),
    name: z.string(),
    channels: channelConfigZ.array(),
  })
  .superRefine((data, ctx) => {
    const indexes: [ChannelConfig, number][] = [];
    data.channels.forEach((channel, i) => {
      if (channel.isIndex) indexes.push([channel, i]);
    });
    if (indexes.length > 1) {
      const found = indexes.map(([i]) => i.name).join(", ");
      ctx.addIssue({
        code: "custom",
        path: ["channels"],
        message: `Only one index channel is allowed per group, found: ${found}`,
      });
      indexes.forEach(([, i]) => {
        ctx.addIssue({
          code: "custom",
          path: ["channels", i],
          message: `Only one index channel is allowed per group, found: ${found}`,
        });
      });
    } else if (indexes.length === 0) {
      ctx.addIssue({
        code: "custom",
        path: ["channels"],
        message: "A group must have at least one index channel",
      });
    }
  });

export type GroupConfig = z.infer<typeof groupConfigZ>;

export interface TestConnCommandResponse extends UnknownRecord {
  message: string;
}

export type TestConnCommandState = task.State<TestConnCommandResponse>;
