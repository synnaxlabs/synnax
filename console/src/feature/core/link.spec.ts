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
import { describe, expect, it, vi } from "vitest";

import { Core } from "@/feature/core";
import { Session } from "@/session";
import { CONNECTION_PARAMS, createCore } from "@/session/core/testutil";
import { type State } from "@/session/store";
import { createConnectedConsoleWrapper } from "@/testutil";

// Each Core is keyed and named after the cluster it reaches, so a spec names one Core
// by the cluster key a link would carry.
const createState = (clusterKeys: string[], selected: string | null): State => ({
  ...Session.ZERO_STATE,
  [Session.Core.SLICE_NAME]: {
    ...Session.Core.ZERO_SLICE_STATE,
    selected: selected ?? undefined,
    cores: Object.fromEntries(
      clusterKeys.map((k) => [k, createCore(k, { key: k, clusterKey: k })]),
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

describe("connect", () => {
  it("should throw if no Core reaches the cluster", async () => {
    const setActive = vi.fn();
    await expect(
      Core.connect("missing", {
        getState: () => createState([], null),
        getClient: sequence(null),
        setActive,
        poll: instantPoll(),
      }),
    ).rejects.toThrow("No Core connects to cluster missing");
    expect(setActive).not.toHaveBeenCalled();
  });

  // Two records reaching one cluster is the whole point of keying links by cluster: a
  // link made against one machine's record opens through whichever record this machine
  // holds for the same cluster.
  it("should resolve a cluster through the Core that names it", async () => {
    const setActive = vi.fn();
    const prior = createTestClient();
    const next = createTestClient();
    const state: State = {
      ...Session.ZERO_STATE,
      [Session.Core.SLICE_NAME]: {
        ...Session.Core.ZERO_SLICE_STATE,
        selected: "here",
        cores: {
          here: createCore("Here", { key: "here", clusterKey: "cluster-a" }),
          there: createCore("There", { key: "there", clusterKey: "cluster-b" }),
        },
      },
    };
    const result = await Core.connect("cluster-b", {
      getState: () => state,
      getClient: sequence(prior, prior, next),
      setActive,
      poll: instantPoll(),
    });
    expect(result).toBe(next);
    expect(setActive).toHaveBeenCalledWith("there");
  });

  it("should return the managed client when already active", async () => {
    const setActive = vi.fn();
    const active = createTestClient();
    const result = await Core.connect("a", {
      getState: () => createState(["a"], "a"),
      getClient: sequence(active),
      setActive,
      poll: instantPoll(),
    });
    expect(result).toBe(active);
    expect(setActive).not.toHaveBeenCalled();
  });

  it("should switch Cores and resolve once the provider swaps clients", async () => {
    const setActive = vi.fn();
    const prior = createTestClient();
    const next = createTestClient();
    const result = await Core.connect("b", {
      getState: () => createState(["a", "b"], "a"),
      getClient: sequence(prior, prior, next),
      setActive,
      poll: instantPoll(),
    });
    expect(result).toBe(next);
    expect(setActive).toHaveBeenCalledWith("b");
  });

  it("should reject with the client's typed error when the connection fails", async () => {
    const dead = createTestClient({
      port: 9999,
      retry: { baseInterval: TimeSpan.milliseconds(5), scale: 1, maxRetries: 1 },
    });
    await expect(
      Core.connect("a", {
        getState: () => createState(["a"], "a"),
        getClient: sequence(dead),
        setActive: vi.fn(),
        poll: instantPoll(),
      }),
    ).rejects.toThrow();
  });

  it("should throw when the provider never swaps clients", async () => {
    const prior = createTestClient();
    await expect(
      Core.connect("b", {
        getState: () => createState(["a", "b"], "a"),
        getClient: sequence(prior),
        setActive: vi.fn(),
        poll: instantPoll(3),
      }),
    ).rejects.toThrow("Timed out connecting to cluster b");
  });
});

describe("useLink", () => {
  it("should resolve the active Core's managed client", async () => {
    const core = createCore("Local", { clusterKey: "local-cluster" });
    const { wrapper } = await createConnectedConsoleWrapper({
      client: null,
      connParams: CONNECTION_PARAMS,
      preloadedState: {
        [Session.Core.SLICE_NAME]: {
          ...Session.Core.ZERO_SLICE_STATE,
          selected: core.key,
          cores: { [core.key]: core },
        },
      },
    });
    const { result } = renderHook(() => Core.useLink(), { wrapper });
    await waitFor(async () => {
      const resolved = await result.current("local-cluster");
      expect(resolved).not.toBeNull();
    });
  });
});
