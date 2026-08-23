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

const createState = (coreKeys: string[], selected: string | null): State => ({
  ...Session.ZERO_STATE,
  [Session.Core.SLICE_NAME]: {
    ...Session.Core.ZERO_SLICE_STATE,
    selected: selected ?? undefined,
    cores: Object.fromEntries(coreKeys.map((k) => [k, createCore(k)])),
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

describe("connectToCore", () => {
  it("should throw if the Core is unknown", async () => {
    const setActive = vi.fn();
    await expect(
      Core.connectToCore("missing", {
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
    const active = createTestClient();
    const result = await Core.connectToCore("a", {
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
    const result = await Core.connectToCore("b", {
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
      Core.connectToCore("a", {
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
      Core.connectToCore("b", {
        getState: () => createState(["a", "b"], "a"),
        getClient: sequence(prior),
        setActive: vi.fn(),
        poll: instantPoll(3),
      }),
    ).rejects.toThrow("Timed out connecting to Core b");
  });
});

describe("useLink", () => {
  it("should resolve the active Core's managed client", async () => {
    const core = createCore("Local");
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
      const resolved = await result.current(core.key);
      expect(resolved).not.toBeNull();
    });
  });
});
