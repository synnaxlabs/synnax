// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type connection, type Synnax as Client } from "@synnaxlabs/client";
import { breaker, TimeSpan } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { Cluster } from "@/feature/cluster";
import { Session } from "@/session";
import { type State } from "@/session/store";

const client = (): Client => ({}) as Client;

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

const makeState = (clusterKeys: string[], active: string | null): State =>
  ({
    [Session.Cluster.SLICE_NAME]: {
      clusters: Object.fromEntries(clusterKeys.map((k) => [k, { key: k, name: k }])),
      activeCluster: active,
    },
  }) as unknown as State;

// sequence returns each snapshot once and then repeats the last one indefinitely, so a
// poll loop that overshoots the scripted transitions keeps seeing the terminal state.
const sequence = (...snapshots: Cluster.Snapshot[]) => {
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
      Cluster.connectToCluster("missing", {
        getState: () => makeState([], null),
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
    const active = client();
    const result = await Cluster.connectToCluster("a", {
      getState: () => makeState(["a"], "a"),
      getSnapshot: sequence({ client: active, connState: connState("connected", "a") }),
      setActive,
      poll: instantPoll(),
    });
    expect(result).toBe(active);
    expect(setActive).not.toHaveBeenCalled();
  });

  it("should switch clusters and resolve once the new client connects", async () => {
    const setActive = vi.fn();
    const prior = client();
    const next = client();
    const result = await Cluster.connectToCluster("b", {
      getState: () => makeState(["a", "b"], "a"),
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
    const prior = client();
    const next = client();
    const result = await Cluster.connectToCluster("b", {
      getState: () => makeState(["a", "b"], "a"),
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
    const active = client();
    await expect(
      Cluster.connectToCluster("a", {
        getState: () => makeState(["a"], "a"),
        getSnapshot: sequence(
          { client: active, connState: connState("connecting", "a") },
          { client: active, connState: connState("failed", "a", "auth rejected") },
        ),
        setActive: vi.fn(),
        poll: instantPoll(),
      }),
    ).rejects.toThrow("Timed out connecting to cluster a");
  });

  it("should throw when the connection times out", async () => {
    const active = client();
    await expect(
      Cluster.connectToCluster("a", {
        getState: () => makeState(["a"], "a"),
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
