// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient, Unreachable } from "@synnaxlabs/freighter";
import { type breaker, observe, TimeSpan, TimeStamp, url } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { auth } from "@/auth";
import { connection } from "@/connection";
import { AuthError } from "@/errors";
import { type status } from "@/status";
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

const FAST_RETRY: breaker.Config = { baseInterval: TimeSpan.milliseconds(1), scale: 1 };

const createConfig = (
  overrides: Partial<connection.Config> = {},
): connection.Config => ({
  clientVersion: __VERSION__,
  name: "test-cluster",
  escalateAfter: connection.DEFAULT_ESCALATE_AFTER,
  clockSkewThreshold: TimeSpan.seconds(1),
  requiresStream: false,
  ...overrides,
});

const createInfo = (overrides: Partial<connection.Info> = {}): connection.Info => ({
  clusterKey: "test-cluster",
  nodeVersion: __VERSION__,
  clockSkew: TimeSpan.ZERO,
  ...overrides,
});

/** Folds events over an initial state, mirroring the client's dispatch. */
const apply = (
  config: connection.Config,
  ...events: connection.Event[]
): connection.State =>
  events.reduce(
    (state, event) => connection.reduce(state, event, config),
    connection.createInitialState(config),
  );

interface Harness {
  readonly state: connection.State;
  readonly handle: connection.Handle;
  readonly prober: connection.Prober;
  dispatch: (event: connection.Event) => void;
  connect: (timeout?: TimeSpan) => Promise<connection.State>;
  stop: () => Promise<void>;
}

/**
 * Composes the reducer, the prober, and an observer exactly as the client does,
 * so loop-level behavior is exercised through the same wiring production uses.
 */
const createHarness = (
  unary: UnaryClient,
  overrides: Partial<connection.Config> = {},
  retry: breaker.Config = FAST_RETRY,
): Harness => {
  const config = createConfig(overrides);
  let state = connection.createInitialState(config);
  const observer = new observe.Observer<connection.State>();
  const dispatch = (event: connection.Event): void => {
    const prev = state;
    state = connection.reduce(prev, event, config);
    if (connection.materialChange(prev, state)) observer.notify(state);
    prober.setMode(connection.modeFor(state));
  };
  const prober = new connection.Prober({
    client: new connection.Client(unary),
    retry,
    heartbeatInterval: TimeSpan.milliseconds(20),
    emit: dispatch,
  });
  const handle: connection.Handle = {
    get state() {
      return state;
    },
    onChange: (callback) => observer.onChange(callback),
    retryNow: () => {
      dispatch({ type: "retry.requested" });
      prober.retryNow();
    },
  };
  prober.start();
  return {
    get state() {
      return state;
    },
    handle,
    prober,
    dispatch,
    connect: async (timeout?: TimeSpan) =>
      await connection.awaitConnected(handle, timeout),
    stop: async () => {
      dispatch({ type: "closed" });
      await prober.stop();
    },
  };
};

const waitForState = async (
  handle: connection.Handle,
  predicate: (state: connection.State) => boolean,
  timeout: TimeSpan = TimeSpan.seconds(5),
): Promise<connection.State> => {
  if (predicate(handle.state)) return handle.state;
  return await new Promise<connection.State>((resolve, reject) => {
    const timer = setTimeout(() => {
      detach();
      reject(new Error("timed out waiting for connection state"));
    }, timeout.milliseconds);
    const detach = handle.onChange((state) => {
      if (!predicate(state)) return;
      clearTimeout(timer);
      detach();
      resolve(state);
    });
  });
};

