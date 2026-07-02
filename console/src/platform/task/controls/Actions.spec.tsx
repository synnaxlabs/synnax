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
import { renderWithConsole } from "@/testutil";

describe("Controls.Actions", () => {
  it("should render its children", async () => {
    await renderWithConsole(
      <Task.Controls.Actions>
        <button type="button">child</button>
      </Task.Controls.Actions>,
    );
    expect(screen.getByRole("button", { name: "child" })).toBeTruthy();
  });

  it("should invoke onClick when clicked", async () => {
    const onClick = vi.fn();
    await renderWithConsole(
      <Task.Controls.Actions onClick={onClick}>
        <span>content</span>
      </Task.Controls.Actions>,
    );
    fireEvent.click(screen.getByText("content"));
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it("should stop click propagation to ancestors", async () => {
    const parentClick = vi.fn();
    const actionsClick = vi.fn();
    await renderWithConsole(
      <div onClick={parentClick}>
        <Task.Controls.Actions onClick={actionsClick}>
          <span>content</span>
        </Task.Controls.Actions>
      </div>,
    );
    fireEvent.click(screen.getByText("content"));
    expect(actionsClick).toHaveBeenCalledTimes(1);
    expect(parentClick).not.toHaveBeenCalled();
  });
});
