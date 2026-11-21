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

import { opcua } from "@/opcua";

const SERVER = {
  key: "22222222-2222-4222-8222-222222222222",
  rack: 1,
  name: "PLC",
  connection: { endpoint: "opc.tcp://plc:4840" },
} as const;

describe("serverZ", () => {
  it("should parse a server and apply the defaults", () => {
    const server = opcua.server.serverZ.parse(SERVER);
    expect(server.connection.securityMode).toEqual("None");
    expect(server.connection.clientTimeoutMs).toEqual(5000);
    expect(server.connection.password).toBeUndefined();
    expect(server.readIndexes).toEqual([]);
    expect(server.readChannels).toEqual({});
  });

  it("should keep the case of node IDs from the wire", () => {
    const wire = {
      ...SERVER,
      read_channels: { "ns=2;s=tag_one": 5 },
      write_channels: { "ns=2;s=tag_two": 6 },
    };
    const server = opcua.server.serverZ.parse(
      caseconv.snakeToCamel(wire, { schema: opcua.server.serverZ }),
    );
    expect(server.readChannels).toEqual({ "ns=2;s=tag_one": 5 });
    expect(server.writeChannels).toEqual({ "ns=2;s=tag_two": 6 });
  });

  const INVALID: [string, Record<string, unknown>][] = [
    ["an empty name", { name: "" }],
    ["an empty endpoint", { connection: { endpoint: "" } }],
    [
      "an unknown security policy",
      { connection: { ...SERVER.connection, securityPolicy: "Basic512" } },
    ],
    ["a zero timeout", { connection: { ...SERVER.connection, clientTimeoutMs: 0 } }],
  ];
  it.each(INVALID)("should reject %s", (_, change) => {
    expect(opcua.server.serverZ.safeParse({ ...SERVER, ...change }).success).toBe(
      false,
    );
  });
});
