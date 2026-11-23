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
    expect(server.scheme).toEqual({
      type: "https",
      certificateValidationDisabled: false,
    });
    expect(server.port).toBeUndefined();
    expect(server.basePath).toEqual("");
    expect(server.timeout).toEqual(TimeSpan.SECOND.nanoseconds);
    expect(server.maxConcurrentRequests).toEqual(6);
    expect(server.auth).toEqual({ type: "none" });
    expect(server.healthCheck.method).toEqual("GET");
    expect(server.healthCheck.response).toBeUndefined();
    expect(server.reads).toEqual([]);
    expect(server.writes).toEqual([]);
  });

  it("should apply the defaults of the requests of bindings", () => {
    const server = http.server.serverZ.parse({
      ...SERVER,
      reads: [{ request: { path: "/now" }, pointer: "/t", index: 1, channel: 2 }],
      writes: [{ request: { path: "/valve" }, channel: 3 }],
    });
    expect(server.reads[0].request).toEqual({ method: "GET", path: "/now", query: [] });
    expect(server.writes[0].request).toEqual({ method: "POST", path: "/valve" });
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

  const INVALID: [string, Record<string, unknown>][] = [
    ["an empty name", { name: "" }],
    ["an empty host", { host: "" }],
    ["an unknown scheme", { scheme: { type: "ftp" } }],
    ["port 0", { port: 0 }],
    ["a timeout under 10 ms", { timeout: TimeSpan.milliseconds(9).nanoseconds }],
    ["more than 64 concurrent requests", { maxConcurrentRequests: 65 }],
    ["a read with no index", { reads: [{ request: {}, index: 0, channel: 2 }] }],
    ["a write with no channel", { writes: [{ request: {}, channel: 0 }] }],
    ["a plaintext token", { auth: { type: "bearer", token: "hunter2" } }],
    ["an API key with no name", { auth: { type: "api_key", key: SECRET } }],
  ];
  it.each(INVALID)("should reject %s", (_, change) => {
    expect(http.server.serverZ.safeParse({ ...SERVER, ...change }).success).toBe(false);
  });
});
