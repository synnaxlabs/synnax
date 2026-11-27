// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { treatment } from "@synnaxlabs/lyra/styletest";
import { TimeStamp } from "@synnaxlabs/x";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Task } from "@/platform/task";
import { createTaskStatus, renderInTaskFormWithClient } from "@/platform/task/testutil";

const client = createTestClient();

describe("Controls.Controls status", () => {
  it("should draw the time with tabular digits and the description in Inter", async () => {
    const time = TimeStamp.now();
    await renderInTaskFormWithClient(
      <Task.Controls.Controls onDeploy={vi.fn()} onStop={vi.fn()} />,
      {
        client,
        values: {
          status: createTaskStatus({
            message: "Running smoothly",
            description: "Read 8 channels",
            time,
          }),
        },
      },
    );
    fireEvent.click(await screen.findByText("Running smoothly"));
    await screen.findByText("Copy diagnostics");
    expect(treatment(screen.getByText(time.toString("time", "local")))).toBe("numeric");
    expect(treatment(screen.getByText("Read 8 channels"))).toBe("prose");
  });
});
