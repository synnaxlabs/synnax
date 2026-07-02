// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { renderTaskFormHook } from "@/platform/task/testutil";

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
    expect(result.current.value.variant).toBe("success");
    expect(result.current.value.message).toBe("Task is running");
    expect(result.current.value.details.running).toBe(true);
  });

  it("should fall back to a default disabled status when absent", async () => {
    const { result } = await renderTaskFormHook({}, (ctx) => Task.useStatus(ctx));
    expect(result.current.value.variant).toBe("disabled");
    expect(result.current.value.message).toBe("Task has not been configured");
    expect(result.current.value.details.running).toBe(false);
  });
});
