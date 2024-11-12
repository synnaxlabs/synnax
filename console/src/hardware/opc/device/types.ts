// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, type task } from "@synnaxlabs/client";
import { type UnknownRecord } from "@synnaxlabs/x";
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
  securityMode: securityModeZ,
  securityPolicy: securityPolicyZ,
  clientCertificate: z.string().optional(),
  clientPrivateKey: z.string().optional(),
  serverCertificate: z.string().optional(),
});

export type ConnectionConfig = z.infer<typeof connectionConfigZ>;

export const scannedNodeZ = z.object({
  nodeId: z.string(),
  dataType: z.string(),
  name: z.string(),
  nodeClass: z.string(),
  isArray: z.boolean(),
});

export type ScannedNode = z.infer<typeof scannedNodeZ>;

export const propertiesZ = z.object({
  connection: connectionConfigZ,
  read: z.object({
    index: z.number(),
    channels: z.record(z.string(), z.number()),
  }),
  write: z.object({
    channels: z.record(z.string(), z.number()),
  }),
});

export type Properties = z.infer<typeof propertiesZ>;

export type Device = device.Device<Properties>;

export interface TestConnCommandResponse extends UnknownRecord {
  message: string;
}

export type TestConnCommandState = task.State<TestConnCommandResponse>;

export const scannerScanCommandResult = z.object({
  channels: scannedNodeZ.array(),
});

export type ScannerScanCommandResult = z.infer<typeof scannerScanCommandResult>;
