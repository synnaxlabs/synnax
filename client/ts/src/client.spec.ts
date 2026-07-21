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
import { z } from "zod";

import { checkConnection, connect } from "@/client";
import { AuthError, DisconnectedError } from "@/errors";
import { createTestClient, TEST_CLIENT_PARAMS } from "@/testutil";

const FAST_RETRY = {
  baseInterval: TimeSpan.milliseconds(5),
  scale: 1,
  maxRetries: 2,
};

describe("checkConnection", () => {
  it("should check connection to the server", async () => {
    const state = await checkConnection({
      host: TEST_CLIENT_PARAMS.host,
      port: TEST_CLIENT_PARAMS.port,
      secure: false,
    });
    expect(state.variant).toEqual("success");
    expect(z.uuid().safeParse(state.clusterKey).success).toBe(true);
  });

  it("should include client version in the connection check", async () => {
    const state = await checkConnection({
      host: TEST_CLIENT_PARAMS.host,
      port: TEST_CLIENT_PARAMS.port,
      secure: false,
    });
    expect(state.clientVersion).toBeDefined();
    expect(state.clientServerCompatible).toBe(true);
  });

  it("should support custom name parameter", async () => {
    const state = await checkConnection({
      host: TEST_CLIENT_PARAMS.host,
      port: TEST_CLIENT_PARAMS.port,
      secure: false,
      name: "test-client",
    });
    expect(state.variant).toEqual("success");
  });

  it("should handle connection failure to invalid host", async () => {
    const state = await checkConnection({
      host: "invalid-host-that-does-not-exist",
      port: 9999,
      secure: false,
      retry: { maxRetries: 0 },
    });
    expect(state.variant).toEqual("error");
    expect(state.reason).toEqual("unreachable");
  });

  it("should handle connection failure to invalid port", async () => {
    const state = await checkConnection({
      host: TEST_CLIENT_PARAMS.host,
      port: 9999,
      secure: false,
      retry: { maxRetries: 0 },
    });
    expect(state.variant).toEqual("error");
  });
});

describe("connect", () => {
  it("should construct a client and await its first success", async () => {
    const client = await connect(TEST_CLIENT_PARAMS);
    expect(client.connection.state.variant).toEqual("success");
    expect(client.connection.state.streamLive).toBe(true);
    client.close();
  });

  it("should be idempotent on an already-connected client", async () => {
    const client = await connect(TEST_CLIENT_PARAMS);
    const state = await client.connect();
    expect(state.variant).toEqual("success");
    client.close();
  });

  it("should reject with AuthError on bad credentials", async () => {
    await expect(
      connect({ ...TEST_CLIENT_PARAMS, password: "definitely-wrong" }),
    ).rejects.toThrow(AuthError);
  });

  it("should reject against an unreachable cluster after the retry budget", async () => {
    await expect(
      connect({ ...TEST_CLIENT_PARAMS, port: 9999, retry: FAST_RETRY }),
    ).rejects.toThrow();
  });

  it("should recover from an auth failure via reauthenticate", async () => {
    const client = createTestClient({ password: "definitely-wrong" });
    await expect(client.connect()).rejects.toThrow(AuthError);
    expect(client.connection.state.reason).toEqual("auth");
    client.reauthenticate({
      username: TEST_CLIENT_PARAMS.username,
      password: TEST_CLIENT_PARAMS.password,
    });
    const state = await client.connect();
    expect(state.variant).toEqual("success");
    client.close();
  });

  it("should reject on timeout", async () => {
    await expect(
      connect(
        {
          ...TEST_CLIENT_PARAMS,
          port: 9999,
          retry: { ...FAST_RETRY, maxRetries: Infinity },
        },
        { timeout: TimeSpan.milliseconds(50) },
      ),
    ).rejects.toThrow();
  });
});

describe("short circuit", () => {
  it("should reject requests instantly while unreachable and heal on recovery", async () => {
    const client = createTestClient({
      port: 9999,
      retry: FAST_RETRY,
    });
    await expect(client.connect()).rejects.toThrow();
    expect(client.connection.state.reason).toEqual("unreachable");
    const start = performance.now();
    await expect(client.channels.retrieve(["missing"])).rejects.toThrow(
      DisconnectedError,
    );
    expect(performance.now() - start).toBeLessThan(100);
    client.close();
  });

  it("should not short-circuit while connected", async () => {
    const client = createTestClient();
    await client.connect();
    const ch = await client.channels.create({
      name: `short_circuit_test_${Math.floor(Math.random() * 1e9)}`,
      dataType: "float32",
      virtual: true,
    });
    expect(ch.key).not.toBe(0);
    client.close();
  });
});
