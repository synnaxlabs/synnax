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

export const headerEntryZ = z.object({ name: z.string(), value: z.string() });
export interface HeaderEntry extends z.infer<typeof headerEntryZ> {}

export const queryParamEntryZ = z.object({ parameter: z.string(), value: z.string() });
export interface QueryParamEntry extends z.infer<typeof queryParamEntryZ> {}

export const checkDuplicateKeys =
  (keyField: string, label: string) =>
  (ctx: { value: Record<string, unknown>[] | undefined; issues: unknown[] }) => {
    if (ctx.value == null) return;
    const seen = new Set<unknown>();
    ctx.value.forEach((entry, i) => {
      const k = entry[keyField];
      if (k === "") return;
      if (seen.has(k))
        ctx.issues.push({
          code: "custom",
          input: ctx.value,
          message: `Duplicate ${label} "${String(k)}"`,
          path: [i, keyField],
        });
      else seen.add(k);
    });
  };

const v0HeadersZ = z.record(z.string(), z.string());
const v1HeadersZ = z.array(headerEntryZ);
export const headersZ: z.ZodType<HeaderEntry[]> = v1HeadersZ
  .or(
    v0HeadersZ.transform((rec) =>
      Object.entries(rec).map(([name, value]) => ({ name, value })),
    ),
  )
  .check(checkDuplicateKeys("name", "header"));

const v0QueryParamsZ = z.record(z.string(), z.string());
const v1QueryParamsZ = z.array(queryParamEntryZ);
export const queryParamsZ: z.ZodType<QueryParamEntry[]> = v1QueryParamsZ
  .or(
    v0QueryParamsZ.transform((rec) =>
      Object.entries(rec).map(([parameter, value]) => ({ parameter, value })),
    ),
  )
  .check(checkDuplicateKeys("parameter", "query parameter"));

const sharedHealthCheckZ = z.object({
  path: z.string(),
  headers: headersZ.optional(),
  queryParams: queryParamsZ.optional(),
});

const noValidateHealthCheckZ = sharedHealthCheckZ.extend({
  validateResponse: z.literal(false),
});

const baseResponseZ = z.object({ pointer: json.pointerZ });

const stringResponseValueZ = baseResponseZ.extend({
  expectedValueType: z.literal("string"),
  expectedValue: z.string(),
});

const numberResponseValueZ = baseResponseZ.extend({
  expectedValueType: z.literal("number"),
  expectedValue: z.number(),
});

const booleanResponseValueZ = baseResponseZ.extend({
  expectedValueType: z.literal("boolean"),
  expectedValue: z.boolean(),
});

const nullResponseValueZ = baseResponseZ.extend({
  expectedValueType: z.literal("null"),
  expectedValue: z.null(),
});

const responseZ = z.discriminatedUnion("expectedValueType", [
  stringResponseValueZ,
  numberResponseValueZ,
  booleanResponseValueZ,
  nullResponseValueZ,
]);

export type Response = z.infer<typeof responseZ>;

export const ZERO_RESPONSE = {
  pointer: "",
  expectedValueType: "string",
  expectedValue: "",
} as const satisfies Response;

const validateHealthCheckZ = sharedHealthCheckZ.extend({
  validateResponse: z.literal(true),
  response: responseZ,
});

const getShapeZ = { method: z.literal("GET") } as const;

const getHealthCheckZ = z.discriminatedUnion("validateResponse", [
  noValidateHealthCheckZ.extend(getShapeZ),
  validateHealthCheckZ.extend(getShapeZ),
]);

const postShapeZ = { method: z.literal("POST"), body: z.string().optional() } as const;

const postHealthCheckZ = z.discriminatedUnion("validateResponse", [
  noValidateHealthCheckZ.extend(postShapeZ),
  validateHealthCheckZ.extend(postShapeZ),
]);

export const healthCheckZ = z.discriminatedUnion("method", [
  getHealthCheckZ,
  postHealthCheckZ,
]);

export type HealthCheck = z.infer<typeof healthCheckZ>;

export const ZERO_HEALTH_CHECK = {
  method: "GET",
  path: "",
  validateResponse: false,
} as const satisfies HealthCheck;

export type HealthCheckMethod = HealthCheck["method"];

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

const readEndpointPropsZ = z.object({
  index: channel.keyZ,
  channels: z.record(z.string(), channel.keyZ),
});

interface ReadEndpointProps extends z.infer<typeof readEndpointPropsZ> {}

const v1PropertiesZ = v0PropertiesZ
  .omit({ auth: true, headers: true, queryParams: true, readIndexes: true })
  .extend({
    auth: authConfigZ,
    healthCheck: healthCheckZ.default(ZERO_HEALTH_CHECK),
    write: z.record(z.string(), channel.keyZ).default({}),
    read: z.record(z.string(), readEndpointPropsZ).default({}),
    version: z.literal(1),
  });

export interface Properties extends z.infer<typeof v1PropertiesZ> {}

export const propertiesZ: z.ZodType<Properties> = v1PropertiesZ.or(
  v0PropertiesZ.transform((p) => {
    const { queryParams, auth, readIndexes, ...rest } = p;
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
    const read: Record<string, ReadEndpointProps> = {};
    for (const [path, indexKey] of Object.entries(readIndexes))
      read[path] = { index: indexKey, channels: {} };
    return {
      ...rest,
      auth: newAuth,
      read,
      version: 1,
      healthCheck: ZERO_HEALTH_CHECK,
      write: {},
    } as const;
  }),
);

export const ZERO_PROPERTIES = {
  secure: true,
  verifySsl: true,
  timeoutMs: defaultTimeoutMs,
  auth: ZERO_AUTH_CONFIGS.none,
  healthCheck: ZERO_HEALTH_CHECK,
  write: {},
  read: {},
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
