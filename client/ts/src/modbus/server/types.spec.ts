// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { modbus } from "@/modbus";

const SERVER = {
  key: "22222222-2222-4222-8222-222222222222",
  rack: 1,
  name: "PLC",
  connection: { transport: "tcp", host: "plc", port: 502 },
} as const;

describe("serverZ", () => {
  it("should parse a TCP server and apply the defaults", () => {
    const server = modbus.server.serverZ.parse(SERVER);
    expect(server.connection).toEqual({
      transport: "tcp",
      host: "plc",
      port: 502,
      unitId: 255,
      bytesSwapped: false,
      wordsSwapped: false,
      timeout: 1000000000,
      retries: 1,
    });
    expect(server.requests).toEqual({
      maxRegisters: 125,
      maxGap: 125,
      writeCodes: "multiple",
      maskWriteSupported: false,
      delay: 0,
    });
    expect(server.reads).toEqual([]);
    expect(server.writes).toEqual([]);
  });

  it("should apply the defaults of an RTU connection", () => {
    const server = modbus.server.serverZ.parse({
      ...SERVER,
      connection: { transport: "rtu", serialPort: "COM3" },
    });
    expect(server.connection).toMatchObject({
      baudRate: 19200,
      parity: "even",
      stopBits: 1,
      unitId: 1,
    });
  });

  it("should parse the register bindings", () => {
    const server = modbus.server.serverZ.parse({
      ...SERVER,
      reads: [
        {
          register: { type: "holding_register", address: 5, dataType: "float32" },
          index: 3,
          channel: 4,
        },
        {
          register: { type: "input_register", address: 6, bit: 3 },
          index: 3,
          channel: 5,
        },
      ],
      writes: [{ register: { type: "coil", address: 1 }, channel: 6 }],
    });
    expect(server.reads[0].register.dataType).toEqual("float32");
    expect(server.reads[1].register.bit).toEqual(3);
    expect(server.writes[0].register).toEqual({ type: "coil", address: 1 });
  });

  it("should parse the request counters of a status as numbers", () => {
    const details = modbus.server.statusDetailsZ.parse({
      rack: 1,
      server: SERVER.key,
      connected: true,
      stats: { requests: 7, retries: 1, lastResponse: 2000000 },
    });
    expect(details.stats?.requests).toEqual(7);
    expect(details.stats?.timeouts).toEqual(0);
    expect(details.stats?.lastResponse.milliseconds).toEqual(2);
  });

  const INVALID: [string, Record<string, unknown>][] = [
    ["an empty name", { name: "" }],
    ["no transport", { connection: { host: "plc", port: 502 } }],
    ["an empty host", { connection: { ...SERVER.connection, host: "" } }],
    [
      "a timeout under a millisecond",
      { connection: { ...SERVER.connection, timeout: 1 } },
    ],
    [
      "a read with no index",
      { reads: [{ register: { type: "coil", address: 0 }, channel: 1 }] },
    ],
    [
      "a read of an unknown register type",
      {
        reads: [
          { register: { type: "file_record", address: 0 }, index: 1, channel: 2 },
        ],
      },
    ],
    [
      "a write of bit 16",
      {
        writes: [
          { register: { type: "holding_register", address: 0, bit: 16 }, channel: 1 },
        ],
      },
    ],
    [
      "an RTU unit of 248",
      { connection: { transport: "rtu", serialPort: "COM3", unitId: 248 } },
    ],
    ["an unknown write code", { requests: { writeCodes: "masked" } }],
  ];
  it.each(INVALID)("should reject %s", (_, change) => {
    expect(modbus.server.serverZ.safeParse({ ...SERVER, ...change }).success).toBe(
      false,
    );
  });
});
