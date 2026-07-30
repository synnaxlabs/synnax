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

import { AuthError, DisconnectedError } from "@/errors";
import { createTestClient, TEST_CLIENT_PARAMS } from "@/testutil";

const FAST_RETRY = {
  baseInterval: TimeSpan.milliseconds(5),
  scale: 1,
  maxRetries: 2,
};

describe("connect", () => {
  it("should await the client's first success", async () => {
    const client = createTestClient();
    const state = await client.connect();
    expect(state.variant).toEqual("success");
    expect(state.details.streamLive).toBe(true);
    client.close();
  });

  it("should be idempotent on an already-connected client", async () => {
    const client = createTestClient();
    await client.connect();
    const state = await client.connect();
    expect(state.variant).toEqual("success");
    client.close();
  });

  it("should reject with AuthError on bad credentials", async () => {
    const client = createTestClient({ password: "definitely-wrong" });
    await expect(client.connect()).rejects.toThrow(AuthError);
    client.close();
  });

  it("should reject against an unreachable cluster after the retry budget", async () => {
    const client = createTestClient({ port: 9999, retry: FAST_RETRY });
    await expect(client.connect()).rejects.toThrow();
    client.close();
  });

  it("should recover from an auth failure via reauthenticate", async () => {
    const client = createTestClient({ password: "definitely-wrong" });
    await expect(client.connect()).rejects.toThrow(AuthError);
    expect(client.connection.status.details.reason).toEqual("auth");
    client.reauthenticate({
      username: TEST_CLIENT_PARAMS.username,
      password: TEST_CLIENT_PARAMS.password,
    });
    const state = await client.connect();
    expect(state.variant).toEqual("success");
    client.close();
  });

  it("should reject on timeout", async () => {
    // the backoff outlasts the timeout, so the timeout is what rejects rather
    // than an escalation to error(unreachable)
    const client = createTestClient({
      port: 9999,
      retry: { baseInterval: TimeSpan.seconds(1), scale: 1, maxRetries: 100 },
    });
    await expect(
      client.connect({ timeout: TimeSpan.milliseconds(10) }),
    ).rejects.toThrow(/timed out after/);
    client.close();
  });

  it("should become disabled on close", async () => {
    const client = createTestClient();
    await client.connect();
    client.close();
    expect(client.connection.status.variant).toEqual("disabled");
  });
});

describe("short circuit", () => {
  it("should reject requests instantly while unreachable and heal on recovery", async () => {
    const client = createTestClient({
      port: 9999,
      retry: FAST_RETRY,
    });
    await expect(client.connect()).rejects.toThrow();
    expect(client.connection.status.details.reason).toEqual("unreachable");
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
