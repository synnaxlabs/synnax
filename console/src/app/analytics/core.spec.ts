// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { emit } from "@tauri-apps/api/event";
import { waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { follow } from "@/app/analytics/core";
import { type Embedded } from "@/feature/embedded";
import { clearSupervisor, mockSupervisor } from "@/feature/embedded/testutil";
import { type Analytics } from "@/platform/analytics";
import { createTestSink } from "@/platform/analytics/testutil";

const FAILED: Embedded.Status = { state: "failed", message: "failed to start" };

const NEVER_STARTED: Embedded.History = {
  starts: 0,
  exits: 3,
  failures: 1,
  readies: 0,
  readyAt: null,
  timeToReadyMs: null,
  lastExit: { reason: "failed_to_start", message: "failed to start", uptimeSeconds: 0 },
};

const EXITED_AFTER_READY: Embedded.History = {
  starts: 1,
  exits: 1,
  failures: 0,
  readies: 1,
  readyAt: null,
  timeToReadyMs: 300,
  lastExit: { reason: "crashed", message: "exited with 3", uptimeSeconds: 1 },
};

const followCore = async (initial: Embedded.History) => {
  let history = initial;
  const sink = createTestSink();
  const supervisor = mockSupervisor(() => FAILED, {
    supervisor_history: () => history,
  });
  await follow(sink);
  const reads = (): number =>
    supervisor.commands.mock.calls.filter(([cmd]) => cmd === "supervisor_history")
      .length;
  return {
    captured: (event: Analytics.Name) =>
      sink.capture.mock.calls.filter(([name]) => name === event),
    /** Delivers a status change and waits until the app has read the next history. */
    change: async (next: Embedded.History): Promise<void> => {
      const before = reads();
      history = next;
      await emit("supervisor://status", FAILED);
      await waitFor(() => expect(reads()).toBe(before + 1));
    },
  };
};

describe("follow", () => {
  afterEach(clearSupervisor);

  it("should report a give-up on a Core that never started", async () => {
    const { captured } = await followCore(NEVER_STARTED);
    expect(captured("core_restart_exhausted")).toEqual([
      ["core_restart_exhausted", { reason: "failed_to_start" }],
    ]);
  });

  it("should report each give-up exactly once", async () => {
    const { captured, change } = await followCore(NEVER_STARTED);
    await change(NEVER_STARTED);
    const second = { ...NEVER_STARTED, exits: 6, failures: 2 };
    await change(second);
    await change(second);
    // Reports finish in order, so once the third give-up lands the rest have too.
    await change({ ...NEVER_STARTED, exits: 9, failures: 3 });
    await waitFor(() => expect(captured("core_restart_exhausted")).toHaveLength(3));
  });

  it("should report a Core that became ready and exited before the history read", async () => {
    const { captured } = await followCore(EXITED_AFTER_READY);
    expect(captured("core_ready")).toEqual([
      ["core_ready", { time_to_ready_ms: 300, starts: 1 }],
    ]);
  });

  it("should report each ready Core exactly once", async () => {
    const { captured, change } = await followCore(EXITED_AFTER_READY);
    await change(EXITED_AFTER_READY);
    // Reports finish in order, so once the second ready lands the rest have too.
    await change({ ...EXITED_AFTER_READY, starts: 2, readies: 2, timeToReadyMs: 200 });
    await waitFor(() => expect(captured("core_ready")).toHaveLength(2));
    expect(captured("core_ready")[1]).toEqual([
      "core_ready",
      { time_to_ready_ms: 200, starts: 2 },
    ]);
  });
});
