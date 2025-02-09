// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { z } from "zod";

export const VERSION = "0.0.0";
type Version = typeof VERSION;

export const MAKE = "opc";
export type Make = typeof MAKE;

export const NO_SECURITY_MODE = "None";
export const SIGN_SECURITY_MODE = "Sign";
export const SIGN_AND_ENCRYPT_SECURITY_MODE = "SignAndEncrypt";
const securityModeZ = z.enum([
  NO_SECURITY_MODE,
  SIGN_SECURITY_MODE,
  SIGN_AND_ENCRYPT_SECURITY_MODE,
]);
export type SecurityMode = z.infer<typeof securityModeZ>;

export const NO_SECURITY_POLICY = "None";
export const BASIC128_RSA15_SECURITY_POLICY = "Basic128Rsa15";
export const BASIC256_SECURITY_POLICY = "Basic256";
export const BASIC256_SHA256_SECURITY_POLICY = "Basic256Sha256";
export const AES128_SHA256_RSAOAEP_SECURITY_POLICY = "Aes128_Sha256_RsaOaep";
export const AES256_SHA256_RSAPSS_SECURITY_POLICY = "Aes256_Sha256_RsaPss";
const securityPolicyZ = z.enum([
  NO_SECURITY_POLICY,
  BASIC128_RSA15_SECURITY_POLICY,
  BASIC256_SECURITY_POLICY,
  BASIC256_SHA256_SECURITY_POLICY,
  AES128_SHA256_RSAOAEP_SECURITY_POLICY,
  AES256_SHA256_RSAPSS_SECURITY_POLICY,
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
  securityMode: NO_SECURITY_MODE,
  securityPolicy: NO_SECURITY_POLICY,
  username: "",
  password: "",
  clientCertificate: "",
  clientPrivateKey: "",
  serverCertificate: "",
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
