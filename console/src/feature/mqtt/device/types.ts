// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, type device } from "@synnaxlabs/client";
import { caseconv } from "@synnaxlabs/x";
import { z } from "zod";

export const MAKE = "mqtt";
const makeZ = z.literal(MAKE);
const modelZ = z.literal("MQTT broker");

const SPARKPLUG_ID_PATTERN = /^[^/+#]*$/;

/**
 * @param label - The name of the ID in the issue message.
 * @returns a schema for an ID that can be one level of a Sparkplug B topic.
 */
export const sparkplugIDZ = (label: string) =>
  z.string().regex(SPARKPLUG_ID_PATTERN, `${label} must not hold /, +, or #`);

// The issue sits on the list, because one input edits all of the groups.
const validateGroups = (ctx: z.core.ParsePayload<string[]>) =>
  ctx.value
    .filter((group) => group === "" || !SPARKPLUG_ID_PATTERN.test(group))
    .forEach((group) =>
      ctx.issues.push({
        code: "custom",
        input: ctx.value,
        message:
          group === ""
            ? "Group must not be empty"
            : `Group "${group}" must not hold /, +, or #`,
        path: [],
      }),
    );

const sparkplugPropertiesZ = z.object({
  hostId: sparkplugIDZ("Host ID").default(""),
  groups: z
    .string()
    .array()
    .check(validateGroups)
    .default(() => []),
});

// Anything that is not a channel key reads as no index.
const storedIndexZ = z.preprocess(
  (v) => (channel.keyZ.safeParse(v).success ? v : 0),
  channel.keyZ,
);

// Pointers and topics are values, not property names, so they keep their case.
const readTopicPropsZ = z.object({
  index: storedIndexZ,
  channels: caseconv.preserveCase(z.record(z.string(), channel.keyZ)),
});

export const propertiesZ = z.object({
  secure: z.boolean().default(false),
  verificationSkipped: z.boolean().default(false),
  port: z
    .number()
    .int("Port must be a whole number")
    .min(0, "Port must be 0 to 65535")
    .max(65535, "Port must be 0 to 65535")
    .default(0),
  keepAlive: z
    .number()
    .int("Keep alive must be a whole number")
    .nonnegative("Keep alive must be non-negative")
    .default(0),
  caFile: z.string().default(""),
  certFile: z.string().default(""),
  keyFile: z.string().default(""),
  username: z.string().default(""),
  password: z.string().default(""),
  clientId: z.string().default(""),
  sparkplug: sparkplugPropertiesZ.prefault({}),
  read: caseconv.preserveKeys(z.record(z.string(), readTopicPropsZ).default({})),
  write: caseconv.preserveCase(z.record(z.string(), channel.keyZ).default({})),
  version: z.literal(1).default(1),
});

export interface Properties extends z.infer<typeof propertiesZ> {}

export const ZERO_PROPERTIES: Properties = {
  secure: false,
  verificationSkipped: false,
  port: 0,
  keepAlive: 0,
  caFile: "",
  certFile: "",
  keyFile: "",
  username: "",
  password: "",
  clientId: "",
  sparkplug: { hostId: "", groups: [] },
  read: {},
  write: {},
  version: 1,
};

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
