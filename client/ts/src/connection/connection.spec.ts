// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient, Unreachable } from "@synnaxlabs/freighter";
import { type breaker, TimeSpan, TimeStamp, url } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { auth } from "@/auth";
import { connection } from "@/connection";
import { modeFor, probe } from "@/connection/client";
import {
  createInitialStatus,
  type Event,
  materialChange,
  reduce,
} from "@/connection/status";
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

const FAST_RETRY: breaker.Config = { baseInterval: TimeSpan.milliseconds(1), scale: 1 };

const createConfig = (
  overrides: Partial<connection.Config> = {},
): connection.Config => ({
  clientVersion: __VERSION__,
  name: "test-cluster",
  escalateAfter: 4,
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

/** Folds events over an initial status, mirroring the client's dispatch. */
const apply = (config: connection.Config, ...events: Event[]): connection.Status =>
  events.reduce(
    (status, event) => reduce(status, event, config),
    createInitialStatus(config),
  );

const createClient = (
  unary: UnaryClient,
  overrides: Partial<connection.ClientParams> = {},
  retry: breaker.Config = FAST_RETRY,
  heartbeatInterval: TimeSpan = TimeSpan.milliseconds(20),
): connection.Client =>
  new connection.Client({
    unary,
    address: "localhost:9090",
    name: "test-cluster",
    retry,
    heartbeatInterval,
    ...overrides,
  });

const waitForStatus = async (
  handle: connection.Handle,
  predicate: (status: connection.Status) => boolean,
  timeout: TimeSpan = TimeSpan.seconds(5),
): Promise<connection.Status> => {
  if (predicate(handle.status)) return handle.status;
  return await new Promise<connection.Status>((resolve, reject) => {
    const timer = setTimeout(() => {
      detach();
      reject(new Error("timed out waiting for connection status"));
    }, timeout.milliseconds);
    const detach = handle.onChange((status) => {
      if (!predicate(status)) return;
      clearTimeout(timer);
      detach();
      resolve(status);
    });
  });
};

const sleep = async (span: TimeSpan): Promise<void> =>
  await new Promise((resolve) => setTimeout(resolve, span.milliseconds));

describe("connection", () => {
  describe("probe", () => {
    it("should probe the cluster", async () => {
      const info = await probe(liveUnary());
      expect(z.uuid().safeParse(info.clusterKey).success).toBe(true);
      expect(info.nodeVersion).not.toBeUndefined();
    });

    it("should fold a probe into a connected status", async () => {
      const config = createConfig();
      const info = await probe(liveUnary());
      const status = apply(config, { type: "probe.success", info });
      expect(status.variant).toEqual("success");
      expect(status.details.authenticated).toBe(true);
      expect(z.uuid().safeParse(status.details.clusterKey).success).toBe(true);
    });

    it("should pull the server and client versions", async () => {
      const config = createConfig();
      const info = await probe(liveUnary());
      const status = apply(config, { type: "probe.success", info });
      expect(status.details.clientServerCompatible).toBe(true);
      expect(status.details.clientVersion).toBe(__VERSION__);
    });

    it("should adjust status if the server is too old", async () => {
      const config = createConfig({ clientVersion: "50000.0.0" });
      const info = await probe(liveUnary());
      const status = apply(config, { type: "probe.success", info });
      expect(status.details.clientServerCompatible).toBe(false);
      expect(status.details.clientVersion).toBe("50000.0.0");
    });

    it("should adjust status if the server is too new", async () => {
      const config = createConfig({ clientVersion: "0.0.0" });
      const info = await probe(liveUnary());
      const status = apply(config, { type: "probe.success", info });
      expect(status.details.clientServerCompatible).toBe(false);
    });

    it("should propagate transport failures", async () => {
      await expect(probe(failingUnary())).rejects.toThrow(Unreachable);
    });
  });

  describe("check", () => {
    it("should return a success status against a live cluster", async () => {
      const status = await connection.check({
        host: TEST_CLIENT_PARAMS.host,
        port: TEST_CLIENT_PARAMS.port,
        name: "test-cluster",
      });
      expect(status.variant).toEqual("success");
      expect(status.details.authenticated).toBe(true);
      expect(z.uuid().safeParse(status.details.clusterKey).success).toBe(true);
    });

    it("should include the client version in the check", async () => {
      const status = await connection.check({
        host: TEST_CLIENT_PARAMS.host,
        port: TEST_CLIENT_PARAMS.port,
      });
      expect(status.details.clientVersion).toBeDefined();
      expect(status.details.clientServerCompatible).toBe(true);
    });

    it("should return an error status against an unreachable host", async () => {
      const status = await connection.check({
        host: "invalid-host-that-does-not-exist",
        port: 9999,
        retry: { maxRetries: 0 },
      });
      expect(status.variant).toEqual("error");
      expect(status.details.reason).toEqual("unreachable");
    });

    it("should return an error status against a dead port", async () => {
      const status = await connection.check({
        host: TEST_CLIENT_PARAMS.host,
        port: 9999,
        retry: { maxRetries: 0 },
      });
      expect(status.variant).toEqual("error");
    });
  });

  describe("clock skew", () => {
    it("should detect clock skew exceeding threshold", async () => {
      const config = createConfig({ clockSkewThreshold: TimeSpan.seconds(1) });
      const unary = mockUnary(() => TimeStamp.now().add(TimeSpan.hours(1)));
      const status = apply(config, {
        type: "probe.success",
        info: await probe(unary),
      });
      expect(status.details.clockSkewExceeded).toBe(true);
      expect(status.details.clockSkew.valueOf()).not.toBe(0n);
    });

    it("should not flag skew within threshold", async () => {
      const config = createConfig({ clockSkewThreshold: TimeSpan.seconds(1) });
      const unary = mockUnary(() => TimeStamp.now());
      const status = apply(config, {
        type: "probe.success",
        info: await probe(unary),
      });
      expect(status.details.clockSkewExceeded).toBe(false);
    });
  });

  describe("reduce", () => {
    it("should start in loading", () => {
      expect(createInitialStatus(createConfig()).variant).toEqual("loading");
    });

    it("should track an in-flight probe without moving the variant", () => {
      const config = createConfig({ escalateAfter: 1 });
      const error = new Unreachable({ message: "server down" });
      const parked = apply(config, { type: "probe.failure", error, attempt: 1 });
      expect(parked.variant).toEqual("error");
      const probing = reduce(parked, { type: "probe.started" }, config);
      expect(probing.variant).toEqual("error");
      expect(probing.details.probing).toBe(true);
      const failed = reduce(
        probing,
        { type: "probe.failure", error, attempt: 2 },
        config,
      );
      expect(failed.details.probing).toBe(false);
      const succeeded = reduce(
        probing,
        { type: "probe.success", info: createInfo() },
        config,
      );
      expect(succeeded.details.probing).toBe(false);
      expect(succeeded.variant).toEqual("success");
    });

    it("should reach success on a probe when no stream is required", () => {
      const status = apply(createConfig(), {
        type: "probe.success",
        info: createInfo(),
      });
      expect(status.variant).toEqual("success");
      expect(status.details.clusterKey).toEqual("test-cluster");
    });

    it("should stay loading until the stream comes up when one is required", () => {
      const config = createConfig({ requiresStream: true });
      const probed = apply(config, { type: "probe.success", info: createInfo() });
      expect(probed.variant).toEqual("loading");
      expect(probed.details.authenticated).toBe(true);
      const live = reduce(probed, { type: "stream.live" }, config);
      expect(live.variant).toEqual("success");
      expect(live.details.streamLive).toBe(true);
    });

    it("should lift error(unreachable) to warning on probe success with a dark stream", () => {
      const config = createConfig({ requiresStream: true, escalateAfter: 1 });
      const error = new Unreachable({ message: "server down" });
      const parked = apply(config, { type: "probe.failure", error, attempt: 1 });
      expect(parked.variant).toEqual("error");
      // reachable again: the error must lift, or its short circuit starves
      // the stream reopen that success waits on
      const probed = reduce(
        parked,
        { type: "probe.success", info: createInfo() },
        config,
      );
      expect(probed.variant).toEqual("warning");
      expect(probed.details.reason).toBeUndefined();
      const live = reduce(probed, { type: "stream.live" }, config);
      expect(live.variant).toEqual("success");
    });

    it("should escalate to error(unreachable) after the escalation budget", () => {
      const config = createConfig({ escalateAfter: 2 });
      const error = new Unreachable({ message: "server down" });
      const first = apply(config, { type: "probe.failure", error, attempt: 1 });
      expect(first.variant).toEqual("loading");
      const second = reduce(
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
      const exhausted = reduce(failed, { type: "retry.exhausted" }, config);
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
      const dropped = reduce(
        connected,
        { type: "stream.drop", error: new Error("socket died") },
        config,
      );
      expect(dropped.variant).toEqual("warning");
      expect(dropped.details.streamLive).toBe(false);
      const reopened = reduce(dropped, { type: "stream.live" }, config);
      expect(reopened.variant).toEqual("success");
    });

    it("should degrade to warning when a heartbeat fails while connected", () => {
      const config = createConfig();
      const connected = apply(config, { type: "probe.success", info: createInfo() });
      const degraded = reduce(
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
      const retrying = reduce(rejected, { type: "retry.requested" }, config);
      expect(retrying.variant).toEqual("error");
      expect(retrying.details.reason).toEqual("auth");
      const replaced = reduce(rejected, { type: "credentials.replaced" }, config);
      expect(replaced.variant).toEqual("loading");
      expect(replaced.details.reason).toBeUndefined();
    });

    it("should treat an auth error from the probe as an auth failure", () => {
      const status = apply(createConfig(), {
        type: "probe.failure",
        error: new AuthError("bad password"),
        attempt: 1,
      });
      expect(status.details.reason).toEqual("auth");
    });

    it("should clear error(unreachable) on retry.requested", () => {
      const config = createConfig({ escalateAfter: 1 });
      const failed = apply(config, {
        type: "probe.failure",
        error: new Unreachable({ message: "server down" }),
        attempt: 1,
      });
      expect(failed.details.reason).toEqual("unreachable");
      const retrying = reduce(failed, { type: "retry.requested" }, config);
      expect(retrying.variant).toEqual("loading");
    });

    it("should publish retry progress", () => {
      const nextAt = TimeStamp.now().add(TimeSpan.seconds(5));
      const status = apply(createConfig(), {
        type: "retry.scheduled",
        attempt: 3,
        nextAt,
      });
      expect(status.details.retry?.attempt).toEqual(3);
      expect(status.details.retry?.nextAt.valueOf()).toEqual(nextAt.valueOf());
    });

    it("should mirror epochs into the status", () => {
      const config = createConfig();
      expect(createInitialStatus(config).details.epoch).toBe(0);
      expect(apply(config, { type: "epoch.advanced", epoch: 2 }).details.epoch).toBe(2);
    });

    it("should return to first contact when a probe answers with a new cluster", () => {
      const config = createConfig({ requiresStream: true });
      const connected = apply(
        config,
        { type: "probe.success", info: createInfo() },
        { type: "stream.live" },
        { type: "epoch.advanced", epoch: 1 },
      );
      expect(connected.variant).toEqual("success");
      const replaced = reduce(
        connected,
        {
          type: "probe.success",
          info: createInfo({ clusterKey: "other-cluster" }),
        },
        config,
      );
      expect(replaced.variant).toEqual("loading");
      expect(replaced.details.clusterKey).toEqual("other-cluster");
      expect(replaced.details.epoch).toBe(0);
      expect(replaced.details.streamLive).toBe(false);
      expect(replaced.details.retry).toBeNull();
    });

    it("should clear a parked error on cluster replacement", () => {
      const config = createConfig();
      const parked = apply(
        config,
        { type: "probe.success", info: createInfo() },
        { type: "probe.failure", error: new Error("gone"), attempt: 10 },
        { type: "retry.exhausted" },
      );
      expect(parked.variant).toEqual("error");
      const replaced = reduce(
        parked,
        {
          type: "probe.success",
          info: createInfo({ clusterKey: "other-cluster" }),
        },
        config,
      );
      expect(replaced.variant).toEqual("loading");
      expect(replaced.details.reason).toBeUndefined();
      expect(replaced.details.error).toBeUndefined();
    });

    it("should become disabled on close and stay there", () => {
      const config = createConfig();
      const closed = apply(config, { type: "closed" });
      expect(closed.variant).toEqual("disabled");
      const after = reduce(
        closed,
        { type: "probe.success", info: createInfo() },
        config,
      );
      expect(after).toBe(closed);
    });
  });

  describe("materialChange", () => {
    const config = createConfig();
    const base = apply(config, { type: "probe.success", info: createInfo() });
    const withStatus = (changes: Partial<connection.Status>): connection.Status => ({
      ...base,
      ...changes,
    });
    const withDetails = (changes: Partial<connection.Details>): connection.Status => ({
      ...base,
      details: { ...base.details, ...changes },
    });

    it("should ignore the timestamp", () => {
      const next = withStatus({ time: TimeStamp.now().add(TimeSpan.hours(1)) });
      expect(materialChange(base, next)).toBe(false);
    });

    it("should ignore raw clock skew jitter", () => {
      const next = withDetails({ clockSkew: TimeSpan.milliseconds(7) });
      expect(materialChange(base, next)).toBe(false);
    });

    it("should not notify on repeated identical probes", () => {
      const again = reduce(base, { type: "probe.success", info: createInfo() }, config);
      expect(materialChange(base, again)).toBe(false);
    });

    it("should ignore error identity churn", () => {
      const a = withDetails({ error: new Unreachable({ message: "down" }) });
      const b = withDetails({ error: new Unreachable({ message: "down" }) });
      expect(materialChange(a, b)).toBe(false);
    });

    it("should ignore value-equal timestamps in retry", () => {
      const nextAt = TimeStamp.now();
      const a = withDetails({ retry: { attempt: 1, nextAt } });
      const b = withDetails({ retry: { attempt: 1, nextAt: new TimeStamp(nextAt) } });
      expect(materialChange(a, b)).toBe(false);
    });

    it("should notify on every other field", () => {
      const nextAt = TimeStamp.now();
      const flips: [string, connection.Status][] = [
        ["variant", withStatus({ variant: "warning" })],
        ["message", withStatus({ message: "other" })],
        ["description", withStatus({ description: "other" })],
        ["reason", withDetails({ reason: "auth" })],
        ["error", withDetails({ error: new Error("appeared") })],
        ["authenticated", withDetails({ authenticated: false })],
        ["streamLive", withDetails({ streamLive: true })],
        ["epoch", withDetails({ epoch: 7 })],
        ["clusterKey", withDetails({ clusterKey: "other-cluster" })],
        ["clientVersion", withDetails({ clientVersion: "9.9.9" })],
        ["nodeVersion", withDetails({ nodeVersion: "9.9.9" })],
        ["clientServerCompatible", withDetails({ clientServerCompatible: false })],
        ["clockSkewExceeded", withDetails({ clockSkewExceeded: true })],
        ["retry", withDetails({ retry: { attempt: 1, nextAt } })],
      ];
      flips.forEach(([field, next]) =>
        expect(materialChange(base, next), field).toBe(true),
      );
      const scheduled = withDetails({ retry: { attempt: 1, nextAt } });
      const bumped = withDetails({ retry: { attempt: 2, nextAt } });
      expect(materialChange(scheduled, bumped), "retry.attempt").toBe(true);
      const delayed = withDetails({
        retry: { attempt: 1, nextAt: nextAt.add(TimeSpan.seconds(5)) },
      });
      expect(materialChange(scheduled, delayed), "retry.nextAt").toBe(true);
    });

    // forces every new Details field to be classified as material or excluded
    it("should account for every details field", () => {
      const classified = [
        "reason",
        "error",
        "authenticated",
        "streamLive",
        "epoch",
        "clusterKey",
        "clientVersion",
        "nodeVersion",
        "clientServerCompatible",
        "clockSkew",
        "clockSkewExceeded",
        "retry",
        "probing",
      ];
      expect(Object.keys(base.details).sort()).toEqual([...classified].sort());
    });
  });

  describe("modeFor", () => {
    it("should map variants onto probe modes", () => {
      const config = createConfig();
      const initial = createInitialStatus(config);
      const withReason = (
        variant: connection.Status["variant"],
        reason?: connection.Reason,
      ): connection.Status => ({
        ...initial,
        variant,
        details: { ...initial.details, reason },
      });
      expect(modeFor(initial)).toEqual("probing");
      expect(modeFor(withReason("success"))).toEqual("heartbeat");
      expect(modeFor(withReason("warning"))).toEqual("probing");
      expect(modeFor(withReason("error", "unreachable"))).toEqual("probing");
      expect(modeFor(withReason("error", "auth"))).toEqual("idle");
      expect(modeFor({ ...initial, variant: "disabled" })).toEqual("idle");
    });
  });

  describe("Client", () => {
    it("should reach success against a live cluster", async () => {
      const client = createClient(liveUnary());
      const status = await client.connect(TimeSpan.seconds(5));
      expect(status.variant).toEqual("success");
      await client.close();
    });

    it("should escalate to error(unreachable) after the escalation budget", async () => {
      const client = createClient(failingUnary(), { escalateAfter: 2 });
      await expect(client.connect(TimeSpan.seconds(5))).rejects.toThrow();
      expect(client.status.variant).toEqual("error");
      expect(client.status.details.reason).toEqual("unreachable");
      await client.close();
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
      const client = createClient(unary, { escalateAfter: 1 });
      await waitForStatus(client, (s) => s.variant === "error");
      failing = false;
      const status = await waitForStatus(client, (s) => s.variant === "success");
      expect(status.details.reason).toBeUndefined();
      await client.close();
    });

    it("should park in error(unreachable) when a finite budget exhausts", async () => {
      const unary = failingUnary();
      const client = createClient(
        unary,
        { escalateAfter: 2 },
        { ...FAST_RETRY, maxRetries: 3 },
      );
      await waitForStatus(client, (s) => s.variant === "error");
      await waitForStatus(client, (s) => s.details.retry == null);
      const send = unary.send as ReturnType<typeof vi.fn>;
      const calls = send.mock.calls.length;
      await sleep(TimeSpan.milliseconds(20));
      expect(send.mock.calls.length).toBe(calls);
      client.retryNow();
      await sleep(TimeSpan.milliseconds(20));
      expect(send.mock.calls.length).toBeGreaterThan(calls);
      await client.close();
    });

    it("should probe immediately on retryNow while in heartbeat mode", async () => {
      const unary = mockUnary(() => TimeStamp.now());
      const client = createClient(unary, {}, FAST_RETRY, TimeSpan.seconds(60));
      await waitForStatus(client, (s) => s.variant === "success");
      const send = unary.send as ReturnType<typeof vi.fn>;
      // the mode switch into heartbeat may fire one buffered probe; settle
      // before measuring quiescence
      await sleep(TimeSpan.milliseconds(30));
      const calls = send.mock.calls.length;
      await sleep(TimeSpan.milliseconds(30));
      expect(send.mock.calls.length).toBe(calls);
      client.retryNow();
      await sleep(TimeSpan.milliseconds(30));
      expect(send.mock.calls.length).toBeGreaterThan(calls);
      await client.close();
    });

    it("should stop probing once the connection is closed", async () => {
      const unary = failingUnary();
      const client = createClient(unary, { escalateAfter: 1 });
      await waitForStatus(client, (s) => s.variant === "error");
      await client.close();
      expect(client.status.variant).toEqual("disabled");
      const send = unary.send as ReturnType<typeof vi.fn>;
      const calls = send.mock.calls.length;
      await sleep(TimeSpan.milliseconds(20));
      expect(send.mock.calls.length).toBe(calls);
    });

    it("should idle rather than probe while parked on an auth error", async () => {
      const unary = mockUnary(() => TimeStamp.now());
      const client = createClient(unary);
      await waitForStatus(client, (s) => s.variant === "success");
      client.notify({ type: "auth.failure", error: new AuthError("bad password") });
      const send = unary.send as ReturnType<typeof vi.fn>;
      const calls = send.mock.calls.length;
      await sleep(TimeSpan.milliseconds(60));
      expect(send.mock.calls.length).toBe(calls);
      await client.close();
    });

    it("should resume probing when credentials are replaced", async () => {
      const unary = mockUnary(() => TimeStamp.now());
      const client = createClient(unary);
      await waitForStatus(client, (s) => s.variant === "success");
      client.notify({ type: "auth.failure", error: new AuthError("bad password") });
      const send = unary.send as ReturnType<typeof vi.fn>;
      const calls = send.mock.calls.length;
      await sleep(TimeSpan.milliseconds(30));
      expect(send.mock.calls.length).toBe(calls);
      client.notify({ type: "credentials.replaced" });
      await waitForStatus(client, (s) => s.variant === "success");
      expect(send.mock.calls.length).toBeGreaterThan(calls);
      await client.close();
    });

    it("should discard an in-flight probe from before a mode change", async () => {
      let rejectProbe: ((error: Error) => void) | undefined;
      const unary: UnaryClient = {
        send: vi.fn().mockImplementation(
          async () =>
            await new Promise((_, reject) => {
              rejectProbe = reject;
            }),
        ),
        use: vi.fn(),
      };
      const client = createClient(
        unary,
        { requiresStream: true },
        FAST_RETRY,
        TimeSpan.seconds(60),
      );
      const send = unary.send as ReturnType<typeof vi.fn>;
      while (send.mock.calls.length === 0) await sleep(TimeSpan.milliseconds(1));
      client.notify({ type: "stream.live" });
      expect(client.status.variant).toEqual("success");
      // the probe issued before the stream came up fails late: it was aimed
      // at the old state and must not degrade the new one
      rejectProbe?.(new Unreachable({ message: "stale" }));
      await sleep(TimeSpan.milliseconds(20));
      expect(client.status.variant).toEqual("success");
      const closing = client.close();
      rejectProbe?.(new Unreachable({ message: "shutdown" }));
      await closing;
    });

    it("should reset then re-demand the stream on cluster replacement", async () => {
      const calls: string[] = [];
      let key = "first";
      const unary: UnaryClient = {
        send: vi.fn().mockImplementation(async () => ({
          clusterKey: key,
          nodeVersion: __VERSION__,
          nodeTime: TimeStamp.now(),
        })),
        use: vi.fn(),
      };
      const client = createClient(
        unary,
        {
          stream: {
            reset: async () => {
              calls.push("reset");
            },
            ensure: async () => {
              calls.push("ensure");
            },
          },
        },
        FAST_RETRY,
        TimeSpan.milliseconds(5),
      );
      await waitForStatus(client, (s) => s.details.clusterKey === "first");
      key = "second";
      await waitForStatus(client, (s) => s.details.clusterKey === "second");
      while (calls.length < 2) await sleep(TimeSpan.milliseconds(1));
      expect(calls.slice(0, 2)).toEqual(["reset", "ensure"]);
      await client.close();
    });

    it("should re-demand a dark stream on each probe", async () => {
      const ensured = vi.fn(async () => {});
      const unary = mockUnary(() => TimeStamp.now());
      const client = createClient(unary, {
        requiresStream: true,
        stream: { reset: async () => {}, ensure: ensured },
      });
      await waitForStatus(client, (s) => s.details.authenticated);
      while (ensured.mock.calls.length === 0) await sleep(TimeSpan.milliseconds(1));
      expect(client.status.variant).toEqual("loading");
      client.notify({ type: "stream.live" });
      expect(client.status.variant).toEqual("success");
      await client.close();
    });
  });

  describe("handle", () => {
    it("should notify on transitions and support unsubscribe", async () => {
      const client = createClient(mockUnary(() => TimeStamp.now()));
      const observed: connection.Status[] = [];
      const detach = client.onChange((status) => observed.push(status));
      await waitForStatus(client, (s) => s.variant === "success");
      expect(observed.some((s) => s.variant === "success")).toBe(true);
      const count = observed.length;
      detach();
      client.notify({ type: "stream.drop", error: new Error("dropped") });
      expect(observed.length).toBe(count);
      await client.close();
    });

    it("should reject connect with the stored auth error", async () => {
      const client = createClient(failingUnary());
      client.notify({ type: "auth.failure", error: new AuthError("bad password") });
      await expect(client.connect()).rejects.toThrow(AuthError);
      await client.close();
    });

    it("should time out connect against a dead cluster", async () => {
      const client = createClient(
        failingUnary(),
        { escalateAfter: Infinity },
        { ...FAST_RETRY, maxRetries: Infinity },
      );
      await expect(client.connect(TimeSpan.milliseconds(50))).rejects.toThrow();
      await client.close();
    });
  });
});
