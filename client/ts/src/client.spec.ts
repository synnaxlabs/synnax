// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";
import { afterEach, assert, describe, expect, it, vi } from "vitest";

import { type connection } from "@/connection";
import { AuthError, DisconnectedError } from "@/errors";
import { createTestClient, TEST_CLIENT_PARAMS } from "@/testutil";

const reasonOf = (status: connection.Status): connection.Reason | undefined =>
  status.variant === "error" ? status.details.reason : undefined;

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
    await client.close();
  });

  it("should be idempotent on an already-connected client", async () => {
    const client = createTestClient();
    await client.connect();
    const state = await client.connect();
    expect(state.variant).toEqual("success");
    await client.close();
  });

  it("should reject with AuthError on bad credentials", async () => {
    const client = createTestClient({ password: "definitely-wrong" });
    await expect(client.connect()).rejects.toThrow(AuthError);
    await client.close();
  });

  it("should reject against an unreachable cluster after the retry budget", async () => {
    const client = createTestClient({ port: 9999, retry: FAST_RETRY });
    await expect(client.connect()).rejects.toThrow();
    await client.close();
  });

  it("should recover from an auth failure via reauthenticate", async () => {
    const client = createTestClient({ password: "definitely-wrong" });
    await expect(client.connect()).rejects.toThrow(AuthError);
    expect(reasonOf(client.connection.status)).toEqual("auth");
    client.reauthenticate({
      username: TEST_CLIENT_PARAMS.username,
      password: TEST_CLIENT_PARAMS.password,
    });
    const state = await client.connect();
    expect(state.variant).toEqual("success");
    await client.close();
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
    await client.close();
  });

  it("should become disabled on close", async () => {
    const client = createTestClient();
    await client.connect();
    await client.close();
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
    expect(reasonOf(client.connection.status)).toEqual("unreachable");
    const start = performance.now();
    await expect(client.channels.retrieve(["missing"])).rejects.toThrow(
      DisconnectedError,
    );
    expect(performance.now() - start).toBeLessThan(100);
    await client.close();
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
    await client.close();
  });
});

describe("close", () => {
  // The test harness closes every client again at file end, so a spy left
  // rejecting would fail the whole suite from afterAll.
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("should settle the connection status on disabled", async () => {
    const client = createTestClient();
    await client.connect();
    await client.close();
    expect(client.connection.status.variant).toEqual("disabled");
  });

  it("should close the connection when the cache close fails, rejecting instead of reporting internally", async () => {
    const onInternalError = vi.fn();
    const client = createTestClient({ onInternalError });
    await client.connect();
    // eslint-disable-next-line dot-notation -- bracket access reaches the private field
    vi.spyOn(client["cache"], "close").mockRejectedValue(new Error("cache boom"));
    await expect(client.close()).rejects.toThrow(AggregateError);
    expect(client.connection.status.variant).toEqual("disabled");
    expect(onInternalError).not.toHaveBeenCalled();
  });

  it("should carry every underlying failure in the rejection", async () => {
    const client = createTestClient();
    // eslint-disable-next-line dot-notation -- bracket access reaches the private field
    vi.spyOn(client["conn"], "close").mockRejectedValue(new Error("conn boom"));
    // eslint-disable-next-line dot-notation -- bracket access reaches the private field
    vi.spyOn(client["cache"], "close").mockRejectedValue(new Error("cache boom"));
    const error = await client.close().then(
      () => null,
      (e: unknown) => e,
    );
    assert(error instanceof AggregateError);
    expect(error.errors.map((e) => (e as Error).message)).toEqual([
      "conn boom",
      "cache boom",
    ]);
  });
});
