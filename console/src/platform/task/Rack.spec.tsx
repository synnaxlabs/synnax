// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { renderInTaskForm, renderInTaskFormWithClient } from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

describe("Rack", () => {
  it("should render nothing for a draft task on rack zero", async () => {
    const { container } = await renderInTaskForm(<Task.Rack />, {
      values: { rack: 0 },
    });
    expect(container.textContent).toBe("");
  });

  it("should display the name of the rack the task is assigned to", async () => {
    const client = createTestClient();
    const rack = await client.racks.create({ name: uniqueName("rack") });
    await renderInTaskFormWithClient(<Task.Rack />, {
      client,
      values: { rack: rack.key },
    });
    await waitFor(() =>
      expect(screen.getAllByText(rack.name).length).toBeGreaterThan(0),
    );
  });
});
