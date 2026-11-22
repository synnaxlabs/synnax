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

import { http } from "@/http";

const SERVER = {
  key: "22222222-2222-4222-8222-222222222222",
  rack: 1,
  name: "Weather API",
  host: "api.example.com",
} as const;

const SECRET = "33333333-3333-4333-8333-333333333333";

describe("serverZ", () => {
  it("should parse a server and apply the defaults", () => {
    const server = http.server.serverZ.parse(SERVER);
    expect(server.scheme).toEqual("https");
    expect(server.certificateValidationDisabled).toBe(false);
    expect(server.timeoutMs).toEqual(100);
    expect(server.maxConcurrentRequests).toEqual(6);
    expect(server.auth).toEqual({ type: "none" });
    expect(server.healthCheck.method).toEqual("GET");
    expect(server.healthCheck.response).toBeUndefined();
    expect(server.readChannels).toEqual({});
  });

  it("should parse an API key that names its key by secret", () => {
    const server = http.server.serverZ.parse({
      ...SERVER,
      auth: { type: "api_key", key: SECRET, name: "X-Key" },
    });
    expect(server.auth).toEqual({
      type: "api_key",
      key: SECRET,
      sendAs: "header",
      name: "X-Key",
    });
  });

  it("should keep the case of paths and pointers from the wire", () => {
    const wire = {
      ...SERVER,
      read_channels: { "/weather_now": { index: 1, channels: { "/temp_c": 2 } } },
      write_channels: { "/set_point": 3 },
    };
    const server = http.server.serverZ.parse(
      caseconv.snakeToCamel(wire, { schema: http.server.serverZ }),
    );
    expect(server.readChannels).toEqual({
      "/weather_now": { index: 1, channels: { "/temp_c": 2 } },
    });
    expect(server.writeChannels).toEqual({ "/set_point": 3 });
  });

  const INVALID: [string, Record<string, unknown>][] = [
    ["an empty name", { name: "" }],
    ["an empty host", { host: "" }],
    ["an unknown scheme", { scheme: "ftp" }],
    ["a zero timeout", { timeoutMs: 0 }],
    ["a plaintext token", { auth: { type: "bearer", token: "hunter2" } }],
    ["an API key with no name", { auth: { type: "api_key", key: SECRET } }],
  ];
  it.each(INVALID)("should reject %s", (_, change) => {
    expect(http.server.serverZ.safeParse({ ...SERVER, ...change }).success).toBe(false);
  });
});
