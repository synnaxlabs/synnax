// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createEvent, fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { type MouseEventHandler, type ReactElement, useEffect, useState } from "react";
import { describe, expect, it, vi } from "vitest";

import { Haul } from "@/haul";
import { Select } from "@/select";
import { Tabs } from "@/tabs";

interface BasicTabsProps {
  initialValue?: string;
  value?: string;
  onChange?: (key: string) => void;
  onClose?: (key: string) => void;
}

const BasicTabs = (props: BasicTabsProps): ReactElement => (
  <Tabs.Frame {...props}>
    <Tabs.Selector>
      <Tabs.Tab itemKey="a">Tab A</Tabs.Tab>
      <Tabs.Tab itemKey="b">Tab B</Tabs.Tab>
      <Tabs.Tab itemKey="c">Tab C</Tabs.Tab>
    </Tabs.Selector>
    <Tabs.Content itemKey="a">
      <span>Content A</span>
    </Tabs.Content>
    <Tabs.Content itemKey="b">
      <span>Content B</span>
    </Tabs.Content>
    <Tabs.Content itemKey="c">
      <span>Content C</span>
    </Tabs.Content>
  </Tabs.Frame>
);

const tab = (name: string): HTMLElement => screen.getByRole("tab", { name });

describe("Tabs", () => {
  describe("Frame", () => {
    it("should render the initially selected tab's content when uncontrolled", () => {
      render(<BasicTabs initialValue="a" />);
      expect(screen.getByText("Content A")).toBeTruthy();
      expect(screen.queryByText("Content B")).toBeNull();
    });

    it("should switch content when another tab is clicked", () => {
      render(<BasicTabs initialValue="a" />);
      fireEvent.click(tab("Tab B"));
      expect(screen.getByText("Content B")).toBeTruthy();
      expect(screen.queryByText("Content A")).toBeNull();
    });

    it("should notify onChange when uncontrolled", () => {
      const onChange = vi.fn();
      render(<BasicTabs initialValue="a" onChange={onChange} />);
      fireEvent.click(tab("Tab C"));
      expect(onChange).toHaveBeenCalledTimes(1);
      expect(onChange).toHaveBeenCalledWith("c");
      expect(screen.getByText("Content C")).toBeTruthy();
    });

    it("should follow the value prop when controlled", () => {
      const Harness = (): ReactElement => {
        const [value, setValue] = useState("a");
        return <BasicTabs value={value} onChange={setValue} />;
      };
      render(<Harness />);
      expect(screen.getByText("Content A")).toBeTruthy();
      fireEvent.click(tab("Tab B"));
      expect(screen.getByText("Content B")).toBeTruthy();
      expect(screen.queryByText("Content A")).toBeNull();
    });

    it("should not fire onChange when the selected tab is clicked again", () => {
      const onChange = vi.fn();
      render(<BasicTabs initialValue="a" onChange={onChange} />);
      fireEvent.click(tab("Tab A"));
      expect(onChange).not.toHaveBeenCalled();
    });

    it("should render no panel when nothing is selected", () => {
      render(<BasicTabs />);
      expect(screen.queryByRole("tabpanel")).toBeNull();
    });

    it("should bind tabs to an enclosing selection when given none", () => {
      const onSelect = vi.fn();
      render(
        <Select.Context value={["b"]} onSelect={onSelect}>
          <BasicTabs />
        </Select.Context>,
      );
      expect(screen.getByText("Content B")).toBeTruthy();
      expect(screen.queryByText("Content A")).toBeNull();
      fireEvent.click(tab("Tab A"));
      expect(onSelect).toHaveBeenCalledWith("a");
    });

    it("should shadow an enclosing selection when it owns one", () => {
      render(
        <Select.Context value={["b"]}>
          <BasicTabs initialValue="a" />
        </Select.Context>,
      );
      expect(screen.getByText("Content A")).toBeTruthy();
      expect(screen.queryByText("Content B")).toBeNull();
    });
  });

  describe("accessibility", () => {
    it("should wire tablist, tab, and tabpanel roles together", () => {
      render(<BasicTabs initialValue="a" />);
      const tablist = screen.getByRole("tablist");
      expect(tablist.getAttribute("aria-orientation")).toEqual("horizontal");
      const selected = tab("Tab A");
      expect(selected.getAttribute("aria-selected")).toEqual("true");
      expect(tab("Tab B").getAttribute("aria-selected")).toEqual("false");
      const panel = screen.getByRole("tabpanel");
      expect(selected.getAttribute("aria-controls")).toEqual(panel.id);
      expect(panel.getAttribute("aria-labelledby")).toEqual(selected.id);
    });

    it("should give the selected tab the roving tab stop", () => {
      render(<BasicTabs initialValue="b" />);
      expect(tab("Tab A").tabIndex).toEqual(-1);
      expect(tab("Tab B").tabIndex).toEqual(0);
      expect(tab("Tab C").tabIndex).toEqual(-1);
    });

    it("should expose the tab key via a data attribute", () => {
      render(<BasicTabs initialValue="a" />);
      expect(tab("Tab A").getAttribute("data-tab-key")).toEqual("a");
    });
  });

  describe("keyboard navigation", () => {
    it("should move focus without selecting on arrow keys", async () => {
      render(<BasicTabs initialValue="a" />);
      tab("Tab A").focus();
      await userEvent.keyboard("{ArrowRight}");
      expect(document.activeElement).toBe(tab("Tab B"));
      expect(screen.getByText("Content A")).toBeTruthy();
      expect(screen.queryByText("Content B")).toBeNull();
    });

    it("should select the focused tab on Enter", async () => {
      render(<BasicTabs initialValue="a" />);
      tab("Tab A").focus();
      await userEvent.keyboard("{ArrowRight}");
      await userEvent.keyboard("{Enter}");
      expect(screen.getByText("Content B")).toBeTruthy();
    });

    it("should select the focused tab on Space", async () => {
      render(<BasicTabs initialValue="a" />);
      tab("Tab B").focus();
      await userEvent.keyboard("[Space]");
      expect(screen.getByText("Content B")).toBeTruthy();
    });

    it("should wrap focus around the ends of the strip", async () => {
      render(<BasicTabs initialValue="a" />);
      const c = tab("Tab C");
      c.focus();
      await userEvent.keyboard("{ArrowRight}");
      expect(document.activeElement).toBe(tab("Tab A"));
      await userEvent.keyboard("{ArrowLeft}");
      expect(document.activeElement).toBe(c);
    });

    it("should jump to the first and last tabs on Home and End", async () => {
      render(<BasicTabs initialValue="b" />);
      tab("Tab B").focus();
      await userEvent.keyboard("{End}");
      expect(document.activeElement).toBe(tab("Tab C"));
      await userEvent.keyboard("{Home}");
      expect(document.activeElement).toBe(tab("Tab A"));
    });

    it("should use vertical arrow keys when the selector is vertical", async () => {
      render(
        <Tabs.Frame initialValue="a" x>
          <Tabs.Selector y>
            <Tabs.Tab itemKey="a">Tab A</Tabs.Tab>
            <Tabs.Tab itemKey="b">Tab B</Tabs.Tab>
          </Tabs.Selector>
          <Tabs.Content itemKey="a">
            <span>Content A</span>
          </Tabs.Content>
        </Tabs.Frame>,
      );
      expect(screen.getByRole("tablist").getAttribute("aria-orientation")).toEqual(
        "vertical",
      );
      const a = tab("Tab A");
      a.focus();
      await userEvent.keyboard("{ArrowRight}");
      expect(document.activeElement).toBe(a);
      await userEvent.keyboard("{ArrowDown}");
      expect(document.activeElement).toBe(tab("Tab B"));
    });

    it("should not rove focus when arrow keys fire inside a tab's children", async () => {
      render(
        <Tabs.Frame initialValue="a">
          <Tabs.Selector>
            <Tabs.Tab itemKey="a">
              <input data-testid="child-input" />
            </Tabs.Tab>
            <Tabs.Tab itemKey="b">Tab B</Tabs.Tab>
          </Tabs.Selector>
        </Tabs.Frame>,
      );
      const child = screen.getByTestId("child-input");
      child.focus();
      await userEvent.keyboard("{ArrowRight}");
      expect(document.activeElement).toBe(child);
    });
  });

  describe("Content", () => {
    it("should always render keyless content", () => {
      render(
        <Tabs.Frame initialValue="a">
          <Tabs.Selector>
            <Tabs.Tab itemKey="a">Tab A</Tabs.Tab>
          </Tabs.Selector>
          <Tabs.Content>
            <span>Always visible</span>
          </Tabs.Content>
        </Tabs.Frame>,
      );
      expect(screen.getByText("Always visible")).toBeTruthy();
    });

    it("should keep keepMounted content mounted and hidden while unselected", () => {
      const mounts = vi.fn();
      const MountProbe = (): ReactElement => {
        useEffect(() => mounts(), []);
        return <span>Content B</span>;
      };
      render(
        <Tabs.Frame initialValue="a">
          <Tabs.Selector>
            <Tabs.Tab itemKey="a">Tab A</Tabs.Tab>
            <Tabs.Tab itemKey="b">Tab B</Tabs.Tab>
          </Tabs.Selector>
          <Tabs.Content itemKey="a">
            <span>Content A</span>
          </Tabs.Content>
          <Tabs.Content itemKey="b" keepMounted>
            <MountProbe />
          </Tabs.Content>
        </Tabs.Frame>,
      );
      expect(mounts).toHaveBeenCalledTimes(1);
      const hidden = screen.getByText("Content B").parentElement;
      expect(hidden?.hasAttribute("hidden")).toBe(true);
      fireEvent.click(tab("Tab B"));
      expect(screen.getByText("Content B").parentElement?.hasAttribute("hidden")).toBe(
        false,
      );
      fireEvent.click(tab("Tab A"));
      expect(mounts).toHaveBeenCalledTimes(1);
    });

    it("should remount content without keepMounted on reselection", () => {
      const mounts = vi.fn();
      const MountProbe = (): ReactElement => {
        useEffect(() => mounts(), []);
        return <span>Content A</span>;
      };
      render(
        <Tabs.Frame initialValue="a">
          <Tabs.Selector>
            <Tabs.Tab itemKey="a">Tab A</Tabs.Tab>
            <Tabs.Tab itemKey="b">Tab B</Tabs.Tab>
          </Tabs.Selector>
          <Tabs.Content itemKey="a">
            <MountProbe />
          </Tabs.Content>
          <Tabs.Content itemKey="b">
            <span>Content B</span>
          </Tabs.Content>
        </Tabs.Frame>,
      );
      expect(mounts).toHaveBeenCalledTimes(1);
      fireEvent.click(tab("Tab B"));
      expect(screen.queryByText("Content A")).toBeNull();
      fireEvent.click(tab("Tab A"));
      expect(mounts).toHaveBeenCalledTimes(2);
    });
  });

  describe("Close", () => {
    const ClosableTabs = ({ onClose }: BasicTabsProps): ReactElement => {
      const close =
        (key: string): MouseEventHandler =>
        (e) => {
          e.stopPropagation();
          onClose?.(key);
        };
      return (
        <Tabs.Frame initialValue="a">
          <Tabs.Selector>
            <Tabs.Tab itemKey="a">
              Tab A
              <Tabs.Close onClick={close("a")} />
            </Tabs.Tab>
            <Tabs.Tab itemKey="b">
              Tab B
              <Tabs.Close onClick={close("b")} />
            </Tabs.Tab>
          </Tabs.Selector>
          <Tabs.Content itemKey="a">
            <span>Content A</span>
          </Tabs.Content>
          <Tabs.Content itemKey="b">
            <span>Content B</span>
          </Tabs.Content>
        </Tabs.Frame>
      );
    };

    it("should call the Frame's onClose with the tab's key", () => {
      const onClose = vi.fn();
      render(<ClosableTabs onClose={onClose} />);
      const closeButtons = screen.getAllByRole("button", { name: /close/i });
      expect(closeButtons).toHaveLength(2);
      fireEvent.click(closeButtons[1]);
      expect(onClose).toHaveBeenCalledTimes(1);
      expect(onClose).toHaveBeenCalledWith("b");
    });

    it("should not select the tab whose close button is clicked", () => {
      const onClose = vi.fn();
      render(<ClosableTabs onClose={onClose} />);
      fireEvent.click(screen.getAllByRole("button", { name: /close/i })[1]);
      expect(screen.getByText("Content A")).toBeTruthy();
      expect(screen.queryByText("Content B")).toBeNull();
    });
  });

  describe("Tab", () => {
    it("should render arbitrary children such as icons alongside text", () => {
      render(
        <Tabs.Frame initialValue="a">
          <Tabs.Selector>
            <Tabs.Tab itemKey="a">
              <svg data-testid="icon" />
              npm
            </Tabs.Tab>
          </Tabs.Selector>
        </Tabs.Frame>,
      );
      const t = tab("npm");
      expect(t.querySelector("[data-testid='icon']")).toBeTruthy();
    });

    it("should support drag handles via native draggable props", () => {
      const onDragStart = vi.fn();
      const onDragEnd = vi.fn();
      render(
        <Tabs.Frame initialValue="a">
          <Tabs.Selector>
            <Tabs.Tab
              itemKey="a"
              draggable
              onDragStart={onDragStart}
              onDragEnd={onDragEnd}
            >
              Tab A
            </Tabs.Tab>
          </Tabs.Selector>
        </Tabs.Frame>,
      );
      const t = tab("Tab A");
      expect(t.draggable).toBe(true);
      fireEvent.dragStart(t);
      expect(onDragStart).toHaveBeenCalledTimes(1);
      fireEvent.dragEnd(t);
      expect(onDragEnd).toHaveBeenCalledTimes(1);
    });

    it("should color the tab heading an ordered multi-selection", () => {
      render(
        <Select.Context value={["b", "a"]}>
          <BasicTabs />
        </Select.Context>,
      );
      expect(tab("Tab B").classList.contains("pluto--alt-color")).toBe(true);
      expect(tab("Tab A").classList.contains("pluto--alt-color")).toBe(false);
    });

    it("should not color the selected tab of a frame-owned scalar selection", () => {
      render(<BasicTabs initialValue="a" />);
      expect(tab("Tab A").classList.contains("pluto--alt-color")).toBe(false);
    });

    it("should throw when rendered outside a Frame", () => {
      expect(() =>
        render(
          <Tabs.Selector>
            <Tabs.Tab itemKey="a">Tab A</Tabs.Tab>
          </Tabs.Selector>,
        ),
      ).toThrow("Tabs.Tab must be used within Tabs.Frame");
    });
  });

  describe("Selector drag-and-drop", () => {
    const TAB_TYPE = "tab";

    const stubRect = (el: Element, x: number, w: number): void => {
      vi.spyOn(el, "getBoundingClientRect").mockReturnValue({
        x,
        y: 0,
        width: w,
        height: 32,
        top: 0,
        bottom: 32,
        left: x,
        right: x + w,
        toJSON: () => ({}),
      });
    };

    const stubStripRects = (tablist: HTMLElement): void => {
      tablist
        .querySelectorAll<HTMLElement>("[data-tab-key]")
        .forEach((t, i) => stubRect(t, i * 100, 100));
    };

    const DragSource = ({ items }: { items: Haul.Item[] }): ReactElement => {
      const { startDrag } = Haul.useDrag({ type: "source", key: "source" });
      return <button data-testid="drag-source" onClick={() => startDrag(items)} />;
    };

    interface DragTabsProps {
      haulType?: string;
      canDrop?: Haul.CanDrop;
      onDrop?: (params: Tabs.SelectorOnDropParams) => Haul.Item[];
      items?: Haul.Item[];
    }

    const DragTabs = ({
      haulType = TAB_TYPE,
      canDrop,
      onDrop,
      items = [{ type: TAB_TYPE, key: "x" }],
    }: DragTabsProps): ReactElement => (
      <Haul.Provider>
        <DragSource items={items} />
        <Tabs.Frame initialValue="a">
          <Tabs.Selector haulType={haulType} canDrop={canDrop} onDrop={onDrop}>
            <Tabs.Tab itemKey="a">Tab A</Tabs.Tab>
            <Tabs.Tab itemKey="b">Tab B</Tabs.Tab>
            <Tabs.Tab itemKey="c">Tab C</Tabs.Tab>
          </Tabs.Selector>
        </Tabs.Frame>
      </Haul.Provider>
    );

    const beginDrag = (): void => {
      fireEvent.click(screen.getByTestId("drag-source"));
      stubStripRects(screen.getByRole("tablist"));
    };

    // jsdom has no DragEvent, so testing-library falls back to a plain Event and the
    // cursor coordinates in the init are lost. Define them on the event directly.
    const fireDragEvent = (type: "dragOver" | "drop", x: number): void => {
      const target = screen.getByRole("tablist");
      const event = createEvent[type](target);
      Object.defineProperties(event, {
        clientX: { value: x },
        clientY: { value: 16 },
        screenX: { value: x },
        screenY: { value: 16 },
      });
      fireEvent(target, event);
    };

    const indicator = (): HTMLElement | null =>
      document.querySelector(".pluto-tabs__insertion");

    it("should render an insertion indicator while dragging an accepted item", () => {
      render(<DragTabs />);
      expect(indicator()).toBeNull();
      beginDrag();
      fireDragEvent("dragOver", 150);
      expect(indicator()).not.toBeNull();
    });

    it("should report the resolved insertion index on drop", () => {
      const onDrop = vi.fn((_p: Tabs.SelectorOnDropParams): Haul.Item[] => []);
      render(<DragTabs onDrop={onDrop} />);
      beginDrag();
      fireDragEvent("drop", 90);
      expect(onDrop).toHaveBeenCalledTimes(1);
      expect(onDrop.mock.calls[0][0].index).toEqual(1);
    });

    it("should resolve the index from the cursor position across the strip", () => {
      const onDrop = vi.fn((_p: Tabs.SelectorOnDropParams): Haul.Item[] => []);
      render(<DragTabs onDrop={onDrop} />);
      // Tab centers sit at 50/150/250; a drop ends the drag, so restart it per slot.
      for (const x of [30, 90, 210, 400]) {
        beginDrag();
        fireDragEvent("drop", x);
      }
      expect(onDrop.mock.calls.map(([p]) => p.index)).toEqual([0, 1, 2, 3]);
    });

    it("should clear the indicator once a drop lands", () => {
      render(<DragTabs onDrop={() => []} />);
      beginDrag();
      fireDragEvent("dragOver", 150);
      expect(indicator()).not.toBeNull();
      fireDragEvent("drop", 150);
      expect(indicator()).toBeNull();
    });

    it("should clear the indicator when the drag leaves the strip", () => {
      render(<DragTabs />);
      beginDrag();
      fireDragEvent("dragOver", 150);
      expect(indicator()).not.toBeNull();
      fireEvent.dragLeave(screen.getByRole("tablist"));
      expect(indicator()).toBeNull();
    });

    it("should register no drop zone when haulType is empty", () => {
      const onDrop = vi.fn((_p: Tabs.SelectorOnDropParams): Haul.Item[] => []);
      render(<DragTabs haulType="" onDrop={onDrop} />);
      beginDrag();
      fireDragEvent("dragOver", 150);
      expect(indicator()).toBeNull();
      fireDragEvent("drop", 150);
      expect(onDrop).not.toHaveBeenCalled();
    });

    it("should reject items whose type does not match haulType", () => {
      const onDrop = vi.fn((_p: Tabs.SelectorOnDropParams): Haul.Item[] => []);
      render(<DragTabs onDrop={onDrop} items={[{ type: "other", key: "y" }]} />);
      beginDrag();
      fireDragEvent("dragOver", 150);
      expect(indicator()).toBeNull();
      fireDragEvent("drop", 150);
      expect(onDrop).not.toHaveBeenCalled();
    });

    it("should honor a custom canDrop predicate", () => {
      const onDrop = vi.fn((_p: Tabs.SelectorOnDropParams): Haul.Item[] => []);
      render(<DragTabs canDrop={() => false} onDrop={onDrop} />);
      beginDrag();
      fireDragEvent("dragOver", 150);
      expect(indicator()).toBeNull();
      fireDragEvent("drop", 150);
      expect(onDrop).not.toHaveBeenCalled();
    });
  });
});
