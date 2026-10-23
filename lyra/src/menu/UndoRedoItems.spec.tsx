// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Menu } from "@/menu";
import { Triggers } from "@/triggers";

interface RenderProps extends Partial<Menu.UndoRedoItemsProps> {}

const renderItems = ({
  undo = vi.fn(),
  redo = vi.fn(),
  canUndo = true,
  canRedo = true,
}: RenderProps = {}) => {
  const c = render(
    <Menu.Menu>
      <Menu.UndoRedoItems undo={undo} redo={redo} canUndo={canUndo} canRedo={canRedo} />
    </Menu.Menu>,
  );
  return { c, undo, redo };
};

describe("UndoRedoItems", () => {
  it("should run the handler the clicked entry names", () => {
    const { c, undo, redo } = renderItems();
    fireEvent.click(c.getByText("Undo"));
    expect(undo).toHaveBeenCalledTimes(1);
    expect(redo).not.toHaveBeenCalled();
    fireEvent.click(c.getByText("Redo"));
    expect(redo).toHaveBeenCalledTimes(1);
  });

  it("should advertise the shortcuts its host binds", () => {
    const { c } = renderItems();
    // The entries never bind the shortcut themselves, so a hint that drifted from
    // Triggers.UNDO / REDO would advertise keys that do nothing. Modifiers render as
    // icons, so the cap count is what ties the hint back to the constant.
    const [undoHint, redoHint] = c.getAllByLabelText("trigger-indicator");
    const caps = (el: HTMLElement): number =>
      el.querySelectorAll(".pluto-text--keyboard").length;
    expect(caps(undoHint)).toBe(Triggers.UNDO.length);
    expect(caps(redoHint)).toBe(Triggers.REDO.length);
    expect(undoHint.textContent).toContain("Z");
    expect(redoHint.textContent).toContain("Z");
  });

  it("should disable an entry whose direction has no history", () => {
    const { c, undo } = renderItems({ canUndo: false });
    const item = c.getByText("Undo").closest("button");
    expect(item?.getAttribute("aria-disabled")).toBe("true");
    fireEvent.click(c.getByText("Undo"));
    expect(undo).not.toHaveBeenCalled();
  });
});
