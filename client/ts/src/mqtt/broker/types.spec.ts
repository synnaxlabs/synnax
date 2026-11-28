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

import { mqtt } from "@/mqtt";

const BROKER = {
  key: "22222222-2222-4222-8222-222222222222",
  rack: 1,
  name: "Plant Broker",
  host: "mqtt.example.com",
} as const;

const SECRET = "33333333-3333-4333-8333-333333333333";

describe("brokerZ", () => {
  it("should parse a broker and apply the defaults", () => {
    const broker = mqtt.broker.brokerZ.parse(BROKER);
    expect(broker.port).toBeUndefined();
    expect(broker.transport).toEqual({ type: "tcp" });
    expect(broker.auth).toEqual({ type: "none" });
    expect(broker.clientId).toBeUndefined();
    expect(broker.keepAlive).toEqual(TimeSpan.seconds(30).nanoseconds);
    expect(broker.sparkplug).toEqual({ mode: { type: "passive" }, groups: [] });
    expect(broker.reads).toEqual([]);
    expect(broker.writes).toEqual([]);
    expect(broker.edgeCommands).toEqual([]);
  });

  it("should apply the defaults of TLS and of a Sparkplug read", () => {
    const broker = mqtt.broker.brokerZ.parse({
      ...BROKER,
      transport: { type: "tls" },
      reads: [
        {
          type: "sparkplug",
          group: "Plant",
          edgeNode: "Line1",
          tag: "Pressure",
          index: 1,
          channel: 2,
        },
      ],
    });
    expect(broker.transport).toEqual({
      type: "tls",
      certificateValidationDisabled: false,
    });
    expect(broker.reads[0]).toMatchObject({ device: "" });
  });

  it("should parse a user name auth that names its password by secret", () => {
    const broker = mqtt.broker.brokerZ.parse({
      ...BROKER,
      auth: { type: "username", username: "admin", password: SECRET },
    });
    expect(broker.auth).toEqual({
      type: "username",
      username: "admin",
      password: SECRET,
    });
  });

  const INVALID: [string, Record<string, unknown>][] = [
    ["an empty name", { name: "" }],
    ["an empty host", { host: "" }],
    ["an unknown transport", { transport: { type: "websocket" } }],
    ["port 0", { port: 0 }],
    ["an empty client ID", { clientId: "" }],
    ["a keep alive under 1 s", { keepAlive: TimeSpan.milliseconds(999).nanoseconds }],
    ["a keep alive over 65535 s", { keepAlive: TimeSpan.seconds(65536).nanoseconds }],
    [
      "a plaintext password",
      { auth: { type: "username", username: "a", password: "x" } },
    ],
    ["a primary host with no ID", { sparkplug: { mode: { type: "primary" } } }],
    [
      "a plain read with no index",
      { reads: [{ type: "plain", topic: "a", message: SECRET, index: 0, channel: 2 }] },
    ],
    [
      "a plain write with no field",
      { writes: [{ type: "plain", topic: "a", channel: 3 }] },
    ],
  ];
  it.each(INVALID)("should reject %s", (_, change) => {
    expect(mqtt.broker.brokerZ.safeParse({ ...BROKER, ...change }).success).toBe(false);
  });
});
