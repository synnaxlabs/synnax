// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type connection, createTestClient } from "@synnaxlabs/client";
import { breaker, TimeSpan } from "@synnaxlabs/x";
import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Node } from "@/feature/node";
import { Session } from "@/session";
import { type State } from "@/session/store";
import { createConnectedConsoleWrapper } from "@/testutil";

const connState = (
  status: connection.Status,
  clusterKey: string,
  message?: string,
): connection.State => ({
  status,
  clusterKey,
  message,
  clientVersion: "0.0.0",
  clientServerCompatible: true,
  clockSkew: TimeSpan.ZERO,
  clockSkewExceeded: false,
});

const createState = (clusterKeys: string[], selected: string | null): State => ({
  ...Session.ZERO_STATE,
  [Session.Node.SLICE_NAME]: {
    ...Session.Node.ZERO_SLICE_STATE,
    selected: selected ?? undefined,
    clusters: Object.fromEntries(
      clusterKeys.map((k) => [
        k,
        {
          key: k,
          name: k,
          host: "localhost",
          port: 9090,
          username: "synnax",
          password: "seldon",
          secure: false,
        },
      ]),
    ),
  },
});

// sequence returns each snapshot once and then repeats the last one indefinitely, so a
// poll loop that overshoots the scripted transitions keeps seeing the terminal state.
const sequence = (...snapshots: Node.Snapshot[]) => {
  let i = 0;
  return () => snapshots[Math.min(i++, snapshots.length - 1)];
};

const instantPoll = (maxRetries = 1_000) =>
  new breaker.Breaker({
    baseInterval: TimeSpan.milliseconds(1),
    scale: 1,
    maxRetries,
    sleepFn: async () => {},
  });

describe("connectToCluster", () => {
  it("should throw if the cluster is unknown", async () => {
    const setActive = vi.fn();
    await expect(
      Node.connectToCluster("missing", {
        getState: () => createState([], null),
        getSnapshot: sequence({
          client: null,
          connState: connState("disconnected", ""),
        }),
        setActive,
        poll: instantPoll(),
      }),
    ).rejects.toThrow("Core with key missing not found");
    expect(setActive).not.toHaveBeenCalled();
  });

  it("should return the managed client when already active and connected", async () => {
    const setActive = vi.fn();
    const active = createTestClient();
    const result = await Node.connectToCluster("a", {
      getState: () => createState(["a"], "a"),
      getSnapshot: sequence({ client: active, connState: connState("connected", "a") }),
      setActive,
      poll: instantPoll(),
    });
    expect(result).toBe(active);
    expect(setActive).not.toHaveBeenCalled();
  });

  it("should switch clusters and resolve once the new client connects", async () => {
    const setActive = vi.fn();
    const prior = createTestClient();
    const next = createTestClient();
    const result = await Node.connectToCluster("b", {
      getState: () => createState(["a", "b"], "a"),
      getSnapshot: sequence(
        { client: prior, connState: connState("connected", "a") },
        { client: prior, connState: connState("connecting", "a") },
        { client: next, connState: connState("connecting", "b") },
        { client: next, connState: connState("connected", "b") },
      ),
      setActive,
      poll: instantPoll(),
    });
    expect(result).toBe(next);
    expect(setActive).toHaveBeenCalledWith("b");
  });

  it("should ignore a stale failure from the previous cluster while switching", async () => {
    const setActive = vi.fn();
    const prior = createTestClient();
    const next = createTestClient();
    const result = await Node.connectToCluster("b", {
      getState: () => createState(["a", "b"], "a"),
      getSnapshot: sequence(
        { client: prior, connState: connState("failed", "a", "stale failure") },
        { client: prior, connState: connState("failed", "a", "stale failure") },
        { client: next, connState: connState("connecting", "b") },
        { client: next, connState: connState("connected", "b") },
      ),
      setActive,
      poll: instantPoll(),
    });
    expect(result).toBe(next);
  });

  it("should throw when the target connection fails", async () => {
    const active = createTestClient();
    await expect(
      Node.connectToCluster("a", {
        getState: () => createState(["a"], "a"),
        getSnapshot: sequence(
          { client: active, connState: connState("connecting", "a") },
          { client: active, connState: connState("failed", "a", "auth rejected") },
        ),
        setActive: vi.fn(),
        poll: instantPoll(),
      }),
    ).rejects.toThrow("auth rejected");
  });

  it("should throw when the connection times out", async () => {
    const active = createTestClient();
    await expect(
      Node.connectToCluster("a", {
        getState: () => createState(["a"], "a"),
        getSnapshot: sequence({
          client: active,
          connState: connState("connecting", "a"),
        }),
        setActive: vi.fn(),
        poll: instantPoll(3),
      }),
    ).rejects.toThrow("Timed out connecting to cluster a");
  });
});

describe("useLink", () => {
  it("should resolve the active cluster's managed client", async () => {
    const c = createTestClient();
    const { clusterKey } = await c.connectivity.check();
    const { wrapper } = await createConnectedConsoleWrapper({
      client: null,
      connParams: {
        host: "localhost",
        port: 9090,
        username: "synnax",
        password: "seldon",
        secure: false,
      },
      preloadedState: {
        [Session.Node.SLICE_NAME]: {
          ...Session.Node.ZERO_SLICE_STATE,
          selected: clusterKey,
          clusters: {
            [clusterKey]: {
              key: clusterKey,
              name: "Local",
              host: "localhost",
              port: 9090,
              username: "synnax",
              password: "seldon",
              secure: false,
            },
          },
        },
      },
    });
    const { result } = renderHook(() => Node.useLink(), { wrapper });
    await waitFor(async () => {
      const resolved = await result.current(clusterKey);
      expect(resolved).not.toBeNull();
    });
  });
});
