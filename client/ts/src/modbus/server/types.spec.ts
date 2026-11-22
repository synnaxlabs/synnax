// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { caseconv } from "@synnaxlabs/x";
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
      timeoutMs: 1000,
      retries: 1,
    });
    expect(server.requests).toEqual({
      maxRegisters: 125,
      maxGap: 125,
      writeCodes: "multiple",
      maskWriteSupported: false,
      delayMs: 0,
    });
    expect(server.readIndex).toEqual(0);
    expect(server.readChannels).toEqual({});
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

  it("should keep the case of register keys from the wire", () => {
    const wire = {
      ...SERVER,
      read_channels: { "holding-register-5-float32": 5 },
      write_channels: { "coil-1": 6 },
    };
    const server = modbus.server.serverZ.parse(
      caseconv.snakeToCamel(wire, { schema: modbus.server.serverZ }),
    );
    expect(server.readChannels).toEqual({ "holding-register-5-float32": 5 });
    expect(server.writeChannels).toEqual({ "coil-1": 6 });
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
    ["a zero timeout", { connection: { ...SERVER.connection, timeoutMs: 0 } }],
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
