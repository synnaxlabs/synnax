// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { breaker, TimeSpan } from "@synnaxlabs/x";
import { renderHook, waitFor } from "@testing-library/react";
import { afterAll, describe, expect, it, vi } from "vitest";

import { Cluster } from "@/feature/cluster";
import { Session } from "@/session";
import { type State } from "@/session/store";
import { createConnectedConsoleWrapper } from "@/testutil";

const createState = (clusterKeys: string[], selected: string | null): State => ({
  ...Session.ZERO_STATE,
  [Session.Cluster.SLICE_NAME]: {
    ...Session.Cluster.ZERO_SLICE_STATE,
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

// sequence returns each client once and then repeats the last one indefinitely, so a
// poll loop that overshoots the scripted swap keeps seeing the terminal client.
const sequence = (...clients: (Client | null)[]) => {
  let i = 0;
  return () => clients[Math.min(i++, clients.length - 1)];
};

const instantPoll = (maxRetries = 1_000) =>
  new breaker.Breaker({
    baseInterval: TimeSpan.milliseconds(1),
    scale: 1,
    maxRetries,
    sleepFn: async () => {},
  });

const openClients: Client[] = [];
const testClient = (params?: Parameters<typeof createTestClient>[0]): Client => {
  const client = createTestClient(params);
  openClients.push(client);
  return client;
};

afterAll(() => {
  openClients.forEach((c) => c.close());
});

describe("connectToCluster", () => {
  it("should throw if the cluster is unknown", async () => {
    const setActive = vi.fn();
    await expect(
      Cluster.connectToCluster("missing", {
        getState: () => createState([], null),
        getClient: sequence(null),
        setActive,
        poll: instantPoll(),
      }),
    ).rejects.toThrow("Core with key missing not found");
    expect(setActive).not.toHaveBeenCalled();
  });

  it("should return the managed client when already active", async () => {
    const setActive = vi.fn();
    const active = testClient();
    const result = await Cluster.connectToCluster("a", {
      getState: () => createState(["a"], "a"),
      getClient: sequence(active),
      setActive,
      poll: instantPoll(),
    });
    expect(result).toBe(active);
    expect(setActive).not.toHaveBeenCalled();
  });

  it("should switch clusters and resolve once the provider swaps clients", async () => {
    const setActive = vi.fn();
    const prior = testClient();
    const next = testClient();
    const result = await Cluster.connectToCluster("b", {
      getState: () => createState(["a", "b"], "a"),
      getClient: sequence(prior, prior, next),
      setActive,
      poll: instantPoll(),
    });
    expect(result).toBe(next);
    expect(setActive).toHaveBeenCalledWith("b");
  });

  it("should reject with the client's typed error when the connection fails", async () => {
    const dead = testClient({
      port: 9999,
      retry: { baseInterval: TimeSpan.milliseconds(5), scale: 1, maxRetries: 1 },
    });
    await expect(
      Cluster.connectToCluster("a", {
        getState: () => createState(["a"], "a"),
        getClient: sequence(dead),
        setActive: vi.fn(),
        poll: instantPoll(),
      }),
    ).rejects.toThrow();
  });

  it("should throw when the provider never swaps clients", async () => {
    const prior = testClient();
    await expect(
      Cluster.connectToCluster("b", {
        getState: () => createState(["a", "b"], "a"),
        getClient: sequence(prior),
        setActive: vi.fn(),
        poll: instantPoll(3),
      }),
    ).rejects.toThrow("Timed out connecting to cluster b");
  });
});

describe("useLink", () => {
  it("should resolve the active cluster's managed client", async () => {
    const c = testClient();
    const {
      details: { clusterKey },
    } = await c.connect();
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
        [Session.Cluster.SLICE_NAME]: {
          ...Session.Cluster.ZERO_SLICE_STATE,
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
    const { result } = renderHook(() => Cluster.useLink(), { wrapper });
    await waitFor(async () => {
      const resolved = await result.current(clusterKey);
      expect(resolved).not.toBeNull();
    });
  });
});
