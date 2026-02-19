// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { TimeSpan } from "@synnaxlabs/x";
import { z } from "zod/v4";

export const MAKE = "http";
const makeZ = z.literal(MAKE);
const modelZ = z.literal("HTTP server");

const noneAuthConfigZ = z.object({ type: z.literal("none") });

const bearerAuthConfigZ = z.object({
  type: z.literal("bearer"),
  token: z.string().min(1, "Token is required"),
});

const apiKeyAuthConfigZ = z.object({
  type: z.literal("api_key"),
  header: z.string().min(1, "Header is required"),
  key: z.string().min(1, "Key is required"),
});

const basicAuthConfigZ = z.object({
  type: z.literal("basic"),
  username: z.string().min(1, "Username is required"),
  password: z.string().min(1, "Password is required"),
});

const authConfigZ = z.discriminatedUnion("type", [
  noneAuthConfigZ,
  bearerAuthConfigZ,
  apiKeyAuthConfigZ,
  basicAuthConfigZ,
]);

export type AuthConfig = z.infer<typeof authConfigZ>;

export type AuthType = AuthConfig["type"];

export const ZERO_AUTH_CONFIGS: Record<AuthType, AuthConfig> = {
  none: { type: "none" },
  bearer: { type: "bearer", token: "" },
  api_key: { type: "api_key", header: "", key: "" },
  basic: { type: "basic", username: "", password: "" },
};

const defaultTimeoutMs = TimeSpan.milliseconds(100).milliseconds;

const propertiesZ = z.object({
  secure: z.boolean().default(true),
  timeoutMs: z
    .number()
    .nonnegative("Timeout must be non-negative")
    .default(defaultTimeoutMs),
  auth: authConfigZ,
  headers: z.record(z.string(), z.string()).optional(),
});

export interface Properties extends z.infer<typeof propertiesZ> {}

export const ZERO_PROPERTIES = {
  secure: true,
  timeoutMs: defaultTimeoutMs,
  auth: ZERO_AUTH_CONFIGS.none,
} as const satisfies Properties;

export interface Device extends device.Device<
  typeof propertiesZ,
  typeof makeZ,
  typeof modelZ
> {}

export const SCHEMAS = {
  properties: propertiesZ,
  make: makeZ,
  model: modelZ,
} as const satisfies device.DeviceSchemas<
  typeof propertiesZ,
  typeof makeZ,
  typeof modelZ
>;
