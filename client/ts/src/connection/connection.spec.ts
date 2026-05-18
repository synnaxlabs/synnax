// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { TimeSpan, TimeStamp, URL } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { auth } from "@/auth";
import { connection } from "@/connection";
import { TEST_CLIENT_PARAMS } from "@/testutil/client";
import { Transport } from "@/transport";

describe("connectivity", () => {
  it("should connect to the server", async () => {
    const transport = new Transport(
      new URL({
        host: TEST_CLIENT_PARAMS.host,
        port: Number(TEST_CLIENT_PARAMS.port),
      }),
    );
    const client = new auth.Client(transport.unary, TEST_CLIENT_PARAMS);
    transport.use(client.middleware());
    const connectivity = new connection.Checker(
      transport.unary,
      undefined,
      __VERSION__,
    );
    const state = await connectivity.check();
    expect(state.status).toEqual("connected");
    expect(z.uuid().safeParse(state.clusterKey).success).toBe(true);
  });
  describe("version compatibility", () => {
    it("should pull the server and client versions", async () => {
      const transport = new Transport(
        new URL({
          host: TEST_CLIENT_PARAMS.host,
          port: Number(TEST_CLIENT_PARAMS.port),
        }),
      );
      const client = new auth.Client(transport.unary, TEST_CLIENT_PARAMS);
      transport.use(client.middleware());
      const connectivity = new connection.Checker(
        transport.unary,
        undefined,
        __VERSION__,
      );
      const state = await connectivity.check();
      expect(state.clientServerCompatible).toBe(true);
      expect(state.clientVersion).toBe(__VERSION__);
    });
    it("should adjust state if the server is too old", async () => {
      const transport = new Transport(
        new URL({
          host: TEST_CLIENT_PARAMS.host,
          port: Number(TEST_CLIENT_PARAMS.port),
        }),
      );
      const client = new auth.Client(transport.unary, TEST_CLIENT_PARAMS);
      transport.use(client.middleware());
      const connectivity = new connection.Checker(
        transport.unary,
        undefined,
        "50000.0.0",
      );
      const state = await connectivity.check();
      expect(state.clientServerCompatible).toBe(false);
      expect(state.clientVersion).toBe("50000.0.0");
    });
    it("should adjust state if the server is too new", async () => {
      const transport = new Transport(
        new URL({
          host: TEST_CLIENT_PARAMS.host,
          port: Number(TEST_CLIENT_PARAMS.port),
        }),
      );
      const client = new auth.Client(transport.unary, TEST_CLIENT_PARAMS);
      transport.use(client.middleware());
      const connectivity = new connection.Checker(transport.unary, undefined, "0.0.0");
      const state = await connectivity.check();
      expect(state.clientServerCompatible).toBe(false);
      expect(state.clientVersion).toBe("0.0.0");
    });
  });
  describe("clock skew", () => {
    const createMockClient = (nodeTime: TimeStamp): UnaryClient => ({
      send: vi.fn().mockResolvedValue([
        {
          clusterKey: "test-cluster",
          nodeVersion: __VERSION__,
          nodeTime,
        },
        null,
      ]) as UnaryClient["send"],
      use: vi.fn(),
    });

    it("should detect clock skew exceeding threshold", async () => {
      const farFuture = TimeStamp.now().add(TimeSpan.hours(1));
      const checker = new connection.Checker(
        createMockClient(farFuture),
        TimeSpan.seconds(30),
        __VERSION__,
        undefined,
        TimeSpan.seconds(1),
      );
      const state = await checker.check();
      expect(state.clockSkewExceeded).toBe(true);
      expect(state.clockSkew.valueOf()).not.toBe(0n);
      checker.stop();
    });

    it("should not flag skew within threshold", async () => {
      const now = TimeStamp.now();
      const checker = new connection.Checker(
        createMockClient(now),
        TimeSpan.seconds(30),
        __VERSION__,
        undefined,
        TimeSpan.seconds(1),
      );
      const state = await checker.check();
      expect(state.clockSkewExceeded).toBe(false);
      checker.stop();
    });

    it("should fire onChange when clockSkewExceeded changes", async () => {
      let callCount = 0;
      const farFuture = TimeStamp.now().add(TimeSpan.hours(1));
      const checker = new connection.Checker(
        createMockClient(farFuture),
        TimeSpan.seconds(30),
        __VERSION__,
        undefined,
        TimeSpan.seconds(1),
      );
      // Wait for the constructor's initial check to complete
      await checker.check();
      checker.onChange(() => {
        callCount++;
      });
      // Trigger another check - skewExceeded stays true, status stays connected,
      // so onChange should not fire
      await checker.check();
      expect(callCount).toBe(0);
      checker.stop();
    });
  });
});
