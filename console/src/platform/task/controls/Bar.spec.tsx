// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Task } from "@/platform/task";
import { createTaskStatus } from "@/platform/task/testutil";
import { getIconButton, renderWithConsole } from "@/testutil";

describe("Controls.Bar", () => {
  it("should deploy on play and stop on pause", async () => {
    const onDeploy = vi.fn();
    const onStop = vi.fn();
    const { container, rerender } = await renderWithConsole(
      <Task.Controls.Bar
        status={createTaskStatus()}
        running={false}
        onDeploy={onDeploy}
        onStop={onStop}
      />,
    );
    fireEvent.click(getIconButton(container, "play"));
    expect(onDeploy).toHaveBeenCalledTimes(1);
    expect(onStop).not.toHaveBeenCalled();

    rerender(
      <Task.Controls.Bar
        status={createTaskStatus()}
        running
        onDeploy={onDeploy}
        onStop={onStop}
      />,
    );
    fireEvent.click(getIconButton(container, "pause"));
    expect(onStop).toHaveBeenCalledTimes(1);
    expect(onDeploy).toHaveBeenCalledTimes(1);
  });

  it("should invoke onDeploy from the redeploy button when drifted", async () => {
    const onDeploy = vi.fn();
    await renderWithConsole(
      <Task.Controls.Bar
        status={createTaskStatus()}
        running
        drifted
        onDeploy={onDeploy}
        onStop={vi.fn()}
      />,
    );
    fireEvent.click(screen.getByText("Redeploy"));
    expect(onDeploy).toHaveBeenCalledTimes(1);
  });

  it("should render extra actions in the action row and hide them on snapshots", async () => {
    const { rerender } = await renderWithConsole(
      <Task.Controls.Bar
        status={createTaskStatus()}
        running={false}
        extraActions={<button>Pick rack</button>}
        onDeploy={vi.fn()}
        onStop={vi.fn()}
      />,
    );
    expect(screen.getByText("Pick rack")).toBeTruthy();

    rerender(
      <Task.Controls.Bar
        status={createTaskStatus()}
        running={false}
        snapshot
        extraActions={<button>Pick rack</button>}
        onDeploy={vi.fn()}
        onStop={vi.fn()}
      />,
    );
    expect(screen.queryByText("Pick rack")).toBeNull();
  });
});
