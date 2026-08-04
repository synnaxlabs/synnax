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

import { StartStopButton } from "@/platform/task/controls/StartStopButton";
import { queryIcon, renderWithConsole } from "@/testutil";

describe("Controls.StartStopButton", () => {
  it("should swap from the play icon to the pause icon when running flips", async () => {
    const { container, rerender } = await renderWithConsole(
      <StartStopButton running={false} onClick={vi.fn()} />,
    );
    expect(queryIcon(container, "play")).toBeTruthy();
    expect(queryIcon(container, "pause")).toBeNull();

    rerender(<StartStopButton running onClick={vi.fn()} />);
    expect(queryIcon(container, "pause")).toBeTruthy();
    expect(queryIcon(container, "play")).toBeNull();
  });

  it("should invoke onClick when pressed", async () => {
    const onClick = vi.fn();
    await renderWithConsole(<StartStopButton running={false} onClick={onClick} />);
    fireEvent.click(screen.getByRole("button"));
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it("should not invoke onClick when disabled", async () => {
    const onClick = vi.fn();
    await renderWithConsole(
      <StartStopButton running={false} onClick={onClick} disabled />,
    );
    fireEvent.click(screen.getByRole("button"));
    expect(onClick).not.toHaveBeenCalled();
  });
});
