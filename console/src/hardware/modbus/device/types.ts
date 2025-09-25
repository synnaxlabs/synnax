// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { z } from "zod/v4";

export const MAKE = "modbus";
export type Make = "modbus";
export type Model = "modbus";

export const connectionConfigZ = z.object({
  host: z.string(),
  port: z.number(),
  swapBytes: z.boolean(),
  swapWords: z.boolean(),
});
export type ConnectionConfig = z.infer<typeof connectionConfigZ>;

export const ZERO_CONNECTION_CONFIG = {
  host: "",
  port: 0,
  swapBytes: false,
  swapWords: false,
} as const satisfies ConnectionConfig;

export const propertiesZ = z.object({
  connection: connectionConfigZ,
  read: z.object({ index: z.number(), channels: z.record(z.string(), z.number()) }),
  write: z.object({ channels: z.record(z.string(), z.number()) }),
});
export type Properties = z.infer<typeof propertiesZ>;

export const ZERO_PROPERTIES = {
  connection: ZERO_CONNECTION_CONFIG,
  read: { index: 0, channels: {} },
  write: { channels: {} },
} as const satisfies Properties;

export interface Device extends device.Device<Properties, Make> {}
