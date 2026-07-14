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

describe("useKey", () => {
  it("should return the value of the key field", async () => {
    const { result } = await renderTaskFormHook({ key: "task-123" }, (ctx) =>
      Task.useKey(ctx),
    );
    expect(result.current.value).toBe("task-123");
  });

  it("should return null when the key field is absent", async () => {
    const { result } = await renderTaskFormHook({}, (ctx) => Task.useKey(ctx));
    expect(result.current.value).toBeNull();
  });

  it("should return a zero key verbatim, since draft rows carry real keys", async () => {
    const { result } = await renderTaskFormHook({ key: "0" }, (ctx) =>
      Task.useKey(ctx),
    );
    expect(result.current.value).toBe("0");
  });
});
