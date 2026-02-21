// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { z } from "zod";

export const MAKE = "Modbus";
const makeZ = z.literal(MAKE);
const modelZ = z.literal("Modbus");

const propertiesZ = z.object({
  connection: z.object({
    host: z.string(),
    port: z.number(),
    swapBytes: z.boolean(),
    swapWords: z.boolean(),
  }),
  read: z.object({ index: z.number(), channels: z.record(z.string(), z.number()) }),
  write: z.object({ channels: z.record(z.string(), z.number()) }),
});

export interface Properties extends z.infer<typeof propertiesZ> {}

export const ZERO_PROPERTIES = {
  connection: { host: "", port: 0, swapBytes: false, swapWords: false },
  read: { index: 0, channels: {} },
  write: { channels: {} },
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
