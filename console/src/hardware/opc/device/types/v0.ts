// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type task } from "@synnaxlabs/client";
import { z } from "zod";

export const VERSION = "0.0.0";
type Version = typeof VERSION;

export const MAKE = "opc";

const securityModeZ = z.union([
  z.literal("None"),
  z.literal("Sign"),
  z.literal("SignAndEncrypt"),
]);
export type SecurityMode = z.infer<typeof securityModeZ>;

const securityPolicyZ = z.union([
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

type ScannedNode = {
  nodeId: string;
  dataType: string;
  name: string;
  nodeClass: string;
  isArray: boolean;
};

export type Properties = {
  version: Version;
  connection: ConnectionConfig;
  read: { index: channel.Key; channels: Record<string, channel.Key> };
  write: { channels: Record<string, channel.Key> };
};
export const ZERO_PROPERTIES: Properties = {
  version: VERSION,
  connection: ZERO_CONNECTION_CONFIG,
  read: { index: 0, channels: {} },
  write: { channels: {} },
};

export type TestConnectionCommandResponse = { message: string };

export interface TestConnectionCommandState
  extends task.State<TestConnectionCommandResponse> {}

export type ScanCommandResult = { channels: ScannedNode[] };
