// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { renderInTaskForm, renderInTaskFormWithClient } from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

describe("Rack", () => {
  it("should render nothing when no rack has been resolved", async () => {
    const { container } = await renderInTaskForm(<Task.Rack />, {
      values: { key: undefined },
    });
    expect(container.textContent).toBe("");
  });

  it("should display the name of the rack the task is deployed to", async () => {
    const client = createTestClient();
    const rack = await client.racks.create({ name: uniqueName("rack") });
    const task = await rack.createTask({
      name: uniqueName("tsk"),
      type: "test_type",
      config: {},
    });
    await renderInTaskFormWithClient(<Task.Rack />, {
      client,
      values: { key: task.key },
    });
    await waitFor(() =>
      expect(screen.getAllByText(rack.name).length).toBeGreaterThan(0),
    );
  });
});
