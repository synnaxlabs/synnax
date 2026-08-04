// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { renderTaskFormHook, type TaskFormValues } from "@/platform/task/testutil";

describe("useDrifted", () => {
  const DEPLOYED = "2de66015b3bdded8";
  const EDITED = "0000000000000000";
  const newValues = (overrides: {
    running?: boolean;
    taskHash?: string;
    statusHash?: string;
    statusRack?: number;
  }): TaskFormValues => {
    const key = uuid.create();
    const {
      running = true,
      taskHash = DEPLOYED,
      statusHash = DEPLOYED,
      statusRack = 1,
    } = overrides;
    return {
      key,
      rack: 1,
      configHash: taskHash,
      status: task.statusZ().parse({
        message: "running",
        variant: "success",
        details: { task: key, running, configHash: statusHash, rack: statusRack },
      }),
    };
  };

  it("should not drift when the deployed hash and rack match", async () => {
    const { result } = await renderTaskFormHook(newValues({}), (ctx) =>
      Task.useDrifted(ctx),
    );
    expect(result.current.value).toBe(false);
  });

  it("should drift when the saved hash differs from the deployed hash", async () => {
    const { result } = await renderTaskFormHook(
      newValues({ taskHash: EDITED }),
      (ctx) => Task.useDrifted(ctx),
    );
    expect(result.current.value).toBe(true);
  });

  it("should drift when the task moved racks while running", async () => {
    const { result } = await renderTaskFormHook(newValues({ statusRack: 2 }), (ctx) =>
      Task.useDrifted(ctx),
    );
    expect(result.current.value).toBe(true);
  });

  it("should never drift when the task is not running", async () => {
    const { result } = await renderTaskFormHook(
      newValues({ running: false, taskHash: EDITED }),
      (ctx) => Task.useDrifted(ctx),
    );
    expect(result.current.value).toBe(false);
  });

  it("should never drift when the deployed hash is unknown", async () => {
    // The optimistic command-loading status reports running with an empty hash.
    const { result } = await renderTaskFormHook(
      newValues({ statusHash: "", taskHash: EDITED, statusRack: 2 }),
      (ctx) => Task.useDrifted(ctx),
    );
    expect(result.current.value).toBe(false);
  });
});
