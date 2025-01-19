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

export const VERSION = "0.0.0";

export const MAKE = "opc";

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
export interface ConnectionConfig extends z.infer<typeof connectionConfigZ> {}
export const ZERO_CONNECTION_CONFIG: ConnectionConfig = {
  endpoint: "opc.tcp://localhost:4840",
  securityMode: "None",
  securityPolicy: "None",
  username: "",
  password: "",
  clientCertificate: "",
  clientPrivateKey: "",
  serverCertificate: "",
};

export const scannedNodeZ = z.object({
  nodeId: z.string(),
  dataType: z.string(),
  name: z.string(),
  nodeClass: z.string(),
  isArray: z.boolean(),
});
export interface ScannedNode extends z.infer<typeof scannedNodeZ> {}

export const propertiesZ = z.object({
  version: z.literal(VERSION).optional().default(VERSION),
  connection: connectionConfigZ,
  read: z.object({ index: z.number(), channels: z.record(z.string(), z.number()) }),
  write: z.object({ channels: z.record(z.string(), z.number()) }),
});

export interface Properties extends z.infer<typeof propertiesZ> {}

export interface Device extends device.Device<Properties> {}

export interface TestConnCommandResponse extends UnknownRecord {
  message: string;
}

export interface TestConnCommandState extends task.State<TestConnCommandResponse> {}

export const scannerScanCommandResult = z.object({ channels: scannedNodeZ.array() });

export interface ScannerScanCommandResult
  extends z.infer<typeof scannerScanCommandResult> {}
