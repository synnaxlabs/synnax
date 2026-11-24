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
  connection: { transport: "tcp", host: "plc" },
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
    expect(server.requestPolicy).toEqual({
      maxReadRegisters: 125,
      maxWriteRegisters: 123,
      maxReadBits: 2000,
      maxWriteBits: 1968,
      maxGap: 125,
      writeFunction: "multiple",
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

  it("should parse the bindings", () => {
    const server = modbus.server.serverZ.parse({
      ...SERVER,
      reads: [
        {
          location: {
            table: "holding_register",
            address: 5,
            value: { type: "whole", dataType: "float32", wordsSwapped: true },
          },
          index: 3,
          channel: 4,
        },
        {
          location: {
            table: "input_register",
            address: 6,
            value: { type: "bit", bit: 3 },
          },
          index: 3,
          channel: 5,
        },
        {
          location: {
            table: "holding_register",
            address: 7,
            value: { type: "string", length: 8 },
          },
          index: 3,
          channel: 6,
        },
      ],
      writes: [{ location: { table: "coil", address: 1 }, channel: 7 }],
    });
    expect(server.reads.map(({ location }) => location)).toEqual([
      {
        table: "holding_register",
        address: 5,
        value: { type: "whole", dataType: "float32", wordsSwapped: true },
      },
      { table: "input_register", address: 6, value: { type: "bit", bit: 3 } },
      { table: "holding_register", address: 7, value: { type: "string", length: 8 } },
    ]);
    expect(server.writes[0].location).toEqual({ table: "coil", address: 1 });
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
    [
      "a baud rate the Driver cannot set",
      { connection: { transport: "rtu", serialPort: "COM3", baudRate: 14400 } },
    ],
    ["an empty name", { name: "" }],
    ["no transport", { connection: { host: "plc", port: 502 } }],
    ["port 0", { connection: { ...SERVER.connection, port: 0 } }],
    [
      "three stop bits",
      { connection: { transport: "rtu", serialPort: "COM3", stopBits: 3 } },
    ],
    ["an empty host", { connection: { ...SERVER.connection, host: "" } }],
    [
      "a timeout under a millisecond",
      { connection: { ...SERVER.connection, timeout: 1 } },
    ],
    [
      "a read with no index",
      { reads: [{ location: { table: "coil", address: 0 }, channel: 1 }] },
    ],
    [
      "a read of an unknown table",
      {
        reads: [
          { location: { table: "file_record", address: 0 }, index: 1, channel: 2 },
        ],
      },
    ],
    [
      "a write of bit 16",
      {
        writes: [
          {
            location: {
              table: "holding_register",
              address: 0,
              value: { type: "bit", bit: 16 },
            },
            channel: 1,
          },
        ],
      },
    ],
    [
      "a string of no characters",
      {
        reads: [
          {
            location: {
              table: "input_register",
              address: 0,
              value: { type: "string", length: 0 },
            },
            index: 1,
            channel: 2,
          },
        ],
      },
    ],
    [
      "a write to a discrete input",
      { writes: [{ location: { table: "discrete_input", address: 0 }, channel: 1 }] },
    ],
    [
      "a write of a string",
      {
        writes: [
          {
            location: {
              table: "holding_register",
              address: 0,
              value: { type: "string", length: 4 },
            },
            channel: 1,
          },
        ],
      },
    ],
    [
      "a holding register with no value",
      {
        writes: [{ location: { table: "holding_register", address: 0 }, channel: 1 }],
      },
    ],
    [
      "an RTU unit of 248",
      { connection: { transport: "rtu", serialPort: "COM3", unitId: 248 } },
    ],
    ["an unknown write function", { requestPolicy: { writeFunction: "masked" } }],
    ["a read of 126 registers", { requestPolicy: { maxReadRegisters: 126 } }],
  ];
  it.each(INVALID)("should reject %s", (_, change) => {
    expect(modbus.server.serverZ.safeParse({ ...SERVER, ...change }).success).toBe(
      false,
    );
  });
});

describe("updateZ", () => {
  it("should parse an identity update and apply its defaults", () => {
    const update = modbus.server.updateZ.parse({
      group: "identity",
      rack: 1,
      name: "PLC",
      connection: { transport: "tcp", host: "plc", port: 502 },
    });
    expect(update.group).toEqual("identity");
    if (update.group !== "identity") return;
    expect(update.connection.timeout).toEqual(1000000000);
    expect(update.requestPolicy.maxReadRegisters).toEqual(125);
  });

  it("should parse a configuration update and apply its defaults", () => {
    const update = modbus.server.updateZ.parse({
      group: "configuration",
      reads: [{ location: { table: "coil", address: 1 }, index: 2, channel: 3 }],
    });
    expect(update.group).toEqual("configuration");
    if (update.group !== "configuration") return;
    expect(update.reads[0].channel).toEqual(3);
    expect(update.writes).toEqual([]);
  });

  it("should reject an unknown group", () => {
    expect(modbus.server.updateZ.safeParse({ group: "other" }).success).toBe(false);
  });
});
