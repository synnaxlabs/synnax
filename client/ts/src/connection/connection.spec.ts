// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Context, type UnaryClient, Unreachable } from "@synnaxlabs/freighter";
import { type breaker, observe, TimeSpan, TimeStamp, url } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { auth } from "@/auth";
import { connection } from "@/connection";
import { advance, createInitialStatus, type Event, reduce } from "@/connection/status";
import {
  AccessDeniedError,
  AuthError,
  DisconnectedError,
  ExpiredLicenseError,
  MissingLicenseError,
} from "@/index";
import { license } from "@/license";
import { query } from "@/query";
import { TEST_CLIENT_PARAMS, TEST_CREDENTIALS } from "@/testutil";
import { Transport } from "@/transport";

const liveUnary = (): UnaryClient => {
  const transport = new Transport(
    new url.URL({
      host: TEST_CLIENT_PARAMS.host,
      port: Number(TEST_CLIENT_PARAMS.port),
    }),
  );
  const client = new auth.Client(transport.unary, { credentials: TEST_CREDENTIALS });
  transport.use(client.middleware());
  return transport.unary;
};

const mockUnary = (nodeTime: () => TimeStamp): UnaryClient => ({
  send: vi.fn().mockImplementation(async () => ({
    clusterKey: "test-cluster",
    nodeVersion: __VERSION__,
    nodeTime: nodeTime(),
    license: "ok",
  })),
  use: vi.fn(),
});

const failingUnary = (): UnaryClient => ({
  send: vi.fn().mockRejectedValue(new Unreachable({ message: "server down" })),
  use: vi.fn(),
});

interface ScriptedUnary extends UnaryClient {
  /** Marks the cluster unreachable (or reachable again) for later checks. */
  setFailing: (failing: boolean) => void;
  /** Changes the cluster key answered by later checks. */
  setClusterKey: (key: string) => void;
  /** Changes the license state answered by later checks. */
  setLicense: (state: license.State) => void;
}

/** A unary whose reachability and cluster identity change mid-test. */
const createScriptedUnary = ({
  clusterKey = "test-cluster",
  failing = false,
  license = "ok",
}: {
  clusterKey?: string;
  failing?: boolean;
  license?: license.State;
} = {}): ScriptedUnary => {
  let key = clusterKey;
  let down = failing;
  let state = license;
  return {
    send: vi.fn().mockImplementation(async () => {
      if (down) throw new Unreachable({ message: "server down" });
      return {
        clusterKey: key,
        nodeVersion: __VERSION__,
        nodeTime: TimeStamp.now(),
        license: state,
      };
    }),
    use: vi.fn(),
    setFailing: (next) => (down = next),
    setClusterKey: (next) => (key = next),
    setLicense: (next) => (state = next),
  };
};

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
  nodeKey: 1,
  clockSkew: TimeSpan.ZERO,
  roundTrip: TimeSpan.ZERO,
  license: "ok",
  ...overrides,
});

/** Folds events over an initial status, mirroring the client's dispatch. */
const apply = (config: connection.Config, ...events: Event[]): connection.Status =>
  events.reduce(
    (status, event) => reduce(status, event, config),
    createInitialStatus(config),
  );

/** Reads the error reason, or undefined when the status is not an error. */
const reasonOf = (status: connection.Status): connection.Reason | undefined =>
  status.variant === "error" ? status.details.reason : undefined;

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

const sleep = async (span: TimeSpan): Promise<void> =>
  await new Promise((resolve) => setTimeout(resolve, span.milliseconds));

