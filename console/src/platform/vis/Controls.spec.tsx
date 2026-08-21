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

import { Vis } from "@/platform/vis";
import { renderWithConsole } from "@/testutil";

describe("Vis.Controls", () => {
  it("renders children and merges the controls class with a caller class", async () => {
    await renderWithConsole(
      <Vis.Controls className="extra">
        <span>content</span>
      </Vis.Controls>,
    );
    const child = screen.getByText("content");
    const box = child.parentElement;
    expect(box?.className).toContain("console-controls");
    expect(box?.className).toContain("extra");
  });

  it("keeps a double click off the visualization behind it", async () => {
    const onDoubleClick = vi.fn();
    await renderWithConsole(
      <div onDoubleClick={onDoubleClick}>
        <Vis.Controls>
          <button>Pause</button>
        </Vis.Controls>
      </div>,
    );
    fireEvent.doubleClick(screen.getByRole("button", { name: "Pause" }));
    expect(onDoubleClick).not.toHaveBeenCalled();
  });

  it("lets a caller override the double click handler", async () => {
    const onDoubleClick = vi.fn();
    await renderWithConsole(
      <Vis.Controls onDoubleClick={onDoubleClick}>
        <button>Pause</button>
      </Vis.Controls>,
    );
    fireEvent.doubleClick(screen.getByRole("button", { name: "Pause" }));
    expect(onDoubleClick).toHaveBeenCalledOnce();
  });
});
