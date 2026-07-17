// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { z } from "zod/v4";

export const MAKE = "slack";
const makeZ = z.literal(MAKE);
const modelZ = z.literal("Slack workspace");

const v1PropertiesZ = z.object({
  botToken: z.string().min(1, "Bot token is required"),
  version: z.literal(1),
});

export interface Properties extends z.infer<typeof v1PropertiesZ> {}

export const propertiesZ: z.ZodType<Properties> = v1PropertiesZ;

export const ZERO_PROPERTIES = {
  botToken: "",
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
