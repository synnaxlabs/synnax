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
import {
  ClipboardItems,
  type ClipboardItemsProps,
} from "@/vis/diagram/menu/ClipboardItems";

interface RenderProps extends Partial<ClipboardItemsProps> {}

const renderItems = ({
  cut = vi.fn(),
  copy = vi.fn(),
  paste = vi.fn(),
  hasSelection = true,
}: RenderProps = {}) => {
  const c = render(
    <Menu.Menu>
      <ClipboardItems cut={cut} copy={copy} paste={paste} hasSelection={hasSelection} />
    </Menu.Menu>,
  );
  return { c, cut, copy, paste };
};

describe("ClipboardItems", () => {
  it("should run the handler the clicked entry names", () => {
    const { c, cut, copy, paste } = renderItems();
    fireEvent.click(c.getByText("Cut"));
    expect(cut).toHaveBeenCalledTimes(1);
    expect(copy).not.toHaveBeenCalled();
    fireEvent.click(c.getByText("Copy"));
    expect(copy).toHaveBeenCalledTimes(1);
    fireEvent.click(c.getByText("Paste"));
    expect(paste).toHaveBeenCalledTimes(1);
  });

  it("should advertise the shortcuts its host binds", () => {
    const { c } = renderItems();
    // The entries never bind the shortcut themselves, so a hint that drifted from
    // Triggers.CUT / COPY / PASTE would advertise keys that do nothing. Modifiers
    // render as icons, so the cap count is what ties the hint back to the constant.
    const [cutHint, copyHint, pasteHint] = c.getAllByLabelText("trigger-indicator");
    const caps = (el: HTMLElement): number =>
      el.querySelectorAll(".pluto-text--keyboard").length;
    expect(caps(cutHint)).toBe(Triggers.CUT.length);
    expect(caps(copyHint)).toBe(Triggers.COPY.length);
    expect(caps(pasteHint)).toBe(Triggers.PASTE.length);
    expect(cutHint.textContent).toContain("X");
    expect(copyHint.textContent).toContain("C");
    expect(pasteHint.textContent).toContain("V");
  });

  it("should disable cut and copy without a selection", () => {
    const { c, cut, copy, paste } = renderItems({ hasSelection: false });
    const disabled = (label: string): string | null =>
      c.getByText(label).closest("button")?.getAttribute("aria-disabled") ?? null;
    expect(disabled("Cut")).toBe("true");
    expect(disabled("Copy")).toBe("true");
    fireEvent.click(c.getByText("Cut"));
    fireEvent.click(c.getByText("Copy"));
    expect(cut).not.toHaveBeenCalled();
    expect(copy).not.toHaveBeenCalled();
    fireEvent.click(c.getByText("Paste"));
    expect(paste).toHaveBeenCalledTimes(1);
  });
});
