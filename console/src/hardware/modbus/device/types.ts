// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { caseconv } from "@synnaxlabs/x";
import { z } from "zod";

export const MAKE = "Modbus";
export const makeZ = z.literal(MAKE);
export type Make = z.infer<typeof makeZ>;
export const MODEL = "Modbus";
export const modelZ = z.literal(MODEL);
export type Model = z.infer<typeof modelZ>;

export const connectionConfigZ = z.object({
  host: z.string(),
  port: z.number(),
  swapBytes: z.boolean(),
  swapWords: z.boolean(),
});
export interface ConnectionConfig extends z.infer<typeof connectionConfigZ> {}

export const ZERO_CONNECTION_CONFIG = {
  host: "",
  port: 0,
  swapBytes: false,
  swapWords: false,
} as const satisfies ConnectionConfig;

export const propertiesZ = z.object({
  connection: connectionConfigZ,
  read: z.object({
    index: z.number(),
    channels: caseconv.preserveCase(z.record(z.string(), z.number())),
  }),
  write: z.object({
    channels: caseconv.preserveCase(z.record(z.string(), z.number())),
  }),
});
export interface Properties extends z.infer<typeof propertiesZ> {}
export const ZERO_PROPERTIES = {
  connection: ZERO_CONNECTION_CONFIG,
  read: { index: 0, channels: {} },
  write: { channels: {} },
} as const satisfies Properties;

export interface Device
  extends device.Device<typeof propertiesZ, typeof makeZ, typeof modelZ> {}
