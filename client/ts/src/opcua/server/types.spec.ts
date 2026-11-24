// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { opcua } from "@/opcua";

const SERVER = {
  key: "22222222-2222-4222-8222-222222222222",
  rack: 1,
  name: "PLC",
  connection: { endpoint: "opc.tcp://plc:4840" },
} as const;

const SIGNED = {
  mode: "sign",
  policy: "Basic256Sha256",
  clientCertificate: "33333333-3333-4333-8333-333333333333",
} as const;

describe("serverZ", () => {
  it("should parse a server and apply the defaults", () => {
    const server = opcua.server.serverZ.parse(SERVER);
    expect(server.connection.auth).toEqual({ type: "anonymous" });
    expect(server.connection.security).toEqual({ mode: "none" });
    expect(server.connection.clientTimeout).toEqual(TimeSpan.seconds(5).nanoseconds);
    expect(server.reads).toEqual([]);
    expect(server.writes).toEqual([]);
  });

  it("should default a secured connection to trust on first use", () => {
    const server = opcua.server.serverZ.parse({
      ...SERVER,
      connection: { ...SERVER.connection, security: SIGNED },
    });
    expect(server.connection.security).toEqual({
      ...SIGNED,
      serverTrust: { type: "first_use" },
    });
  });

  it("should parse a user name login and the bindings", () => {
    const server = opcua.server.serverZ.parse({
      ...SERVER,
      connection: {
        ...SERVER.connection,
        auth: { type: "username", username: "op" },
      },
      reads: [{ node: "nsu=urn:plc;s=Temp", index: 1, channel: 2 }],
      writes: [{ node: "nsu=urn:plc;s=Valve", channel: 3 }],
    });
    expect(server.connection.auth).toEqual({ type: "username", username: "op" });
    expect(server.reads[0].field).toEqual("");
    expect(server.writes[0].channel).toEqual(3);
  });

  const INVALID: [string, Record<string, unknown>][] = [
    ["an empty name", { name: "" }],
    ["an empty endpoint", { connection: { endpoint: "" } }],
    [
      "an unknown security policy",
      { connection: { ...SERVER.connection, security: { ...SIGNED, policy: "None" } } },
    ],
    [
      "a user name login without a user name",
      {
        connection: { ...SERVER.connection, auth: { type: "username", username: "" } },
      },
    ],
    [
      "a request timeout under 100 ms",
      {
        connection: {
          ...SERVER.connection,
          clientTimeout: TimeSpan.milliseconds(99).nanoseconds,
        },
      },
    ],
    [
      "a session timeout under 1 s",
      {
        connection: {
          ...SERVER.connection,
          sessionTimeout: TimeSpan.milliseconds(999).nanoseconds,
        },
      },
    ],
    ["a read without an index", { reads: [{ node: "i=85", channel: 2 }] }],
    ["a write without a node", { writes: [{ node: "", channel: 3 }] }],
  ];
  it.each(INVALID)("should reject %s", (_, change) => {
    expect(opcua.server.serverZ.safeParse({ ...SERVER, ...change }).success).toBe(
      false,
    );
  });
});