describe("connection", () => {
  describe("version compatibility", () => {
    // A Core built without an injected version reports 0.0.0, which pairs with
    // anything, so the mismatch cases carry their own node version.
    it("should adjust status if the server is too old", () => {
      const config = createConfig({ clientVersion: "50000.0.0" });
      const info = createInfo({ nodeVersion: "1.0.0" });
      const status = apply(config, { type: "check.success", info });
      expect(status.details.clientServerCompatible).toBe(false);
      expect(status.details.clientVersion).toBe("50000.0.0");
    });

    it("should adjust status if the server is too new", () => {
      const config = createConfig({ clientVersion: "0.1.0" });
      const info = createInfo({ nodeVersion: "1.0.0" });
      const status = apply(config, { type: "check.success", info });
      expect(status.details.clientServerCompatible).toBe(false);
    });

    it("should treat a 0.0 build on either side as compatible", () => {
      const config = createConfig({ clientVersion: "0.0.0" });
      const info = createInfo({ nodeVersion: "50000.0.0" });
      let status = apply(config, { type: "check.success", info });
      expect(status.details.clientServerCompatible).toBe(true);
      status = apply(createConfig({ clientVersion: "50000.0.0" }), {
        type: "check.success",
        info: { ...info, nodeVersion: "0.0.0-abc1234" },
      });
      expect(status.details.clientServerCompatible).toBe(true);
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
      expect(z.validate(z.uuid(), status.details.clusterKey)).toBe(true);
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
      });
      expect(status.variant).toEqual("error");
      expect(reasonOf(status)).toEqual("unreachable");
    });

    it("should return an error status against a dead port", async () => {
      const status = await connection.check({
        host: TEST_CLIENT_PARAMS.host,
        port: 9999,
      });
      expect(status.variant).toEqual("error");
    });
  });

  describe("clock skew", () => {
    it("should detect and warn about clock skew exceeding threshold", async () => {
      const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
      const unary = mockUnary(() => TimeStamp.now().add(TimeSpan.hours(1)));
      const client = createClient(unary, { clockSkewThreshold: TimeSpan.seconds(1) });
      const status = await client.connect(TimeSpan.seconds(5));
      expect(status.details.clockSkewExceeded).toBe(true);
      expect(status.details.clockSkew.valueOf()).not.toBe(0n);
      await client.close();
      expect(warn).toHaveBeenCalledWith(
        expect.stringContaining("excessive clock skew"),
      );
      warn.mockRestore();
    });

    it("should not flag skew within threshold", async () => {
      const unary = mockUnary(() => TimeStamp.now());
      const client = createClient(unary, { clockSkewThreshold: TimeSpan.seconds(1) });
      const status = await client.connect(TimeSpan.seconds(5));
      expect(status.details.clockSkewExceeded).toBe(false);
      await client.close();
    });

    it("should not flag skew within half the round trip", () => {
      const config = createConfig({ clockSkewThreshold: TimeSpan.seconds(1) });
      const status = apply(config, {
        type: "check.success",
        info: createInfo({
          clockSkew: TimeSpan.milliseconds(-1500),
          roundTrip: TimeSpan.seconds(2),
        }),
      });
      expect(status.details.clockSkewExceeded).toBe(false);
    });

    it("should flag skew beyond half the round trip", () => {
      const config = createConfig({ clockSkewThreshold: TimeSpan.seconds(1) });
      const status = apply(config, {
        type: "check.success",
        info: createInfo({
          clockSkew: TimeSpan.milliseconds(-2500),
          roundTrip: TimeSpan.seconds(2),
        }),
      });
      expect(status.details.clockSkewExceeded).toBe(true);
    });
  });

  describe("reduce", () => {
    it("should start in loading", () => {
      const initial = createInitialStatus(createConfig());
      expect(initial.variant).toEqual("loading");
      expect(initial.message).toEqual("Connecting");
    });

    it("should track an in-flight check without moving the variant", () => {
      const config = createConfig({ escalateAfter: 1 });
      const error = new Unreachable({ message: "server down" });
      const parked = apply(config, { type: "check.failure", error, attempt: 1 });
      expect(parked.variant).toEqual("error");
      const checking = reduce(parked, { type: "check.started" }, config);
      expect(checking.variant).toEqual("error");
      expect(checking.details.checking).toBe(true);
      const failed = reduce(
        checking,
        { type: "check.failure", error, attempt: 2 },
        config,
      );
      expect(failed.details.checking).toBe(false);
      const succeeded = reduce(
        checking,
        { type: "check.success", info: createInfo() },
        config,
      );
      expect(succeeded.details.checking).toBe(false);
      expect(succeeded.variant).toEqual("success");
    });

    it("should reach success on a check when no stream is required", () => {
      const status = apply(createConfig({ name: "Lab Core" }), {
        type: "check.success",
        info: createInfo(),
      });
      expect(status.variant).toEqual("success");
      expect(status.name).toEqual("Lab Core");
      expect(status.message).toEqual("Connected");
      expect(status.details.clusterKey).toEqual("test-cluster");
    });

    it("should stay loading until the stream comes up when one is required", () => {
      const config = createConfig({ requiresStream: true });
      const checked = apply(config, { type: "check.success", info: createInfo() });
      expect(checked.variant).toEqual("loading");
      expect(checked.details.authenticated).toBe(true);
      const live = reduce(checked, { type: "stream.live" }, config);
      expect(live.variant).toEqual("success");
      expect(live.details.streamLive).toBe(true);
    });

    it("should lift error(unreachable) to reconnecting on check success with a dark stream", () => {
      const config = createConfig({ requiresStream: true, escalateAfter: 1 });
      const error = new Unreachable({ message: "server down" });
      const parked = apply(config, { type: "check.failure", error, attempt: 1 });
      expect(parked.variant).toEqual("error");
      // reachable again: the error must lift, or its short circuit starves
      // the stream reopen that success waits on
      const checked = reduce(
        parked,
        { type: "check.success", info: createInfo() },
        config,
      );
      expect(checked.variant).toEqual("loading");
      expect(checked.message).toEqual("Reconnecting");
      expect(reasonOf(checked)).toBeUndefined();
      const live = reduce(checked, { type: "stream.live" }, config);
      expect(live.variant).toEqual("success");
    });

    it("should escalate to error(unreachable) after the escalation budget", () => {
      const config = createConfig({ escalateAfter: 2 });
      const error = new Unreachable({ message: "server down" });
      const first = apply(config, { type: "check.failure", error, attempt: 1 });
      expect(first.variant).toEqual("loading");
      const second = reduce(
        first,
        { type: "check.failure", error, attempt: 2 },
        config,
      );
      expect(second.variant).toEqual("error");
      expect(reasonOf(second)).toEqual("unreachable");
    });

    it("should escalate when a finite retry budget exhausts", () => {
      const config = createConfig({ escalateAfter: 100 });
      const error = new Unreachable({ message: "server down" });
      const failed = apply(config, { type: "check.failure", error, attempt: 1 });
      const exhausted = reduce(failed, { type: "retry.exhausted" }, config);
      expect(exhausted.variant).toEqual("error");
      expect(exhausted.message).toEqual("Cannot reach Core");
      expect(exhausted.description).toEqual("server down");
      expect(reasonOf(exhausted)).toEqual("unreachable");
      expect(exhausted.details.retry).toBeNull();
    });

    it("should transition success -> reconnecting on stream drop and back on reopen", () => {
      const config = createConfig({ requiresStream: true });
      const connected = apply(
        config,
        { type: "check.success", info: createInfo() },
        { type: "stream.live" },
      );
      expect(connected.variant).toEqual("success");
      const dropped = reduce(
        connected,
        { type: "stream.drop", error: new Error("socket died") },
        config,
      );
      expect(dropped.variant).toEqual("loading");
      expect(dropped.message).toEqual("Reconnecting");
      expect(dropped.details.streamLive).toBe(false);
      const reopened = reduce(dropped, { type: "stream.live" }, config);
      expect(reopened.variant).toEqual("success");
    });

    it("should settle on warning when the cluster denies the stream", () => {
      const config = createConfig({ requiresStream: true });
      const denied = apply(
        config,
        { type: "check.success", info: createInfo() },
        { type: "stream.denied" },
      );
      expect(denied.variant).toEqual("warning");
      expect(denied.message).toEqual("Connected");
      expect(denied.description).toEqual(
        "Live updates are unavailable. This user cannot read the change channels.",
      );
      expect(denied.details.streamDenied).toBe(true);
      expect(denied.details.streamLive).toBe(false);
      // later checks must not drag the settled denial back into loading
      const checked = reduce(
        denied,
        { type: "check.success", info: createInfo() },
        config,
      );
      expect(checked.variant).toEqual("warning");
      expect(checked.details.streamDenied).toBe(true);
    });

    it("should clear a denial when the stream comes up or credentials change", () => {
      const config = createConfig({ requiresStream: true });
      const denied = apply(
        config,
        { type: "check.success", info: createInfo() },
        { type: "stream.denied" },
      );
      const live = reduce(denied, { type: "stream.live" }, config);
      expect(live.variant).toEqual("success");
      expect(live.details.streamDenied).toBe(false);
      expect(live.description).toEqual("");
      const replaced = reduce(denied, { type: "credentials.replaced" }, config);
      expect(replaced.variant).toEqual("loading");
      expect(replaced.details.streamDenied).toBe(false);
    });

    it("should hold connecting on a live stream until the new credentials log in", () => {
      const config = createConfig({ requiresStream: true });
      const replaced = apply(
        config,
        { type: "check.success", info: createInfo() },
        { type: "credentials.replaced" },
      );
      expect(replaced.details.authenticated).toBe(false);
      const live = reduce(replaced, { type: "stream.live" }, config);
      expect(live.variant).toEqual("loading");
      expect(live.details.streamLive).toBe(true);
      const checked = reduce(
        live,
        { type: "check.success", info: createInfo() },
        config,
      );
      expect(checked.variant).toEqual("success");
    });

    it("should degrade to reconnecting when a heartbeat fails while connected", () => {
      const config = createConfig();
      const connected = apply(config, { type: "check.success", info: createInfo() });
      const degraded = reduce(
        connected,
        {
          type: "check.failure",
          error: new Unreachable({ message: "gone" }),
          attempt: 1,
        },
        config,
      );
      expect(degraded.variant).toEqual("loading");
      expect(degraded.message).toEqual("Reconnecting");
    });

    it("should enter error(auth) on auth failure and recover on new credentials", () => {
      const config = createConfig();
      const rejected = apply(config, {
        type: "auth.failure",
        error: new AuthError("invalid credentials"),
      });
      expect(rejected.variant).toEqual("error");
      expect(rejected.message).toEqual("Failed to authenticate");
      expect(rejected.description).toEqual("invalid credentials");
      expect(reasonOf(rejected)).toEqual("auth");
      expect(rejected.details.authenticated).toBe(false);
      const retrying = reduce(rejected, { type: "retry.requested" }, config);
      expect(retrying.variant).toEqual("error");
      expect(reasonOf(retrying)).toEqual("auth");
      const replaced = reduce(rejected, { type: "credentials.replaced" }, config);
      expect(replaced.variant).toEqual("loading");
      expect(reasonOf(replaced)).toBeUndefined();
    });

    it("should store the session key on auth success", () => {
      const status = apply(createConfig(), { type: "auth.success", session: "a" });
      expect(status.details.authenticated).toBe(true);
      expect(status.details.session).toEqual("a");
    });

    it("should clear the session key on auth failure", () => {
      const status = apply(
        createConfig(),
        { type: "auth.success", session: "a" },
        { type: "auth.failure", error: new AuthError("session ended") },
      );
      expect(status.details.session).toBeUndefined();
    });

    it("should clear the session key when credentials are replaced", () => {
      const status = apply(
        createConfig(),
        { type: "auth.success", session: "a" },
        { type: "credentials.replaced" },
      );
      expect(status.details.session).toBeUndefined();
    });

    it("should treat an auth error from the check as an auth failure", () => {
      const status = apply(createConfig(), {
        type: "check.failure",
        error: new AuthError("bad password"),
        attempt: 1,
      });
      expect(reasonOf(status)).toEqual("auth");
    });

    it("should enter error(unlicensed) when the Core reports no license", () => {
      const config = createConfig();
      const status = apply(config, {
        type: "check.success",
        info: createInfo({ license: "missing" }),
      });
      expect(reasonOf(status)).toEqual("unlicensed");
      expect(status.details.license).toEqual("missing");
      expect(status.details.authenticated).toBe(true);
      expect(MissingLicenseError.matches(status.details.error)).toBe(true);
    });

    it("should report an expired license with its own error", () => {
      const status = apply(createConfig(), {
        type: "check.success",
        info: createInfo({ license: "expired" }),
      });
      expect(reasonOf(status)).toEqual("unlicensed");
      expect(ExpiredLicenseError.matches(status.details.error)).toBe(true);
    });

    it("should leave error(unlicensed) once a check reports a license", () => {
      const config = createConfig();
      const unlicensed = apply(config, {
        type: "check.success",
        info: createInfo({ license: "missing" }),
      });
      const licensed = reduce(
        unlicensed,
        { type: "check.success", info: createInfo() },
        config,
      );
      expect(licensed.variant).toEqual("success");
      expect(licensed.details.license).toEqual("ok");
      expect(licensed.details.error).toBeUndefined();
    });

    it("should lift error(unlicensed) to reconnecting with a dark stream", () => {
      const config = createConfig({ requiresStream: true });
      const unlicensed = apply(config, {
        type: "check.success",
        info: createInfo({ license: "missing" }),
      });
      const checked = reduce(
        unlicensed,
        { type: "check.success", info: createInfo() },
        config,
      );
      expect(checked.variant).toEqual("loading");
      expect(reasonOf(checked)).toBeUndefined();
    });

    it("should escalate an unlicensed Core that stops answering to unreachable", () => {
      const config = createConfig({ escalateAfter: 2 });
      const unlicensed = apply(config, {
        type: "check.success",
        info: createInfo({ license: "missing" }),
      });
      const error = new Unreachable({ message: "server down" });
      const once = reduce(
        unlicensed,
        { type: "check.failure", error, attempt: 1 },
        config,
      );
      expect(once.variant).toEqual("loading");
      const twice = reduce(once, { type: "check.failure", error, attempt: 2 }, config);
      expect(reasonOf(twice)).toEqual("unreachable");
      expect(twice.message).toEqual("Cannot reach Core");
      expect(twice.description).toEqual("server down");
      expect(twice.details.error).toBe(error);
    });

    it("should clear error(unlicensed) on retry.requested", () => {
      const config = createConfig();
      const unlicensed = apply(config, {
        type: "check.success",
        info: createInfo({ license: "missing" }),
      });
      const retrying = reduce(unlicensed, { type: "retry.requested" }, config);
      expect(retrying.variant).toEqual("loading");
    });

    it("should clear error(unreachable) on retry.requested", () => {
      const config = createConfig({ escalateAfter: 1 });
      const failed = apply(config, {
        type: "check.failure",
        error: new Unreachable({ message: "server down" }),
        attempt: 1,
      });
      expect(reasonOf(failed)).toEqual("unreachable");
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

    it("should return to first contact when a check answers with a new cluster", () => {
      const config = createConfig({ requiresStream: true });
      const connected = apply(
        config,
        { type: "check.success", info: createInfo() },
        { type: "stream.live" },
        { type: "epoch.advanced", epoch: 1 },
      );
      expect(connected.variant).toEqual("success");
      const replaced = reduce(
        connected,
        {
          type: "check.success",
          info: createInfo({ clusterKey: "other-cluster" }),
        },
        config,
      );
      expect(replaced.variant).toEqual("loading");
      expect(replaced.message).toEqual("Connecting");
      expect(replaced.details.clusterKey).toEqual("other-cluster");
      expect(replaced.details.epoch).toBe(0);
      expect(replaced.details.streamLive).toBe(false);
      expect(replaced.details.retry).toBeNull();
    });

    it("should clear a parked error on cluster replacement", () => {
      const config = createConfig();
      const parked = apply(
        config,
        { type: "check.success", info: createInfo() },
        { type: "check.failure", error: new Error("gone"), attempt: 10 },
        { type: "retry.exhausted" },
      );
      expect(parked.variant).toEqual("error");
      const replaced = reduce(
        parked,
        {
          type: "check.success",
          info: createInfo({ clusterKey: "other-cluster" }),
        },
        config,
      );
      expect(replaced.variant).toEqual("loading");
      expect(reasonOf(replaced)).toBeUndefined();
      expect(replaced.details.error).toBeUndefined();
    });

    it("should become disabled on close and stay there", () => {
      const config = createConfig();
      const closed = apply(config, { type: "closed" });
      expect(closed.variant).toEqual("disabled");
      const after = reduce(
        closed,
        { type: "check.success", info: createInfo() },
        config,
      );
      expect(after).toBe(closed);
    });
  });

  describe("advance", () => {
    const config = createConfig();
    const stale = (status: connection.Status): connection.Status => ({
      ...status,
      time: TimeStamp.ZERO,
    });
    const base = stale(apply(config, { type: "check.success", info: createInfo() }));
    const authenticated = stale(
      apply(
        config,
        { type: "check.success", info: createInfo() },
        { type: "auth.success", session: "a" },
      ),
    );
    const nextAt = TimeStamp.now();
    const scheduled = stale(
      apply(config, { type: "retry.scheduled", attempt: 1, nextAt }),
    );

    const expectQuiet = (prev: connection.Status, event: Event): connection.Status => {
      const [next, changed] = advance(prev, event, config);
      expect(changed).toBe(false);
      expect(next.time).toBe(prev.time);
      return next;
    };

    it("should not notify on repeated identical checks", () => {
      expectQuiet(base, { type: "check.success", info: createInfo() });
    });

    it("should not notify on a repeat of the same session", () => {
      expectQuiet(authenticated, { type: "auth.success", session: "a" });
    });

    it("should ignore raw clock skew jitter", () => {
      const info = createInfo({ clockSkew: TimeSpan.milliseconds(7) });
      const next = expectQuiet(base, { type: "check.success", info });
      expect(next.details.clockSkew).toEqual(TimeSpan.milliseconds(7));
    });

    it("should ignore error identity churn", () => {
      const failure = (attempt: number): Event => ({
        type: "check.failure",
        error: new Unreachable({ message: "down" }),
        attempt,
      });
      expectQuiet(stale(apply(config, failure(1))), failure(2));
    });

    it("should ignore value-equal timestamps in retry", () => {
      expectQuiet(scheduled, {
        type: "retry.scheduled",
        attempt: 1,
        nextAt: new TimeStamp(nextAt),
      });
    });

    it("should notify and stamp the time on every material change", () => {
      const unreachable = stale(
        apply(
          config,
          { type: "check.failure", error: new Error("down"), attempt: 1 },
          { type: "retry.exhausted" },
        ),
      );
      const changes: [string, connection.Status, Event][] = [
        ["variant", base, { type: "stream.drop" }],
        ["checking", base, { type: "check.started" }],
        ["streamLive", base, { type: "stream.live" }],
        ["streamDenied", base, { type: "stream.denied" }],
        ["authenticated", base, { type: "credentials.replaced" }],
        ["session", authenticated, { type: "auth.success", session: "b" }],
        ["epoch", base, { type: "epoch.advanced", epoch: 7 }],
        [
          "clusterKey",
          base,
          { type: "check.success", info: createInfo({ clusterKey: "other" }) },
        ],
        [
          "nodeVersion",
          base,
          { type: "check.success", info: createInfo({ nodeVersion: "9.9.9" }) },
        ],
        [
          "clockSkewExceeded",
          base,
          {
            type: "check.success",
            info: createInfo({ clockSkew: TimeSpan.hours(1) }),
          },
        ],
        [
          "license",
          base,
          { type: "check.success", info: createInfo({ license: "missing" }) },
        ],
        [
          "error",
          stale(apply(config)),
          { type: "check.failure", error: new Error("appeared"), attempt: 1 },
        ],
        [
          "reason",
          unreachable,
          { type: "auth.failure", error: new AuthError("bad password") },
        ],
        ["nodeKey", base, { type: "check.success", info: createInfo({ nodeKey: 2 }) }],
        ["retry", base, { type: "retry.scheduled", attempt: 1, nextAt }],
        ["retry.attempt", scheduled, { type: "retry.scheduled", attempt: 2, nextAt }],
        [
          "retry.nextAt",
          scheduled,
          {
            type: "retry.scheduled",
            attempt: 1,
            nextAt: nextAt.add(TimeSpan.seconds(5)),
          },
        ],
      ];
      changes.forEach(([field, prev, event]) => {
        const [next, changed] = advance(prev, event, config);
        expect(changed, field).toBe(true);
        expect(next.time.after(prev.time), field).toBe(true);
      });
    });

    // forces every new Details field to be classified as material or excluded
    it("should account for every details field", () => {
      const classified = [
        "error",
        "authenticated",
        "session",
        "streamLive",
        "streamDenied",
        "epoch",
        "clusterKey",
        "clientVersion",
        "nodeVersion",
        "nodeKey",
        "clientServerCompatible",
        "clockSkew",
        "clockSkewExceeded",
        "retry",
        "checking",
        "license",
      ];
      const authError = reduce(
        authenticated,
        { type: "auth.failure", error: new AuthError("bad password") },
        config,
      );
      expect(Object.keys(authenticated.details).sort()).toEqual([...classified].sort());
      expect(Object.keys(authError.details).sort()).toEqual(
        [...classified, "reason"].sort(),
      );
    });
  });

  describe("Client", () => {
    it("should reach success against a live cluster", async () => {
      const client = createClient(liveUnary());
      const status = await client.connect(TimeSpan.seconds(5));
      expect(status.variant).toEqual("success");
      expect(status.details.authenticated).toBe(true);
      expect(z.validate(z.uuid(), status.details.clusterKey)).toBe(true);
      expect(status.details.nodeVersion).not.toEqual("");
      expect(status.details.nodeKey).toBe(1);
      expect(status.details.clientVersion).toBe(__VERSION__);
      expect(status.details.clientServerCompatible).toBe(true);
      await client.close();
    });

    it("should escalate to error(unreachable) after the escalation budget", async () => {
      const client = createClient(failingUnary(), { escalateAfter: 2 });
      await expect(client.connect(TimeSpan.seconds(5))).rejects.toThrow(Unreachable);
      expect(client.status.variant).toEqual("error");
      expect(reasonOf(client.status)).toEqual("unreachable");
      await client.close();
    });

    it("should keep checking after escalation and self-heal", async () => {
      const unary = createScriptedUnary({ failing: true });
      const client = createClient(unary, { escalateAfter: 1 });
      await observe.until(
        client,
        () => client.status,
        (s) => s.variant === "error",
      );
      unary.setFailing(false);
      const status = await observe.until(
        client,
        () => client.status,
        (s) => s.variant === "success",
      );
      expect(reasonOf(status)).toBeUndefined();
      await client.close();
    });

    it("should park in error(unreachable) when a finite budget exhausts", async () => {
      const unary = failingUnary();
      const client = createClient(
        unary,
        { escalateAfter: 2 },
        { ...FAST_RETRY, maxRetries: 3 },
      );
      await observe.until(
        client,
        () => client.status,
        (s) => s.variant === "error",
      );
      await observe.until(
        client,
        () => client.status,
        (s) => s.details.retry == null,
      );
      const send = unary.send as ReturnType<typeof vi.fn>;
      const calls = send.mock.calls.length;
      await sleep(TimeSpan.milliseconds(20));
      expect(send.mock.calls.length).toBe(calls);
      client.retryNow();
      await sleep(TimeSpan.milliseconds(20));
      expect(send.mock.calls.length).toBeGreaterThan(calls);
      await client.close();
    });

    it("should check immediately on retryNow while in heartbeat mode", async () => {
      const unary = mockUnary(() => TimeStamp.now());
      const client = createClient(unary, {}, FAST_RETRY, TimeSpan.seconds(60));
      await observe.until(
        client,
        () => client.status,
        (s) => s.variant === "success",
      );
      const send = unary.send as ReturnType<typeof vi.fn>;
      // the mode switch into heartbeat may fire one buffered check; settle
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

    it("should stop checking once the connection is closed", async () => {
      const unary = failingUnary();
      const client = createClient(unary, { escalateAfter: 1 });
      await observe.until(
        client,
        () => client.status,
        (s) => s.variant === "error",
      );
      await client.close();
      expect(client.status.variant).toEqual("disabled");
      const send = unary.send as ReturnType<typeof vi.fn>;
      const calls = send.mock.calls.length;
      await sleep(TimeSpan.milliseconds(20));
      expect(send.mock.calls.length).toBe(calls);
    });

    it("should idle rather than check while parked on an auth error", async () => {
      const unary = mockUnary(() => TimeStamp.now());
      const client = createClient(unary);
      await observe.until(
        client,
        () => client.status,
        (s) => s.variant === "success",
      );
      client.notify({ type: "auth.failure", error: new AuthError("bad password") });
      const send = unary.send as ReturnType<typeof vi.fn>;
      const calls = send.mock.calls.length;
      await sleep(TimeSpan.milliseconds(60));
      expect(send.mock.calls.length).toBe(calls);
      await client.close();
    });

    it("should resume checking when credentials are replaced", async () => {
      const unary = mockUnary(() => TimeStamp.now());
      const client = createClient(unary);
      await observe.until(
        client,
        () => client.status,
        (s) => s.variant === "success",
      );
      client.notify({ type: "auth.failure", error: new AuthError("bad password") });
      const send = unary.send as ReturnType<typeof vi.fn>;
      const calls = send.mock.calls.length;
      await sleep(TimeSpan.milliseconds(30));
      expect(send.mock.calls.length).toBe(calls);
      client.notify({ type: "credentials.replaced" });
      await observe.until(
        client,
        () => client.status,
        (s) => s.variant === "success",
      );
      expect(send.mock.calls.length).toBeGreaterThan(calls);
      await client.close();
    });

    it("should discard an in-flight check from before a mode change", async () => {
      let rejectCheck: ((error: Error) => void) | undefined;
      const unary: UnaryClient = {
        send: vi.fn().mockImplementation(
          async () =>
            await new Promise((_, reject) => {
              rejectCheck = reject;
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
      // the stream opens behind the auth middleware, so a login lands first
      client.notify({ type: "auth.success", session: "session" });
      client.notify({ type: "stream.live" });
      expect(client.status.variant).toEqual("success");
      // the check issued before the stream came up fails late: it was aimed
      // at the old state and must not degrade the new one
      rejectCheck?.(new Unreachable({ message: "stale" }));
      await sleep(TimeSpan.milliseconds(20));
      expect(client.status.variant).toEqual("success");
      const closing = client.close();
      rejectCheck?.(new Unreachable({ message: "shutdown" }));
      await closing;
    });

    it("should reset then re-demand the stream on cluster replacement", async () => {
      const calls: string[] = [];
      const unary = createScriptedUnary({ clusterKey: "first" });
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
      await observe.until(
        client,
        () => client.status,
        (s) => s.details.clusterKey === "first",
      );
      unary.setClusterKey("second");
      await observe.until(
        client,
        () => client.status,
        (s) => s.details.clusterKey === "second",
      );
      while (calls.length < 2) await sleep(TimeSpan.milliseconds(1));
      expect(calls.slice(0, 2)).toEqual(["reset", "ensure"]);
      await client.close();
    });

    it("should re-demand a dark stream on each check", async () => {
      const ensured = vi.fn(async () => {});
      const unary = mockUnary(() => TimeStamp.now());
      const client = createClient(unary, {
        requiresStream: true,
        stream: { reset: async () => {}, ensure: ensured },
      });
      await observe.until(
        client,
        () => client.status,
        (s) => s.details.authenticated,
      );
      while (ensured.mock.calls.length === 0) await sleep(TimeSpan.milliseconds(1));
      expect(client.status.variant).toEqual("loading");
      client.notify({ type: "stream.live" });
      expect(client.status.variant).toEqual("success");
      await client.close();
    });

    it("should not demand the stream until the Core is licensed", async () => {
      const ensured = vi.fn(async () => {});
      const unary = createScriptedUnary({ license: "missing" });
      const client = createClient(unary, {
        requiresStream: true,
        stream: { reset: async () => {}, ensure: ensured },
      });
      const send = unary.send as ReturnType<typeof vi.fn>;
      while (send.mock.calls.length < 3) await sleep(TimeSpan.milliseconds(1));
      expect(ensured).not.toHaveBeenCalled();
      unary.setLicense("ok");
      while (ensured.mock.calls.length === 0) await sleep(TimeSpan.milliseconds(1));
      await client.close();
    });

    it("should probe a denied stream on a slow cadence and report it once", async () => {
      const internal: Error[] = [];
      let checks = 0;
      let probes = 0;
      const unary = mockUnary(() => {
        checks += 1;
        return TimeStamp.now();
      });
      const client: connection.Client = createClient(unary, {
        requiresStream: true,
        deniedStreamProbeAfter: 3,
        stream: {
          reset: async () => {},
          ensure: async () => {
            probes += 1;
            client.notify({ type: "stream.denied" });
            throw new AccessDeniedError("no permission to stream");
          },
        },
        onInternalError: (error) => internal.push(error),
      });
      await observe.until(
        client,
        () => client.status,
        (s) => s.details.streamDenied,
      );
      expect(client.status.variant).toEqual("warning");
      expect(probes).toEqual(1);
      const checksAtDenial = checks;
      while (probes < 2) await sleep(TimeSpan.milliseconds(1));
      expect(checks - checksAtDenial).toBeGreaterThanOrEqual(3);
      expect(internal).toHaveLength(1);
      expect(AccessDeniedError.matches(internal[0])).toBe(true);
      await client.close();
    });

    it("should recover from escalated downtime without internal errors or a cache reset", async () => {
      const internal: Error[] = [];
      const streamCalls: string[] = [];
      const unary = createScriptedUnary({ failing: true });
      const client: connection.Client = createClient(unary, {
        escalateAfter: 2,
        requiresStream: true,
        stream: {
          reset: async () => {
            streamCalls.push("reset");
          },
          ensure: async () => {
            streamCalls.push("ensure");
            client.notify({ type: "stream.live" });
          },
        },
        onInternalError: (error) => internal.push(error),
      });
      await observe.until(
        client,
        () => client.status,
        (s) => s.variant === "error" && s.details.reason === "unreachable",
      );
      // parked beneath the error, retries keep climbing
      await observe.until(
        client,
        () => client.status,
        (s) => (s.details.retry?.attempt ?? 0) >= 3,
      );
      unary.setFailing(false);
      const status = await observe.until(
        client,
        () => client.status,
        (s) => s.variant === "success" && s.details.streamLive,
      );
      // same cluster returned: recovery must not take the replacement path
      expect(status.details.clusterKey).toEqual("test-cluster");
      expect(streamCalls).not.toContain("reset");
      expect(internal).toEqual([]);
      await client.close();
    });

    it("should surface a reset failure during cluster replacement", async () => {
      const internal: Error[] = [];
      const unary = createScriptedUnary({ clusterKey: "first" });
      const client = createClient(
        unary,
        {
          stream: {
            reset: async () => {
              throw new Error("cache reset failed");
            },
            ensure: async () => {},
          },
          onInternalError: (error) => internal.push(error),
        },
        FAST_RETRY,
        TimeSpan.milliseconds(5),
      );
      await observe.until(
        client,
        () => client.status,
        (s) => s.details.clusterKey === "first",
      );
      unary.setClusterKey("second");
      await observe.until(
        client,
        () => client.status,
        (s) => s.details.clusterKey === "second",
      );
      while (internal.length === 0) await sleep(TimeSpan.milliseconds(1));
      expect(internal[0].message).toEqual(
        "failed to reset cache after cluster replacement",
      );
      const cause = internal[0].cause;
      if (!(cause instanceof Error)) throw new Error("cause is not an Error");
      expect(cause.message).toEqual("cache reset failed");
      await client.close();
    });
  });

  describe("middleware", () => {
    const createUnaryContext = (target: string): Context => ({
      target,
      role: "client",
      protocol: "http",
      params: {},
    });

    it("should short-circuit unary calls while the cluster is unreachable", async () => {
      const unary = createScriptedUnary({ failing: true });
      const client = createClient(unary, { escalateAfter: 1 });
      await observe.until(
        client,
        () => client.status,
        (s) => s.variant === "error" && s.details.reason === "unreachable",
      );
      const next = vi.fn(async (ctx: Context) => ctx);
      await expect(
        client.middleware()(createUnaryContext("/channel/retrieve"), next),
      ).rejects.toThrow(DisconnectedError);
      expect(next).not.toHaveBeenCalled();
      await client.close();
    });

    it("should exempt the check and session endpoints so the connection can heal", async () => {
      const unary = createScriptedUnary({ failing: true });
      const client = createClient(unary, { escalateAfter: 1 });
      await observe.until(
        client,
        () => client.status,
        (s) => s.variant === "error",
      );
      const mw = client.middleware();
      const next = vi.fn(async (ctx: Context) => ctx);
      await mw(createUnaryContext("/api/v1/connectivity/check"), next);
      await mw(createUnaryContext("/api/v1/auth/login"), next);
      await mw(createUnaryContext("/api/v1/auth/session/create"), next);
      await mw(createUnaryContext("/api/v1/auth/session/renew"), next);
      await mw(createUnaryContext("/api/v1/auth/session/retrieve"), next);
      await mw(createUnaryContext("/api/v1/auth/session/delete"), next);
      expect(next).toHaveBeenCalledTimes(6);
      await client.close();
    });

    it("should let session upkeep through while the Core is unlicensed", async () => {
      const client = createClient(createScriptedUnary({ license: "expired" }));
      await observe.until(
        client,
        () => client.status,
        (s) => s.variant === "error" && s.details.reason === "unlicensed",
      );
      const mw = client.middleware();
      const next = vi.fn(async (ctx: Context) => ctx);
      await mw(createUnaryContext("/api/v1/auth/session/renew"), next);
      await mw(createUnaryContext("/api/v1/auth/session/retrieve"), next);
      await expect(mw(createUnaryContext("/channel/retrieve"), next)).rejects.toThrow(
        ExpiredLicenseError,
      );
      expect(next).toHaveBeenCalledTimes(2);
      await client.close();
    });

    it("should not demand the stream from an unlicensed replacement", async () => {
      const calls: string[] = [];
      const unary = createScriptedUnary({ clusterKey: "first" });
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
      await observe.until(
        client,
        () => client.status,
        (s) => s.details.clusterKey === "first",
      );
      unary.setLicense("missing");
      unary.setClusterKey("second");
      await observe.until(
        client,
        () => client.status,
        (s) => s.details.clusterKey === "second",
      );
      while (calls.length === 0) await sleep(TimeSpan.milliseconds(1));
      const send = unary.send as ReturnType<typeof vi.fn>;
      const sent = send.mock.calls.length;
      while (send.mock.calls.length < sent + 3) await sleep(TimeSpan.milliseconds(1));
      expect(calls).toEqual(["reset"]);
      await client.close();
    });

    it("should short-circuit unary calls with the expired license error", async () => {
      const client = createClient(createScriptedUnary({ license: "expired" }));
      await observe.until(
        client,
        () => client.status,
        (s) => s.variant === "error" && s.details.reason === "unlicensed",
      );
      const next = vi.fn(async (ctx: Context) => ctx);
      await expect(
        client.middleware()(createUnaryContext("/channel/retrieve"), next),
      ).rejects.toThrow(ExpiredLicenseError);
      expect(next).not.toHaveBeenCalled();
      await client.close();
    });

    it("should check again once a license is activated", async () => {
      const client = createClient(
        createScriptedUnary({ license: "missing" }),
        {},
        FAST_RETRY,
        TimeSpan.seconds(60),
      );
      await observe.until(
        client,
        () => client.status,
        (s) => s.variant === "error",
      );
      const retryNow = vi.spyOn(client, "retryNow");
      const licenses = new license.Client({
        unary: {
          send: vi
            .fn()
            .mockResolvedValue({ state: "ok", warning: "", fingerprint: [] }),
          use: vi.fn(),
        },
        connection: client,
        cache: new query.Cache({ openStreamer: null }),
      });
      await licenses.activate("key");
      expect(retryNow).toHaveBeenCalledOnce();
      await client.close();
    });

    it("should short-circuit unary calls with the license error while unlicensed", async () => {
      const client = createClient(createScriptedUnary({ license: "missing" }));
      await observe.until(
        client,
        () => client.status,
        (s) => s.variant === "error" && s.details.reason === "unlicensed",
      );
      const next = vi.fn(async (ctx: Context) => ctx);
      await expect(
        client.middleware()(createUnaryContext("/channel/retrieve"), next),
      ).rejects.toThrow(MissingLicenseError);
      expect(next).not.toHaveBeenCalled();
      await client.close();
    });

    it("should exempt the license endpoints so a license can be activated", async () => {
      const client = createClient(createScriptedUnary({ license: "missing" }));
      await observe.until(
        client,
        () => client.status,
        (s) => s.variant === "error",
      );
      const mw = client.middleware();
      const next = vi.fn(async (ctx: Context) => ctx);
      await mw(createUnaryContext(`/api/v1${license.RETRIEVE_ENDPOINT}`), next);
      await mw(createUnaryContext(`/api/v1${license.ACTIVATE_ENDPOINT}`), next);
      await mw(createUnaryContext(`/api/v1${license.DEACTIVATE_ENDPOINT}`), next);
      expect(next).toHaveBeenCalledTimes(3);
      await client.close();
    });

    it("should pass calls through while the cluster is reachable", async () => {
      const client = createClient(createScriptedUnary());
      await observe.until(
        client,
        () => client.status,
        (s) => s.variant === "success",
      );
      const next = vi.fn(async (ctx: Context) => ctx);
      await client.middleware()(createUnaryContext("/channel/retrieve"), next);
      expect(next).toHaveBeenCalledTimes(1);
      await client.close();
    });
  });

  describe("handle", () => {
    it("should notify on transitions and support unsubscribe", async () => {
      const client = createClient(mockUnary(() => TimeStamp.now()));
      const observed: connection.Status[] = [];
      const detach = client.onChange((status) => observed.push(status));
      await observe.until(
        client,
        () => client.status,
        (s) => s.variant === "success",
      );
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
