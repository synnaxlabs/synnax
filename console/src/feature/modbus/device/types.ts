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

/** The name a rack lists in its integrations when its Driver runs this one. */
export const INTEGRATION = "modbus";
const makeZ = z.literal(MAKE);
const modelZ = z.literal("Modbus");

const WRITE_CODES = ["multiple", "single"] as const;
export const writeCodesZ = z.enum(WRITE_CODES);
export type WriteCodes = z.infer<typeof writeCodesZ>;

const ZERO_CONNECTION = {
  transport: "tcp",
  host: "",
  port: 0,
  swapBytes: false,
  swapWords: false,
  unitId: 255,
  timeoutMs: 1000,
  retries: 1,
} as const;

const ZERO_REQUESTS = {
  maxRegisters: 125,
  maxGap: 125,
  writeCodes: "multiple",
  delayMs: 0,
} as const;

const wholeNumberZ = (name: string) => z.int(`${name} must be a whole number`);

const propertiesZ = z.object({
  connection: z
    .object({
      transport: z.literal("tcp").default(ZERO_CONNECTION.transport),
      host: z.string().default(ZERO_CONNECTION.host),
      port: z.number().default(ZERO_CONNECTION.port),
      swapBytes: z.boolean().default(ZERO_CONNECTION.swapBytes),
      swapWords: z.boolean().default(ZERO_CONNECTION.swapWords),
      unitId: wholeNumberZ("Unit ID").default(ZERO_CONNECTION.unitId),
      timeoutMs: wholeNumberZ("Timeout").default(ZERO_CONNECTION.timeoutMs),
      retries: wholeNumberZ("Retries").default(ZERO_CONNECTION.retries),
    })
    .default(() => ({ ...ZERO_CONNECTION })),
  requests: z
    .object({
      maxRegisters: wholeNumberZ("Max registers").default(ZERO_REQUESTS.maxRegisters),
      maxGap: wholeNumberZ("Max gap").default(ZERO_REQUESTS.maxGap),
      writeCodes: writeCodesZ.default(ZERO_REQUESTS.writeCodes),
      delayMs: wholeNumberZ("Request delay").default(ZERO_REQUESTS.delayMs),
    })
    .default(() => ({ ...ZERO_REQUESTS })),
  read: z
    .object({
      index: z.number().default(0),
      channels: z.record(z.string(), z.number()).default(() => ({})),
    })
    .default(() => ({ index: 0, channels: {} })),
  write: z
    .object({ channels: z.record(z.string(), z.number()).default(() => ({})) })
    .default(() => ({ channels: {} })),
});

export interface Properties extends z.infer<typeof propertiesZ> {}

export type Connection = Properties["connection"];

export const ZERO_PROPERTIES = {
  connection: ZERO_CONNECTION,
  requests: ZERO_REQUESTS,
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

/**
 * The order of the four bytes of a two-register value as the device sends them, with A
 * the most significant byte. A one-register value uses the first two letters.
 */
export const BYTE_ORDERS = ["ABCD", "BADC", "CDAB", "DCBA"] as const;
export const byteOrderZ = z.enum(BYTE_ORDERS);
export type ByteOrder = z.infer<typeof byteOrderZ>;

interface Swaps {
  swapBytes: boolean;
  swapWords: boolean;
}

// The Driver puts the low word first unless swapWords is set.
const SWAPS: Record<ByteOrder, Swaps> = {
  ABCD: { swapBytes: false, swapWords: true },
  BADC: { swapBytes: true, swapWords: true },
  CDAB: { swapBytes: false, swapWords: false },
  DCBA: { swapBytes: true, swapWords: false },
};

/** @returns the device swap settings that give the byte order. */
export const swapsOf = (order: ByteOrder): Swaps => SWAPS[order];

/** @returns the byte order that the device swap settings give. */
export const byteOrderOf = ({ swapBytes, swapWords }: Swaps): ByteOrder => {
  if (swapWords) return swapBytes ? "BADC" : "ABCD";
  return swapBytes ? "DCBA" : "CDAB";
};
