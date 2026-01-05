// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { z } from "zod";

import { checkConnection, newConnectionChecker } from "@/client";
import { TEST_CLIENT_PARAMS } from "@/testutil/client";

describe("checkConnection", () => {
  it("should check connection to the server", async () => {
    const state = await checkConnection({
      host: TEST_CLIENT_PARAMS.host,
      port: TEST_CLIENT_PARAMS.port,
      secure: false,
    });
    expect(state.status).toEqual("connected");
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
    expect(state.status).toEqual("connected");
  });

  it("should handle connection failure to invalid host", async () => {
    const state = await checkConnection({
      host: "invalid-host-that-does-not-exist",
      port: 9999,
      secure: false,
      retry: {
        maxRetries: 0, // Disable retries for faster test
      },
    });
    expect(state.status).toEqual("failed");
  });

  it("should handle connection failure to invalid port", async () => {
    const state = await checkConnection({
      host: TEST_CLIENT_PARAMS.host,
      port: 9999, // Wrong port
      secure: false,
      retry: {
        maxRetries: 0, // Disable retries for faster test
      },
    });
    expect(state.status).toEqual("failed");
  });
});

describe("newConnectionChecker", () => {
  it("should create a connection checker", () => {
    const checker = newConnectionChecker({
      host: TEST_CLIENT_PARAMS.host,
      port: TEST_CLIENT_PARAMS.port,
      secure: false,
    });
    expect(checker).toBeDefined();
  });

  it("should create a checker that can check connection", async () => {
    const checker = newConnectionChecker({
      host: TEST_CLIENT_PARAMS.host,
      port: TEST_CLIENT_PARAMS.port,
      secure: false,
    });
    const state = await checker.check();
    expect(state.status).toEqual("connected");
    expect(z.uuid().safeParse(state.clusterKey).success).toBe(true);
  });

  it("should support custom name parameter", async () => {
    const checker = newConnectionChecker({
      host: TEST_CLIENT_PARAMS.host,
      port: TEST_CLIENT_PARAMS.port,
      secure: false,
      name: "custom-checker-name",
    });
    const state = await checker.check();
    expect(state.status).toEqual("connected");
  });

  it("should support secure connection parameter", () => {
    const checker = newConnectionChecker({
      host: TEST_CLIENT_PARAMS.host,
      port: TEST_CLIENT_PARAMS.port,
      secure: true,
    });
    expect(checker).toBeDefined();
  });

  it("should create multiple independent checkers", async () => {
    const checker1 = newConnectionChecker({
      host: TEST_CLIENT_PARAMS.host,
      port: TEST_CLIENT_PARAMS.port,
      secure: false,
      name: "checker-1",
    });
    const checker2 = newConnectionChecker({
      host: TEST_CLIENT_PARAMS.host,
      port: TEST_CLIENT_PARAMS.port,
      secure: false,
      name: "checker-2",
    });

    const state1 = await checker1.check();
    const state2 = await checker2.check();

    expect(state1.status).toEqual("connected");
    expect(state2.status).toEqual("connected");
    expect(checker1).not.toBe(checker2); // Different instances
  });

  it("should handle version compatibility checking", async () => {
    const checker = newConnectionChecker({
      host: TEST_CLIENT_PARAMS.host,
      port: TEST_CLIENT_PARAMS.port,
      secure: false,
    });
    const state = await checker.check();
    expect(state.clientVersion).toBeDefined();
    expect(state.clientServerCompatible).toBe(true);
  });
});
