// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient, Unreachable } from "@synnaxlabs/freighter";
import { TimeSpan, TimeStamp, url } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { auth } from "@/auth";
import { connection } from "@/connection";
import { AuthError } from "@/errors";
import { TEST_CLIENT_PARAMS } from "@/testutil";
import { Transport } from "@/transport";

const liveUnary = (): UnaryClient => {
  const transport = new Transport(
    new url.URL({
      host: TEST_CLIENT_PARAMS.host,
      port: Number(TEST_CLIENT_PARAMS.port),
    }),
  );
  const client = new auth.Client(transport.unary, TEST_CLIENT_PARAMS);
  transport.use(client.middleware());
  return transport.unary;
};

const mockUnary = (nodeTime: () => TimeStamp): UnaryClient => ({
  send: vi.fn().mockImplementation(async () => ({
    clusterKey: "test-cluster",
    nodeVersion: __VERSION__,
    nodeTime: nodeTime(),
  })),
  use: vi.fn(),
});

const failingUnary = (): UnaryClient => ({
  send: vi.fn().mockRejectedValue(new Unreachable({ message: "server down" })),
  use: vi.fn(),
});

const FAST_RETRY = { baseInterval: TimeSpan.milliseconds(1), scale: 1 };

describe("connection", () => {
  describe("check", () => {
    it("should connect to the server", async () => {
      const machine = new connection.Machine({
        unary: liveUnary(),
        clientVersion: __VERSION__,
      });
      const state = await machine.check();
      expect(state.variant).toEqual("success");
      expect(state.authenticated).toBe(true);
      expect(z.uuid().safeParse(state.clusterKey).success).toBe(true);
      await machine.stop();
    });

    it("should pull the server and client versions", async () => {
      const machine = new connection.Machine({
        unary: liveUnary(),
        clientVersion: __VERSION__,
      });
      const state = await machine.check();
      expect(state.clientServerCompatible).toBe(true);
      expect(state.clientVersion).toBe(__VERSION__);
      await machine.stop();
    });

    it("should adjust state if the server is too old", async () => {
      const machine = new connection.Machine({
        unary: liveUnary(),
        clientVersion: "50000.0.0",
      });
      const state = await machine.check();
      expect(state.clientServerCompatible).toBe(false);
      expect(state.clientVersion).toBe("50000.0.0");
      await machine.stop();
    });

    it("should adjust state if the server is too new", async () => {
      const machine = new connection.Machine({
        unary: liveUnary(),
        clientVersion: "0.0.0",
      });
      const state = await machine.check();
      expect(state.clientServerCompatible).toBe(false);
      await machine.stop();
    });
  });

  describe("clock skew", () => {
    it("should detect clock skew exceeding threshold", async () => {
      const machine = new connection.Machine({
        unary: mockUnary(() => TimeStamp.now().add(TimeSpan.hours(1))),
        clientVersion: __VERSION__,
        clockSkewThreshold: TimeSpan.seconds(1),
      });
      const state = await machine.check();
      expect(state.clockSkewExceeded).toBe(true);
      expect(state.clockSkew.valueOf()).not.toBe(0n);
      await machine.stop();
    });

    it("should not flag skew within threshold", async () => {
      const machine = new connection.Machine({
        unary: mockUnary(() => TimeStamp.now()),
        clientVersion: __VERSION__,
        clockSkewThreshold: TimeSpan.seconds(1),
      });
      const state = await machine.check();
      expect(state.clockSkewExceeded).toBe(false);
      await machine.stop();
    });
  });

  describe("transitions", () => {
    it("should start in loading and reach success without a stream feeder", async () => {
      const machine = new connection.Machine({
        unary: liveUnary(),
        clientVersion: __VERSION__,
      });
      expect(machine.state.variant).toEqual("loading");
      machine.start();
      const state = await machine.awaitConnected(TimeSpan.seconds(5));
      expect(state.variant).toEqual("success");
      await machine.stop();
    });

    it("should stay loading until the stream comes up when one is required", async () => {
      let bringUps = 0;
      const machine = new connection.Machine({
        unary: mockUnary(() => TimeStamp.now()),
        clientVersion: __VERSION__,
        bringUpStream: async () => {
          bringUps++;
        },
      });
      await machine.check();
      expect(machine.state.variant).toEqual("loading");
      expect(bringUps).toBeGreaterThan(0);
      machine.notifyStreamLive();
      expect(machine.state.variant).toEqual("success");
      expect(machine.state.streamLive).toBe(true);
      await machine.stop();
    });

    it("should escalate to error(unreachable) after the escalation budget", async () => {
      const machine = new connection.Machine({
        unary: failingUnary(),
        clientVersion: __VERSION__,
        retry: FAST_RETRY,
        escalateAfter: 2,
      });
      machine.start();
      await expect(machine.awaitConnected(TimeSpan.seconds(5))).rejects.toThrow();
      const { state } = machine;
      expect(state.variant).toEqual("error");
      expect(state.reason).toEqual("unreachable");
      await machine.stop();
    });

    it("should keep probing after escalation and self-heal", async () => {
      let failing = true;
      const unary: UnaryClient = {
        send: vi.fn().mockImplementation(async () => {
          if (failing) throw new Unreachable({ message: "server down" });
          return {
            clusterKey: "test-cluster",
            nodeVersion: __VERSION__,
            nodeTime: TimeStamp.now(),
          };
        }),
        use: vi.fn(),
      };
      const machine = new connection.Machine({
        unary,
        clientVersion: __VERSION__,
        retry: FAST_RETRY,
        escalateAfter: 1,
      });
      machine.start();
      await machine.waitFor((s) => s.variant === "error", TimeSpan.seconds(5));
      failing = false;
      const state = await machine.waitFor(
        (s) => s.variant === "success",
        TimeSpan.seconds(5),
      );
      expect(state.reason).toBeUndefined();
      await machine.stop();
    });

    it("should park in error(unreachable) when a finite budget exhausts", async () => {
      const unary = failingUnary();
      const machine = new connection.Machine({
        unary,
        clientVersion: __VERSION__,
        retry: { ...FAST_RETRY, maxRetries: 3 },
        escalateAfter: 2,
      });
      machine.start();
      await machine.waitFor((s) => s.variant === "error", TimeSpan.seconds(5));
      await expect(
        machine.waitFor((s) => s.retry == null, TimeSpan.seconds(5)),
      ).resolves.toBeTruthy();
      const calls = (unary.send as ReturnType<typeof vi.fn>).mock.calls.length;
      await new Promise((resolve) => setTimeout(resolve, 20));
      expect((unary.send as ReturnType<typeof vi.fn>).mock.calls.length).toBe(calls);
      machine.retryNow();
      await new Promise((resolve) => setTimeout(resolve, 20));
      expect(
        (unary.send as ReturnType<typeof vi.fn>).mock.calls.length,
      ).toBeGreaterThan(calls);
      await machine.stop();
    });

    it("should transition success -> warning on stream drop and back on reopen", async () => {
      const machine = new connection.Machine({
        unary: mockUnary(() => TimeStamp.now()),
        clientVersion: __VERSION__,
        bringUpStream: async () => {},
      });
      await machine.check();
      machine.notifyStreamLive();
      expect(machine.state.variant).toEqual("success");
      machine.notifyStreamDrop(new Error("socket died"));
      expect(machine.state.variant).toEqual("warning");
      expect(machine.state.streamLive).toBe(false);
      machine.notifyStreamLive();
      expect(machine.state.variant).toEqual("success");
      await machine.stop();
    });

    it("should enter error(auth) on auth failure and recover via reauthenticate", async () => {
      const machine = new connection.Machine({
        unary: mockUnary(() => TimeStamp.now()),
        clientVersion: __VERSION__,
      });
      machine.notifyAuthFailure(new AuthError("invalid credentials"));
      expect(machine.state.variant).toEqual("error");
      expect(machine.state.reason).toEqual("auth");
      expect(machine.state.authenticated).toBe(false);
      const setCredentials = vi.fn();
      machine.reauthenticate(setCredentials);
      expect(setCredentials).toHaveBeenCalledOnce();
      expect(machine.state.variant).toEqual("loading");
      await machine.stop();
    });

    it("should reject awaitConnected with the stored auth error", async () => {
      const machine = new connection.Machine({
        unary: failingUnary(),
        clientVersion: __VERSION__,
      });
      machine.notifyAuthFailure(new AuthError("bad password"));
      await expect(machine.awaitConnected()).rejects.toThrow(AuthError);
    });

    it("should mirror epochs into the state", async () => {
      const machine = new connection.Machine({
        unary: mockUnary(() => TimeStamp.now()),
        clientVersion: __VERSION__,
      });
      expect(machine.state.epoch).toBe(0);
      machine.ingestEpoch(2);
      expect(machine.state.epoch).toBe(2);
      await machine.stop();
    });

    it("should become disabled on stop", async () => {
      const machine = new connection.Machine({
        unary: mockUnary(() => TimeStamp.now()),
        clientVersion: __VERSION__,
      });
      machine.start();
      await machine.stop();
      expect(machine.state.variant).toEqual("disabled");
    });
  });

  describe("onChange", () => {
    it("should notify on transitions and support unsubscribe", async () => {
      const machine = new connection.Machine({
        unary: mockUnary(() => TimeStamp.now()),
        clientVersion: __VERSION__,
      });
      const states: connection.State[] = [];
      const detach = machine.onChange((state) => states.push(state));
      await machine.check();
      expect(states.some((s) => s.variant === "success")).toBe(true);
      const count = states.length;
      detach();
      machine.notifyStreamDrop(new Error("dropped"));
      expect(states.length).toBe(count);
      await machine.stop();
    });

    it("should time out awaitConnected against a dead cluster", async () => {
      const machine = new connection.Machine({
        unary: failingUnary(),
        clientVersion: __VERSION__,
        retry: { ...FAST_RETRY, maxRetries: Infinity },
        escalateAfter: Infinity,
      });
      machine.start();
      await expect(machine.awaitConnected(TimeSpan.milliseconds(50))).rejects.toThrow();
      await machine.stop();
    });
  });
});
