// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, TimeSpan } from "@synnaxlabs/client";
import { z } from "zod";

export const MAKE = "Modbus";

/** The name a rack lists in its integrations when its Driver runs this one. */
export const INTEGRATION = "modbus";
const makeZ = z.literal(MAKE);
const modelZ = z.literal("Modbus");

const WRITE_CODES = ["multiple", "single"] as const;
const writeCodesZ = z.enum(WRITE_CODES);
export type WriteCodes = z.infer<typeof writeCodesZ>;

export type Transport = "tcp" | "rtu";

/** The baud rates the Driver accepts, which libmodbus sets on every platform. */
export const BAUD_RATES = [
  1200, 2400, 4800, 9600, 19200, 38400, 57600, 115200, 230400,
] as const;

export const PARITIES = ["none", "even", "odd"] as const;
const parityZ = z.enum(PARITIES);
export type Parity = z.infer<typeof parityZ>;

export const STOP_BITS = [1, 2] as const;

const ZERO_COMMON_CONNECTION = {
  swapBytes: false,
  swapWords: false,
  timeoutMs: 1000,
  retries: 1,
} as const;

const ZERO_TCP_CONNECTION = {
  ...ZERO_COMMON_CONNECTION,
  transport: "tcp",
  host: "",
  port: 0,
  unitId: 255,
} as const;

const ZERO_RTU_CONNECTION = {
  ...ZERO_COMMON_CONNECTION,
  transport: "rtu",
  serialPort: "",
  baudRate: 19200,
  parity: "even",
  stopBits: 1,
  unitId: 1,
} as const;

const ZERO_REQUESTS = {
  maxRegisters: 125,
  maxGap: 125,
  writeCodes: "multiple",
  maskWriteSupported: false,
  delayMs: 0,
} as const;

const wholeNumberZ = (name: string) => z.int(`${name} must be a whole number`);

const commonConnectionZ = z.object({
  swapBytes: z.boolean().default(ZERO_COMMON_CONNECTION.swapBytes),
  swapWords: z.boolean().default(ZERO_COMMON_CONNECTION.swapWords),
  timeoutMs: wholeNumberZ("Timeout").default(ZERO_COMMON_CONNECTION.timeoutMs),
  retries: wholeNumberZ("Retries").default(ZERO_COMMON_CONNECTION.retries),
});

const tcpConnectionZ = commonConnectionZ.extend({
  transport: z.literal("tcp"),
  host: z.string().default(ZERO_TCP_CONNECTION.host),
  port: z.number().default(ZERO_TCP_CONNECTION.port),
  unitId: wholeNumberZ("Unit ID").default(ZERO_TCP_CONNECTION.unitId),
});

const rtuConnectionZ = commonConnectionZ.extend({
  transport: z.literal("rtu"),
  serialPort: z.string().default(ZERO_RTU_CONNECTION.serialPort),
  baudRate: wholeNumberZ("Baud rate").default(ZERO_RTU_CONNECTION.baudRate),
  parity: parityZ.default(ZERO_RTU_CONNECTION.parity),
  stopBits: wholeNumberZ("Stop bits").default(ZERO_RTU_CONNECTION.stopBits),
  unitId: wholeNumberZ("Unit ID").default(ZERO_RTU_CONNECTION.unitId),
});

// Devices stored before serial connections have no transport, and are all TCP.
const fillTransport = (v: unknown): unknown =>
  typeof v === "object" && v != null && !("transport" in v)
    ? { ...v, transport: "tcp" }
    : v;

const propertiesZ = z.object({
  connection: z
    .preprocess(
      fillTransport,
      z.discriminatedUnion("transport", [tcpConnectionZ, rtuConnectionZ]),
    )
    .default(() => ({ ...ZERO_TCP_CONNECTION })),
  requests: z
    .object({
      maxRegisters: wholeNumberZ("Max registers").default(ZERO_REQUESTS.maxRegisters),
      maxGap: wholeNumberZ("Max gap").default(ZERO_REQUESTS.maxGap),
      writeCodes: writeCodesZ.default(ZERO_REQUESTS.writeCodes),
      maskWriteSupported: z.boolean().default(ZERO_REQUESTS.maskWriteSupported),
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
export type Requests = Properties["requests"];

/** The connection a device starts with on each transport. */
export const ZERO_CONNECTIONS = {
  tcp: ZERO_TCP_CONNECTION,
  rtu: ZERO_RTU_CONNECTION,
} as const satisfies Record<Transport, Connection>;

export const ZERO_PROPERTIES = {
  connection: ZERO_TCP_CONNECTION,
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

/** The request counters of a server, as the Driver reports them in its status. */
const statsZ = z.object({
  requests: z.number(),
  retries: z.number(),
  timeouts: z.number(),
  exceptions: z.number(),
  lastResponse: TimeSpan.z,
  maxResponse: TimeSpan.z,
});
export interface Stats extends z.infer<typeof statsZ> {}

const statusDataZ = z.object({ stats: statsZ });

/** @returns the request counters in a device status, or undefined if it has none. */
export const statsOf = (status?: device.Status | null): Stats | undefined => {
  const data = status?.details.data;
  return data == null ? undefined : statusDataZ.parse(data).stats;
};

/**
 * The order of the four bytes of a two-register value as the device sends them, with A
 * the most significant byte. A one-register value uses the first two letters.
 */
export const BYTE_ORDERS = ["ABCD", "BADC", "CDAB", "DCBA"] as const;
export type ByteOrder = (typeof BYTE_ORDERS)[number];

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
