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
import { queryIcon, renderWithConsole } from "@/testutil";

describe("Controls.StartStopButton", () => {
  it("should swap from the play icon to the stop icon when running flips", async () => {
    const { container, rerender } = await renderWithConsole(
      <Task.Controls.StartStopButton running={false} onClick={vi.fn()} />,
    );
    expect(queryIcon(container, "play")).toBeTruthy();
    expect(queryIcon(container, "stop")).toBeNull();

    rerender(<Task.Controls.StartStopButton running onClick={vi.fn()} />);
    expect(queryIcon(container, "stop")).toBeTruthy();
    expect(queryIcon(container, "play")).toBeNull();
  });

  it("should style the stop state as an error and keep the start state neutral", async () => {
    const { rerender } = await renderWithConsole(
      <Task.Controls.StartStopButton running={false} onClick={vi.fn()} />,
    );
    const button = screen.getByRole("button");
    expect(button.className).not.toContain("pluto--status-error");

    rerender(<Task.Controls.StartStopButton running onClick={vi.fn()} />);
    expect(button.className).toContain("pluto--status-error");
  });

  it("should let a loading status variant override the running error styling", async () => {
    await renderWithConsole(
      <Task.Controls.StartStopButton
        running
        statusVariant="loading"
        onClick={vi.fn()}
      />,
    );
    const button = screen.getByRole("button");
    expect(button.className).toContain("pluto--status-loading");
    expect(button.className).not.toContain("pluto--status-error");
  });

  it("should invoke onClick when pressed", async () => {
    const onClick = vi.fn();
    await renderWithConsole(
      <Task.Controls.StartStopButton running={false} onClick={onClick} />,
    );
    fireEvent.click(screen.getByRole("button"));
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it("should not invoke onClick when disabled", async () => {
    const onClick = vi.fn();
    await renderWithConsole(
      <Task.Controls.StartStopButton running={false} onClick={onClick} disabled />,
    );
    fireEvent.click(screen.getByRole("button"));
    expect(onClick).not.toHaveBeenCalled();
  });
});
