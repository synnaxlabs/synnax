// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Window } from "@/platform/window";

describe("Window.Buttons", () => {
  it("should call the handler matching each button", () => {
    const onMinimize = vi.fn();
    const onMaximize = vi.fn();
    const onClose = vi.fn();
    render(
      <Window.Buttons
        onClose={onClose}
        onMaximize={onMaximize}
        onMinimize={onMinimize}
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: "Minimize" }));
    fireEvent.click(screen.getByRole("button", { name: "Maximize" }));
    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    expect(onMinimize).toHaveBeenCalledTimes(1);
    expect(onMaximize).toHaveBeenCalledTimes(1);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("should offer a restore button while the window is maximized", () => {
    const onMaximize = vi.fn();
    render(<Window.Buttons maximized onMaximize={onMaximize} />);
    expect(screen.queryByRole("button", { name: "Maximize" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Restore" }));
    expect(onMaximize).toHaveBeenCalledTimes(1);
  });

  it("should leave out a disabled button", () => {
    render(<Window.Buttons disabled={["maximize"]} />);
    expect(screen.queryByRole("button", { name: "Maximize" })).toBeNull();
    expect(screen.getAllByRole("button")).toHaveLength(2);
  });
});