describe("connection", () => {
  describe("Client", () => {
    it("should probe the cluster", async () => {
      const info = await new connection.Client(liveUnary()).check();
      expect(z.uuid().safeParse(info.clusterKey).success).toBe(true);
      expect(info.nodeVersion).not.toBeUndefined();
    });

    it("should fold a probe into a connected state", async () => {
      const config = createConfig();
      const info = await new connection.Client(liveUnary()).check();
      const state = apply(config, { type: "probe.success", info });
      expect(state.variant).toEqual("success");
      expect(state.details.authenticated).toBe(true);
      expect(z.uuid().safeParse(state.details.clusterKey).success).toBe(true);
    });

    it("should pull the server and client versions", async () => {
      const config = createConfig();
      const info = await new connection.Client(liveUnary()).check();
      const state = apply(config, { type: "probe.success", info });
      expect(state.details.clientServerCompatible).toBe(true);
      expect(state.details.clientVersion).toBe(__VERSION__);
    });

    it("should adjust state if the server is too old", async () => {
      const config = createConfig({ clientVersion: "50000.0.0" });
      const info = await new connection.Client(liveUnary()).check();
      const state = apply(config, { type: "probe.success", info });
      expect(state.details.clientServerCompatible).toBe(false);
      expect(state.details.clientVersion).toBe("50000.0.0");
    });

    it("should adjust state if the server is too new", async () => {
      const config = createConfig({ clientVersion: "0.0.0" });
      const info = await new connection.Client(liveUnary()).check();
      const state = apply(config, { type: "probe.success", info });
      expect(state.details.clientServerCompatible).toBe(false);
    });

    it("should propagate transport failures", async () => {
      await expect(new connection.Client(failingUnary()).check()).rejects.toThrow(
        Unreachable,
      );
    });
  });

  describe("clock skew", () => {
    it("should detect clock skew exceeding threshold", async () => {
      const config = createConfig({ clockSkewThreshold: TimeSpan.seconds(1) });
      const client = new connection.Client(
        mockUnary(() => TimeStamp.now().add(TimeSpan.hours(1))),
      );
      const state = apply(config, {
        type: "probe.success",
        info: await client.check(),
      });
      expect(state.details.clockSkewExceeded).toBe(true);
      expect(state.details.clockSkew.valueOf()).not.toBe(0n);
    });

    it("should not flag skew within threshold", async () => {
      const config = createConfig({ clockSkewThreshold: TimeSpan.seconds(1) });
      const client = new connection.Client(mockUnary(() => TimeStamp.now()));
      const state = apply(config, {
        type: "probe.success",
        info: await client.check(),
      });
      expect(state.details.clockSkewExceeded).toBe(false);
    });
  });

  describe("reduce", () => {
    it("should start in loading", () => {
      expect(connection.createInitialState(createConfig()).variant).toEqual("loading");
    });

    it("should reach success on a probe when no stream is required", () => {
      const state = apply(createConfig(), {
        type: "probe.success",
        info: createInfo(),
      });
      expect(state.variant).toEqual("success");
      expect(state.details.clusterKey).toEqual("test-cluster");
    });

    it("should stay loading until the stream comes up when one is required", () => {
      const config = createConfig({ requiresStream: true });
      const probed = apply(config, { type: "probe.success", info: createInfo() });
      expect(probed.variant).toEqual("loading");
      expect(probed.details.authenticated).toBe(true);
      const live = connection.reduce(probed, { type: "stream.live" }, config);
      expect(live.variant).toEqual("success");
      expect(live.details.streamLive).toBe(true);
    });

    it("should escalate to error(unreachable) after the escalation budget", () => {
      const config = createConfig({ escalateAfter: 2 });
      const error = new Unreachable({ message: "server down" });
      const first = apply(config, { type: "probe.failure", error, attempt: 1 });
      expect(first.variant).toEqual("loading");
      const second = connection.reduce(
        first,
        { type: "probe.failure", error, attempt: 2 },
        config,
      );
      expect(second.variant).toEqual("error");
      expect(second.details.reason).toEqual("unreachable");
    });

    it("should escalate when a finite retry budget exhausts", () => {
      const config = createConfig({ escalateAfter: 100 });
      const error = new Unreachable({ message: "server down" });
      const failed = apply(config, { type: "probe.failure", error, attempt: 1 });
      const exhausted = connection.reduce(failed, { type: "retry.exhausted" }, config);
      expect(exhausted.variant).toEqual("error");
      expect(exhausted.details.reason).toEqual("unreachable");
      expect(exhausted.details.retry).toBeNull();
    });

    it("should transition success -> warning on stream drop and back on reopen", () => {
      const config = createConfig({ requiresStream: true });
      const connected = apply(
        config,
        { type: "probe.success", info: createInfo() },
        { type: "stream.live" },
      );
      expect(connected.variant).toEqual("success");
      const dropped = connection.reduce(
        connected,
        { type: "stream.drop", error: new Error("socket died") },
        config,
      );
      expect(dropped.variant).toEqual("warning");
      expect(dropped.details.streamLive).toBe(false);
      const reopened = connection.reduce(dropped, { type: "stream.live" }, config);
      expect(reopened.variant).toEqual("success");
    });

    it("should degrade to warning when a heartbeat fails while connected", () => {
      const config = createConfig();
      const connected = apply(config, { type: "probe.success", info: createInfo() });
      const degraded = connection.reduce(
        connected,
        {
          type: "probe.failure",
          error: new Unreachable({ message: "gone" }),
          attempt: 1,
        },
        config,
      );
      expect(degraded.variant).toEqual("warning");
    });

    it("should enter error(auth) on auth failure and recover on new credentials", () => {
      const config = createConfig();
      const rejected = apply(config, {
        type: "auth.failure",
        error: new AuthError("invalid credentials"),
      });
      expect(rejected.variant).toEqual("error");
      expect(rejected.details.reason).toEqual("auth");
      expect(rejected.details.authenticated).toBe(false);
      const retrying = connection.reduce(rejected, { type: "retry.requested" }, config);
      expect(retrying.variant).toEqual("error");
      expect(retrying.details.reason).toEqual("auth");
      const replaced = connection.reduce(
        rejected,
        { type: "credentials.replaced" },
        config,
      );
      expect(replaced.variant).toEqual("loading");
      expect(replaced.details.reason).toBeUndefined();
    });

    it("should treat an auth error from the probe as an auth failure", () => {
      const state = apply(createConfig(), {
        type: "probe.failure",
        error: new AuthError("bad password"),
        attempt: 1,
      });
      expect(state.details.reason).toEqual("auth");
    });

    it("should clear error(unreachable) on retry.requested", () => {
      const config = createConfig({ escalateAfter: 1 });
      const failed = apply(config, {
        type: "probe.failure",
        error: new Unreachable({ message: "server down" }),
        attempt: 1,
      });
      expect(failed.details.reason).toEqual("unreachable");
      const retrying = connection.reduce(failed, { type: "retry.requested" }, config);
      expect(retrying.variant).toEqual("loading");
    });

    it("should publish retry progress", () => {
      const nextAt = TimeStamp.now().add(TimeSpan.seconds(5));
      const state = apply(createConfig(), {
        type: "retry.scheduled",
        attempt: 3,
        nextAt,
      });
      expect(state.details.retry?.attempt).toEqual(3);
      expect(state.details.retry?.nextAt.valueOf()).toEqual(nextAt.valueOf());
    });

    it("should mirror epochs into the state", () => {
      const config = createConfig();
      expect(connection.createInitialState(config).details.epoch).toBe(0);
      expect(apply(config, { type: "epoch.advanced", epoch: 2 }).details.epoch).toBe(2);
    });

    it("should become disabled on close and stay there", () => {
      const config = createConfig();
      const closed = apply(config, { type: "closed" });
      expect(closed.variant).toEqual("disabled");
      const after = connection.reduce(
        closed,
        { type: "probe.success", info: createInfo() },
        config,
      );
      expect(after).toBe(closed);
    });

    it("should not notify on immaterial changes", () => {
      const config = createConfig();
      const connected = apply(config, { type: "probe.success", info: createInfo() });
      const again = connection.reduce(
        connected,
        { type: "probe.success", info: createInfo() },
        config,
      );
      expect(connection.materialChange(connected, again)).toBe(false);
    });
  });

  describe("modeFor", () => {
    it("should map variants onto probe modes", () => {
      const config = createConfig();
      const initial = connection.createInitialState(config);
      const withReason = (
        variant: status.Variant,
        reason?: connection.Reason,
      ): connection.State => ({
        ...initial,
        variant,
        details: { ...initial.details, reason },
      });
      expect(connection.modeFor(initial)).toEqual("probing");
      expect(connection.modeFor(withReason("success"))).toEqual("heartbeat");
      expect(connection.modeFor(withReason("warning"))).toEqual("probing");
      expect(connection.modeFor(withReason("error", "unreachable"))).toEqual("probing");
      expect(connection.modeFor(withReason("error", "auth"))).toEqual("idle");
      expect(connection.modeFor({ ...initial, variant: "disabled" })).toEqual("idle");
    });
  });

  describe("Prober", () => {
    it("should reach success against a live cluster", async () => {
      const harness = createHarness(liveUnary());
      const state = await harness.connect(TimeSpan.seconds(5));
      expect(state.variant).toEqual("success");
      await harness.stop();
    });

    it("should escalate to error(unreachable) after the escalation budget", async () => {
      const harness = createHarness(failingUnary(), { escalateAfter: 2 });
      await expect(harness.connect(TimeSpan.seconds(5))).rejects.toThrow();
      expect(harness.state.variant).toEqual("error");
      expect(harness.state.details.reason).toEqual("unreachable");
      await harness.stop();
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
      const harness = createHarness(unary, { escalateAfter: 1 });
      await waitForState(harness.handle, (s) => s.variant === "error");
      failing = false;
      const state = await waitForState(harness.handle, (s) => s.variant === "success");
      expect(state.details.reason).toBeUndefined();
      await harness.stop();
    });

    it("should park in error(unreachable) when a finite budget exhausts", async () => {
      const unary = failingUnary();
      const harness = createHarness(
        unary,
        { escalateAfter: 2 },
        {
          ...FAST_RETRY,
          maxRetries: 3,
        },
      );
      await waitForState(harness.handle, (s) => s.variant === "error");
      await waitForState(harness.handle, (s) => s.details.retry == null);
      const send = unary.send as ReturnType<typeof vi.fn>;
      const calls = send.mock.calls.length;
      await new Promise((resolve) => setTimeout(resolve, 20));
      expect(send.mock.calls.length).toBe(calls);
      harness.handle.retryNow();
      await new Promise((resolve) => setTimeout(resolve, 20));
      expect(send.mock.calls.length).toBeGreaterThan(calls);
      await harness.stop();
    });

    it("should stop probing once the connection is closed", async () => {
      const unary = failingUnary();
      const harness = createHarness(unary, { escalateAfter: 1 });
      await waitForState(harness.handle, (s) => s.variant === "error");
      await harness.stop();
      expect(harness.state.variant).toEqual("disabled");
      const send = unary.send as ReturnType<typeof vi.fn>;
      const calls = send.mock.calls.length;
      await new Promise((resolve) => setTimeout(resolve, 20));
      expect(send.mock.calls.length).toBe(calls);
    });

    it("should idle rather than probe while parked on an auth error", async () => {
      const unary = mockUnary(() => TimeStamp.now());
      const harness = createHarness(unary);
      await waitForState(harness.handle, (s) => s.variant === "success");
      harness.dispatch({ type: "auth.failure", error: new AuthError("bad password") });
      const send = unary.send as ReturnType<typeof vi.fn>;
      const calls = send.mock.calls.length;
      await new Promise((resolve) => setTimeout(resolve, 60));
      expect(send.mock.calls.length).toBe(calls);
      await harness.stop();
    });
  });

  describe("handle", () => {
    it("should notify on transitions and support unsubscribe", async () => {
      const harness = createHarness(mockUnary(() => TimeStamp.now()));
      const states: connection.State[] = [];
      const detach = harness.handle.onChange((state) => states.push(state));
      await waitForState(harness.handle, (s) => s.variant === "success");
      expect(states.some((s) => s.variant === "success")).toBe(true);
      const count = states.length;
      detach();
      harness.dispatch({ type: "stream.drop", error: new Error("dropped") });
      expect(states.length).toBe(count);
      await harness.stop();
    });

    it("should reject connect with the stored auth error", async () => {
      const harness = createHarness(failingUnary());
      harness.dispatch({
        type: "auth.failure",
        error: new AuthError("bad password"),
      });
      await expect(harness.connect()).rejects.toThrow(AuthError);
      await harness.stop();
    });

    it("should time out connect against a dead cluster", async () => {
      const harness = createHarness(
        failingUnary(),
        { escalateAfter: Infinity },
        {
          ...FAST_RETRY,
          maxRetries: Infinity,
        },
      );
      await expect(harness.connect(TimeSpan.milliseconds(50))).rejects.toThrow();
      await harness.stop();
    });
  });
});
