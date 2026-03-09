// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, type device } from "@synnaxlabs/client";
import { json, TimeSpan } from "@synnaxlabs/x";
import { z } from "zod/v4";

export const MAKE = "http";
const makeZ = z.literal(MAKE);
const modelZ = z.literal("HTTP server");

const noneAuthConfigZ = z.object({ type: z.literal("none") });

const bearerAuthConfigZ = z.object({
  type: z.literal("bearer"),
  token: z.string().min(1, "Token is required"),
});

const v0APIKeyAuthConfigZ = z.object({
  type: z.literal("api_key"),
  header: z.string().min(1, "Header is required"),
  key: z.string().min(1, "Key is required"),
});

const baseAPIKeyAuthConfigZ = v0APIKeyAuthConfigZ.omit({ header: true });

const queryParamAPIKeyAuthConfigZ = baseAPIKeyAuthConfigZ.extend({
  sendAs: z.literal("query_param"),
  parameter: z.string().min(1, "Parameter is required"),
});

const headerAPIKeyAuthConfigZ = baseAPIKeyAuthConfigZ.extend({
  sendAs: z.literal("header"),
  header: z.string().min(1, "Header is required"),
});

const basicAuthConfigZ = z.object({
  type: z.literal("basic"),
  username: z.string().min(1, "Username is required"),
  password: z.string().min(1, "Password is required"),
});

const v0AuthConfigZ = z.discriminatedUnion("type", [
  noneAuthConfigZ,
  bearerAuthConfigZ,
  v0APIKeyAuthConfigZ,
  basicAuthConfigZ,
]);

const apiKeyAuthConfigZ = z.discriminatedUnion("sendAs", [
  queryParamAPIKeyAuthConfigZ,
  headerAPIKeyAuthConfigZ,
]);

export type APIKeyAuthConfigSendAs = z.infer<typeof apiKeyAuthConfigZ>["sendAs"];

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
  api_key: { type: "api_key", header: "", key: "", sendAs: "header" },
  basic: { type: "basic", username: "", password: "" },
};

export type JsonPrimitiveType = json.PrimitiveType;

const healthCheckResponseZ = z.object({
  pointer: json.pointerZ,
  expectedValueType: json.primitiveTypeZ,
  expectedValue: json.primitiveZ,
});

const healthCheckMethodZ = z.enum(["GET", "POST"]);

const healthCheckZ = z.object({
  method: healthCheckMethodZ,
  path: z.string().min(1, "Path is required"),
  headers: z.record(z.string(), z.string()).optional(),
  queryParams: z.record(z.string(), z.string()).optional(),
  body: z.string().optional(),
  validateResponse: z.boolean(),
  response: healthCheckResponseZ.optional(),
});

export type HealthCheck = z.infer<typeof healthCheckZ>;

export type HealthCheckMethod = z.infer<typeof healthCheckMethodZ>;

export const ZERO_HEALTH_CHECK: HealthCheck = {
  method: "GET",
  path: "/health",
  body: "",
  validateResponse: false,
  response: {
    pointer: "",
    expectedValueType: "string",
    expectedValue: "",
  },
};

const defaultTimeoutMs = TimeSpan.milliseconds(100).milliseconds;

const v0PropertiesZ = z.object({
  secure: z.boolean().default(true),
  verifySsl: z.boolean().default(true),
  timeoutMs: z
    .number()
    .nonnegative("Timeout must be non-negative")
    .default(defaultTimeoutMs),
  auth: v0AuthConfigZ,
  headers: z.record(z.string(), z.string()).optional(),
  queryParams: z.record(z.string(), z.string()).optional(),
  readIndexes: z.record(z.string(), channel.keyZ),
});

const v1PropertiesZ = v0PropertiesZ
  .omit({ auth: true, headers: true, queryParams: true })
  .extend({
    auth: authConfigZ,
    healthCheck: healthCheckZ.default(ZERO_HEALTH_CHECK),
    version: z.literal(1),
  });

export interface Properties extends z.infer<typeof v1PropertiesZ> {}

export const propertiesZ: z.ZodType<Properties> = v1PropertiesZ.or(
  v0PropertiesZ.transform((p) => {
    const { queryParams, auth, ...rest } = p;
    delete rest.headers;
    let newAuth: AuthConfig = { type: "none" };
    if (auth.type === "api_key")
      newAuth = {
        type: "api_key",
        sendAs: "header",
        header: auth.header,
        key: auth.key,
      };
    else if (auth.type === "none") {
      if (queryParams != null && Object.keys(queryParams).length > 0) {
        const [parameter, key] = Object.entries(queryParams)[0];
        newAuth = { type: "api_key", sendAs: "query_param", parameter, key };
      }
    } else newAuth = auth;
    return {
      ...rest,
      auth: newAuth,
      version: 1,
      healthCheck: ZERO_HEALTH_CHECK,
    } as const;
  }),
);

export const ZERO_PROPERTIES = {
  secure: true,
  verifySsl: true,
  timeoutMs: defaultTimeoutMs,
  auth: ZERO_AUTH_CONFIGS.none,
  healthCheck: ZERO_HEALTH_CHECK,
  readIndexes: {},
  version: 1,
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
