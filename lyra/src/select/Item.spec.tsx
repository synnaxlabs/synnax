// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, render, type RenderResult } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Dialog } from "@/dialog";
import { Select } from "@/select";
import { Triggers } from "@/triggers";

interface RenderOptions {
  value?: string;
  visible?: boolean;
  onChange?: (key: string | null) => void;
  search?: boolean;
}

const renderSelect = ({
  value,
  visible = false,
  onChange = vi.fn(),
  search = false,
}: RenderOptions = {}): RenderResult =>
  render(
    <Triggers.Provider>
      <Dialog.Frame visible={visible}>
        <Select.Frame<string, undefined>
          data={["alpha", "bravo"]}
          value={value}
          onChange={onChange}
          allowNone
        >
          <Select.SingleTrigger placeholder="Pick" />
          <Select.Dialog>
            {search && <Select.Search />}
            <Select.List>
              <Select.Item itemKey="none">
                <span>None</span>
              </Select.Item>
              <Select.Items<string>>
                {({ key, ...p }): ReactElement => (
                  <Select.Item key={key} {...p}>
                    {p.itemKey}
                  </Select.Item>
                )}
              </Select.Items>
              <Select.Item itemKey="custom">Custom</Select.Item>
            </Select.List>
          </Select.Dialog>
        </Select.Frame>
      </Dialog.Frame>
    </Triggers.Provider>,
  );

const hovered = (c: RenderResult): string | undefined =>
  c.baseElement.querySelector(".pluto-list__item.pluto--hovered")?.id;

const keyDown = (code: string): void => {
  act(() => {
    fireEvent.keyDown(window, { code });
  });
  act(() => {
    fireEvent.keyUp(window, { code });
  });
};

describe("Select.Item", () => {
  describe("closed dialog", () => {
    it("should show the selected fixed item's children in the trigger", () => {
      const c = renderSelect({ value: "custom" });
      const trigger = c.getByRole("button");
      expect(trigger.textContent).toBe("Custom");
      expect(trigger.querySelector(".pluto-select__label")).not.toBeNull();
    });

    it("should not render unselected fixed items", () => {
      const c = renderSelect({ value: "custom" });
      expect(c.queryByText("None")).toBeNull();
    });

    it("should show the placeholder when nothing is selected", () => {
      const c = renderSelect();
      expect(c.getByRole("button").textContent).toBe("Pick");
    });
  });

  describe("open dialog", () => {
    it("should render fixed items around the data block in page order", () => {
      const c = renderSelect({ visible: true });
      const ids = Array.from(
        c.baseElement.querySelectorAll('[role="option"]'),
        (el) => el.id,
      );
      expect(ids).toEqual(["none", "alpha", "bravo", "custom"]);
    });

    it("should show the selected label in the row and the trigger at once", () => {
      const c = renderSelect({ value: "custom", visible: true });
      expect(c.getAllByText("Custom")).toHaveLength(2);
    });

    it("should walk fixed items and data keys in page order and wrap", () => {
      const c = renderSelect({ visible: true });
      const walk: (string | undefined)[] = [];
      for (let i = 0; i < 5; i++) {
        keyDown("ArrowDown");
        walk.push(hovered(c));
      }
      expect(walk).toEqual(["none", "alpha", "bravo", "custom", "none"]);
    });

    it("should wrap upward from the first fixed item to the last", () => {
      const c = renderSelect({ visible: true });
      keyDown("ArrowDown");
      keyDown("ArrowUp");
      expect(hovered(c)).toBe("custom");
    });

    it("should select the hovered fixed item on Enter", () => {
      const onChange = vi.fn();
      renderSelect({ visible: true, onChange });
      keyDown("ArrowDown");
      keyDown("Enter");
      expect(onChange).toHaveBeenCalledWith("none", { clicked: "none" });
    });

    it("should select a fixed item on click", () => {
      const onChange = vi.fn();
      const c = renderSelect({ visible: true, onChange });
      fireEvent.click(c.getByText("Custom"));
      expect(onChange).toHaveBeenCalledWith("custom", { clicked: "custom" });
    });
  });

  describe("search", () => {
    const type = (c: RenderResult, term: string): void => {
      fireEvent.change(c.getByPlaceholderText("Search..."), {
        target: { value: term },
      });
    };

    it("should hide fixed items whose text does not match", () => {
      const c = renderSelect({ visible: true, search: true });
      type(c, "cus");
      expect(c.baseElement.querySelector("#none")?.hasAttribute("hidden")).toBe(true);
      expect(c.baseElement.querySelector("#custom")?.hasAttribute("hidden")).toBe(
        false,
      );
    });

    it("should match without regard to case", () => {
      const c = renderSelect({ visible: true, search: true });
      type(c, "NON");
      expect(c.baseElement.querySelector("#none")?.hasAttribute("hidden")).toBe(false);
    });

    it("should skip hidden fixed items in the walk", () => {
      const c = renderSelect({ visible: true, search: true });
      type(c, "cus");
      keyDown("ArrowDown");
      expect(hovered(c)).toBe("alpha");
      keyDown("ArrowUp");
      expect(hovered(c)).toBe("custom");
    });
  });
});

describe("Select.MultipleTrigger", () => {
  it("should show a selected fixed item's children in its tag", () => {
    const c = render(
      <Dialog.Frame>
        <Select.Frame<string, undefined>
          multiple
          data={["alpha"]}
          value={["all"]}
          onChange={vi.fn()}
        >
          <Select.MultipleTrigger placeholder="Pick" />
          <Select.Dialog>
            <Select.List>
              <Select.Item itemKey="all">All</Select.Item>
            </Select.List>
          </Select.Dialog>
        </Select.Frame>
      </Dialog.Frame>,
    );
    const tag = c.baseElement.querySelector(".pluto-tag");
    expect(tag?.textContent).toContain("All");
    expect(tag?.className).not.toContain("error");
  });
});
