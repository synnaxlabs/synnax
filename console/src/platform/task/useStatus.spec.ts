// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Task } from "@/platform/task";
import { renderTaskFormHook } from "@/platform/task/testutil";

// The task form declares the field optional: the Core allows a task with no status.
const STATUS_FORM_Z = z.object({ status: task.statusZ().optional().nullable() });

describe("useStatus", () => {
  it("should return the status field value when it is present", async () => {
    const status = {
      key: "status-key",
      name: "Task Status",
      variant: "success",
      message: "Task is running",
      description: "",
      details: { task: "task-1", running: true, cmd: "start", data: {} },
    };
    const { result } = await renderTaskFormHook({ status }, (ctx) =>
      Task.useStatus(ctx),
    );
    expect(result.current.value).toMatchObject({
      variant: "success",
      message: "Task is running",
      details: { running: true },
    });
  });

  it("should return undefined for a task with no status", async () => {
    const { result } = await renderTaskFormHook(
      { status: null },
      (ctx) => Task.useStatus(ctx),
      STATUS_FORM_Z,
    );
    expect(result.current.value).toBeUndefined();
  });

  it("should throw when the form carries no status", async () => {
    await expect(renderTaskFormHook({}, (ctx) => Task.useStatus(ctx))).rejects.toThrow(
      "Field status is not in the form",
    );
  });
});

describe("useIsRunning", () => {
  it("should be true when the status reports the task as running", async () => {
    const { result } = await renderTaskFormHook(
      { status: { details: { running: true } } },
      (ctx) => Task.useIsRunning(ctx),
    );
    expect(result.current.value).toBe(true);
  });

  it("should be false when the status reports the task as not running", async () => {
    const { result } = await renderTaskFormHook(
      { status: { details: { running: false } } },
      (ctx) => Task.useIsRunning(ctx),
    );
    expect(result.current.value).toBe(false);
  });

  it("should be false for a task with no status", async () => {
    const { result } = await renderTaskFormHook(
      { status: null },
      (ctx) => Task.useIsRunning(ctx),
      STATUS_FORM_Z,
    );
    expect(result.current.value).toBe(false);
  });

  it("should throw when the form carries no status", async () => {
    await expect(
      renderTaskFormHook({}, (ctx) => Task.useIsRunning(ctx)),
    ).rejects.toThrow("Field status is not in the form");
  });
});
