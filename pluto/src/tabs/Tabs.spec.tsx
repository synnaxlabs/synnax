// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render, screen } from "@testing-library/react";
import {
  type MouseEventHandler,
  type ReactElement,
  useEffect,
  useState,
} from "react";
import { describe, expect, it, vi } from "vitest";

import { Select } from "@/select/base";
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
    it("should move focus without selecting on arrow keys", () => {
      render(<BasicTabs initialValue="a" />);
      const a = tab("Tab A");
      a.focus();
      fireEvent.keyDown(a, { key: "ArrowRight" });
      expect(document.activeElement).toBe(tab("Tab B"));
      expect(screen.getByText("Content A")).toBeTruthy();
      expect(screen.queryByText("Content B")).toBeNull();
    });

    it("should select the focused tab on Enter", () => {
      render(<BasicTabs initialValue="a" />);
      const a = tab("Tab A");
      a.focus();
      fireEvent.keyDown(a, { key: "ArrowRight" });
      const b = tab("Tab B");
      fireEvent.keyDown(b, { key: "Enter" });
      expect(screen.getByText("Content B")).toBeTruthy();
    });

    it("should select the focused tab on Space", () => {
      render(<BasicTabs initialValue="a" />);
      const b = tab("Tab B");
      b.focus();
      fireEvent.keyDown(b, { key: " " });
      expect(screen.getByText("Content B")).toBeTruthy();
    });

    it("should wrap focus around the ends of the strip", () => {
      render(<BasicTabs initialValue="a" />);
      const c = tab("Tab C");
      c.focus();
      fireEvent.keyDown(c, { key: "ArrowRight" });
      expect(document.activeElement).toBe(tab("Tab A"));
      fireEvent.keyDown(tab("Tab A"), { key: "ArrowLeft" });
      expect(document.activeElement).toBe(c);
    });

    it("should jump to the first and last tabs on Home and End", () => {
      render(<BasicTabs initialValue="b" />);
      const b = tab("Tab B");
      b.focus();
      fireEvent.keyDown(b, { key: "End" });
      expect(document.activeElement).toBe(tab("Tab C"));
      fireEvent.keyDown(tab("Tab C"), { key: "Home" });
      expect(document.activeElement).toBe(tab("Tab A"));
    });

    it("should use vertical arrow keys when the selector is vertical", () => {
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
      fireEvent.keyDown(a, { key: "ArrowRight" });
      expect(document.activeElement).toBe(a);
      fireEvent.keyDown(a, { key: "ArrowDown" });
      expect(document.activeElement).toBe(tab("Tab B"));
    });

    it("should not rove focus when arrow keys fire inside a tab's children", () => {
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
      fireEvent.keyDown(child, { key: "ArrowRight" });
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
      const close = (key: string): MouseEventHandler => (e) => {
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
      const closeButtons = screen.getAllByLabelText("pluto-tabs__close");
      expect(closeButtons).toHaveLength(2);
      fireEvent.click(closeButtons[1]);
      expect(onClose).toHaveBeenCalledTimes(1);
      expect(onClose).toHaveBeenCalledWith("b");
    });

    it("should not select the tab whose close button is clicked", () => {
      const onClose = vi.fn();
      render(<ClosableTabs onClose={onClose} />);
      fireEvent.click(screen.getAllByLabelText("pluto-tabs__close")[1]);
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

  describe("Selector", () => {
    it("should render a drop indicator at the given insertion index", () => {
      const { container, rerender } = render(<BasicTabs initialValue="a" />);
      expect(container.querySelector(".pluto-tabs__insertion")).toBeNull();
      rerender(
        <Tabs.Frame initialValue="a">
          <Tabs.Selector insertionIndex={1}>
            <Tabs.Tab itemKey="a">Tab A</Tabs.Tab>
            <Tabs.Tab itemKey="b">Tab B</Tabs.Tab>
          </Tabs.Selector>
        </Tabs.Frame>,
      );
      expect(container.querySelector(".pluto-tabs__insertion")).toBeTruthy();
    });

    describe("getInsertionIndex", () => {
      const stubBounds = (el: Element, left: number, width: number): void => {
        vi.spyOn(el, "getBoundingClientRect").mockReturnValue({
          x: left,
          y: 0,
          width,
          height: 20,
          top: 0,
          bottom: 20,
          left,
          right: left + width,
          toJSON: () => ({}),
        });
      };

      it("should derive the index from the cursor position", () => {
        render(<BasicTabs initialValue="a" />);
        const tablist = screen.getByRole("tablist");
        const tabs = tablist.querySelectorAll("[data-tab-key]");
        stubBounds(tabs[0], 0, 100);
        stubBounds(tabs[1], 100, 100);
        stubBounds(tabs[2], 200, 100);
        expect(Tabs.getInsertionIndex(tablist, { x: 30, y: 10 })).toEqual(0);
        expect(Tabs.getInsertionIndex(tablist, { x: 90, y: 10 })).toEqual(1);
        expect(Tabs.getInsertionIndex(tablist, { x: 210, y: 10 })).toEqual(2);
        expect(Tabs.getInsertionIndex(tablist, { x: 400, y: 10 })).toEqual(3);
      });
    });
  });
});
