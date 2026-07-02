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
import { renderInTaskForm } from "@/platform/task/testutil";

describe("ParentRangeButton", () => {
  it("should render nothing when the task has no parent range", async () => {
    const { container } = await renderInTaskForm(<Task.ParentRangeButton />, {
      values: { key: undefined },
    });
    expect(container.textContent).toBe("");
  });
});
