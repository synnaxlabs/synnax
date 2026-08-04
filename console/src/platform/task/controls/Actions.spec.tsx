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

import { Actions } from "@/platform/task/controls/Actions";
import { renderWithConsole } from "@/testutil";

describe("Controls.Actions", () => {
  it("should invoke its own onClick while stopping propagation to ancestors", async () => {
    const parentClick = vi.fn();
    const actionsClick = vi.fn();
    await renderWithConsole(
      <div onClick={parentClick}>
        <Actions onClick={actionsClick}>
          <span>content</span>
        </Actions>
      </div>,
    );
    fireEvent.click(screen.getByText("content"));
    expect(actionsClick).toHaveBeenCalledTimes(1);
    expect(parentClick).not.toHaveBeenCalled();
  });

  it("should stop propagation even when no onClick handler is provided", async () => {
    const parentClick = vi.fn();
    await renderWithConsole(
      <div onClick={parentClick}>
        <Actions>
          <span>content</span>
        </Actions>
      </div>,
    );
    fireEvent.click(screen.getByText("content"));
    expect(parentClick).not.toHaveBeenCalled();
  });
});
