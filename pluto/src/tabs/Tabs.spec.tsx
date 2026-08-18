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
import {
  type CSSProperties,
  type MouseEventHandler,
  type ReactElement,
  useEffect,
  useState,
} from "react";
import {
  afterEach,
  assert,
  beforeEach,
  describe,
  expect,
  it,
  type MockInstance,
  vi,
} from "vitest";

import { Haul } from "@/haul";
import { Select } from "@/select";
import { Tabs } from "@/tabs";
import { fireDragEvent } from "@/testutil/dom";

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

// jsdom lays nothing out, so the scrollport geometry is stubbed per element.
const stubStrip = (scrollWidth: number, clientWidth: number): HTMLElement => {
  const strip = screen.getByRole("tablist");
  let scrollLeft = 0;
  Object.defineProperties(strip, {
    scrollWidth: { value: scrollWidth, configurable: true },
    clientWidth: { value: clientWidth, configurable: true },
    scrollLeft: {
      get: () => scrollLeft,
      set: (v: number) => (scrollLeft = v),
      configurable: true,
    },
  });
  return strip;
};

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

    it("should select a tab that does not head an ordered selection", () => {
      const onSelect = vi.fn();
      render(
        <Select.Context value={["b", "a"]} onSelect={onSelect}>
          <BasicTabs />
        </Select.Context>,
      );
      fireEvent.click(tab("Tab A"));
      expect(onSelect).toHaveBeenCalledWith("a");
    });

    it("should not select the tab heading an ordered selection", () => {
      const onSelect = vi.fn();
      render(
        <Select.Context value={["b", "a"]} onSelect={onSelect}>
          <BasicTabs />
        </Select.Context>,
      );
      fireEvent.click(tab("Tab B"));
      expect(onSelect).not.toHaveBeenCalled();
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

  describe("style knobs", () => {
    const strip = (props: Tabs.SelectorProps = {}): HTMLElement => {
      render(
        <Tabs.Frame initialValue="a">
          <Tabs.Selector {...props}>
            <Tabs.Tab itemKey="a">Tab A</Tabs.Tab>
          </Tabs.Selector>
        </Tabs.Frame>,
      );
      return screen.getByRole("tablist");
    };

    it.each([
      ["default", "align-center", "sizing-elastic"],
      ["pill", "align-start", "sizing-fixed"],
    ] as const)("should default %s to %s and %s", (variant, align, sizing) => {
      const el = strip({ variant });
      expect(el.classList.contains(`pluto-tabs__selector--${align}`)).toBe(true);
      expect(el.classList.contains(`pluto-tabs__selector--${sizing}`)).toBe(true);
    });

    it.each(["elastic", "fixed", "content"] as const)(
      "should apply the %s sizing",
      (sizing) => {
        const el = strip({ sizing });
        expect(el.classList.contains(`pluto-tabs__selector--sizing-${sizing}`)).toBe(
          true,
        );
      },
    );

    it("should let explicit knobs override the variant defaults", () => {
      const el = strip({ variant: "pill", align: "center", sizing: "elastic" });
      expect(el.classList.contains("pluto-tabs__selector--align-center")).toBe(true);
      expect(el.classList.contains("pluto-tabs__selector--sizing-elastic")).toBe(true);
    });

    it("should start-align a vertical strip whose variant centers when horizontal", () => {
      const el = strip({ variant: "default", y: true });
      expect(el.classList.contains("pluto-tabs__selector--align-start")).toBe(true);
    });

    it("should still honor an explicit align on a vertical strip", () => {
      const el = strip({ variant: "default", y: true, align: "center" });
      expect(el.classList.contains("pluto-tabs__selector--align-center")).toBe(true);
    });
  });

  describe("scroll thumb", () => {
    // The thumb renders as the strip's sibling so it can hang below the scrollport.
    const thumb = (strip: HTMLElement): HTMLElement => {
      const el = strip.parentElement?.querySelector<HTMLElement>(".pluto-tabs__thumb");
      assert(el != null);
      return el;
    };

    it("should mark an overflowing strip scrollable", () => {
      render(<BasicTabs initialValue="a" />);
      const strip = stubStrip(500, 200);
      fireEvent.scroll(strip);
      expect(strip.classList.contains("pluto-tabs__selector--scrollable")).toBe(true);
    });

    it("should not mark a strip whose tabs fit", () => {
      render(<BasicTabs initialValue="a" />);
      const strip = stubStrip(200, 200);
      fireEvent.scroll(strip);
      expect(strip.classList.contains("pluto-tabs__selector--scrollable")).toBe(false);
    });

    it("should mark only the edges with tabs hidden past them", () => {
      render(<BasicTabs initialValue="a" />);
      const strip = stubStrip(500, 200);
      const clipped = (edge: string): boolean =>
        strip.classList.contains(`pluto-tabs__selector--clipped-${edge}`);
      fireEvent.scroll(strip);
      expect(clipped("start")).toBe(false);
      expect(clipped("end")).toBe(true);
      strip.scrollLeft = 150;
      fireEvent.scroll(strip);
      expect(clipped("start")).toBe(true);
      expect(clipped("end")).toBe(true);
      strip.scrollLeft = 300;
      fireEvent.scroll(strip);
      expect(clipped("start")).toBe(true);
      expect(clipped("end")).toBe(false);
    });

    it("should size and place the thumb from the scroll geometry", () => {
      render(<BasicTabs initialValue="a" />);
      const strip = stubStrip(500, 200);
      strip.scrollLeft = 150;
      fireEvent.scroll(strip);
      // Width covers the visible fraction: 200 / 500 * 200. The offset walks the
      // free track with scroll progress: (150 / 300) * (200 - 80).
      expect(thumb(strip).style.width).toEqual("80px");
      expect(thumb(strip).style.transform).toEqual("translateX(60px)");
    });

    it("should clamp the thumb to its minimum width", () => {
      render(<BasicTabs initialValue="a" />);
      const strip = stubStrip(4000, 100);
      fireEvent.scroll(strip);
      expect(thumb(strip).style.width).toEqual("24px");
    });

    it("should scroll the strip when the thumb is dragged", () => {
      render(<BasicTabs initialValue="a" />);
      const strip = stubStrip(500, 200);
      fireEvent.scroll(strip);
      const el = thumb(strip);
      el.setPointerCapture = vi.fn();
      Object.defineProperty(el, "offsetWidth", { value: 80, configurable: true });
      fireEvent.pointerDown(el, { pointerId: 1, clientX: 0 });
      fireEvent.pointerMove(el, { pointerId: 1, clientX: 40 });
      // 40px over the 120px of free track maps to 100px of the 300px scroll range.
      expect(strip.scrollLeft).toEqual(100);
      fireEvent.pointerUp(el, { pointerId: 1 });
      fireEvent.pointerMove(el, { pointerId: 1, clientX: 80 });
      expect(strip.scrollLeft).toEqual(100);
    });
  });

  describe("selected tab visibility", () => {
    let scrollLeft: MockInstance<(offset: number) => void>;

    // jsdom lays nothing out and ignores scrollLeft writes, so every tab takes a 100px
    // slot in a 100px port and the write is read back off the setter.
    beforeEach(() => {
      vi.spyOn(HTMLElement.prototype, "offsetWidth", "get").mockReturnValue(100);
      vi.spyOn(Element.prototype, "clientWidth", "get").mockReturnValue(100);
      vi.spyOn(HTMLElement.prototype, "offsetLeft", "get").mockImplementation(function (
        this: HTMLElement,
      ): number {
        return Array.from(this.parentElement?.children ?? []).indexOf(this) * 100;
      });
      scrollLeft = vi.spyOn(Element.prototype, "scrollLeft", "set");
    });

    afterEach(() => {
      vi.restoreAllMocks();
    });

    // The strip each write scrolled, in order.
    const scrolled = (): Element[] => scrollLeft.mock.contexts as Element[];

    // The offsets written, in order.
    const offsets = (): number[] => scrollLeft.mock.calls.map(([offset]) => offset);

    // Two strips sharing one selection, the shape Panel.Mosaic renders.
    const SplitTabs = ({ value }: { value: string }): ReactElement => (
      <Select.Context value={value}>
        <Tabs.Frame>
          <Tabs.Selector>
            <Tabs.Tab itemKey="a">Tab A</Tabs.Tab>
          </Tabs.Selector>
        </Tabs.Frame>
        <Tabs.Frame>
          <Tabs.Selector>
            <Tabs.Tab itemKey="b">Tab B</Tabs.Tab>
          </Tabs.Selector>
        </Tabs.Frame>
      </Select.Context>
    );

    it("should scroll the selected tab into view when the selection changes", () => {
      const { rerender } = render(<BasicTabs value="a" onChange={vi.fn()} />);
      scrollLeft.mockClear();
      rerender(<BasicTabs value="c" onChange={vi.fn()} />);
      expect(scrolled()).toEqual([screen.getByRole("tablist")]);
      expect(offsets()).toEqual([200]);
    });

    it("should scroll the already selected tab into view on mount", () => {
      render(<BasicTabs initialValue="b" />);
      expect(scrolled()).toEqual([screen.getByRole("tablist")]);
      expect(offsets()).toEqual([100]);
    });

    it("should not scroll when the selected tab belongs to another strip", () => {
      const { rerender } = render(<SplitTabs value="a" />);
      scrollLeft.mockClear();
      rerender(<SplitTabs value="b" />);
      expect(scrolled()).toEqual([screen.getAllByRole("tablist")[1]]);
    });

    // Two strips each holding a selection, the shape a split Panel.Mosaic renders.
    const SplitSelection = ({ value }: { value: string[] }): ReactElement => (
      <Select.Context value={value}>
        <Tabs.Frame>
          <Tabs.Selector>
            <Tabs.Tab itemKey="a1">Tab A1</Tabs.Tab>
            <Tabs.Tab itemKey="a2">Tab A2</Tabs.Tab>
          </Tabs.Selector>
        </Tabs.Frame>
        <Tabs.Frame>
          <Tabs.Selector>
            <Tabs.Tab itemKey="b1">Tab B1</Tabs.Tab>
            <Tabs.Tab itemKey="b2">Tab B2</Tabs.Tab>
          </Tabs.Selector>
        </Tabs.Frame>
      </Select.Context>
    );

    it("should leave a strip alone when a sibling's selection changes", () => {
      const { rerender } = render(<SplitSelection value={["a1", "b1"]} />);
      scrollLeft.mockClear();
      rerender(<SplitSelection value={["a2", "b1"]} />);
      expect(scrolled()).toEqual([screen.getAllByRole("tablist")[0]]);
      expect(offsets()).toEqual([100]);
    });
  });

  describe("wheel scrolling", () => {
    const wheel = (strip: HTMLElement, init: WheelEventInit): Event => {
      const event = createEvent.wheel(strip, { cancelable: true, ...init });
      fireEvent(strip, event);
      return event;
    };

    it("should scroll an overflowing strip right on a downward wheel", () => {
      render(<BasicTabs initialValue="a" />);
      const strip = stubStrip(500, 200);
      const event = wheel(strip, { deltaY: 100 });
      expect(strip.scrollLeft).toEqual(100);
      expect(event.defaultPrevented).toBe(true);
    });

    it("should scroll left on an upward wheel", () => {
      render(<BasicTabs initialValue="a" />);
      const strip = stubStrip(500, 200);
      strip.scrollLeft = 150;
      wheel(strip, { deltaY: -100 });
      expect(strip.scrollLeft).toEqual(50);
    });

    it("should not scroll a strip that does not overflow", () => {
      render(<BasicTabs initialValue="a" />);
      const strip = stubStrip(200, 200);
      const event = wheel(strip, { deltaY: 100 });
      expect(strip.scrollLeft).toEqual(0);
      expect(event.defaultPrevented).toBe(false);
    });

    it("should ignore a wheel already travelling horizontally", () => {
      render(<BasicTabs initialValue="a" />);
      const strip = stubStrip(500, 200);
      const event = wheel(strip, { deltaX: 100, deltaY: 20 });
      expect(strip.scrollLeft).toEqual(0);
      expect(event.defaultPrevented).toBe(false);
    });

    it("should clamp at the end and let the wheel chain to an ancestor", () => {
      render(<BasicTabs initialValue="a" />);
      const strip = stubStrip(500, 200);
      const clamped = wheel(strip, { deltaY: 1000 });
      expect(strip.scrollLeft).toEqual(300);
      expect(clamped.defaultPrevented).toBe(true);
      const atLimit = wheel(strip, { deltaY: 100 });
      expect(strip.scrollLeft).toEqual(300);
      expect(atLimit.defaultPrevented).toBe(false);
    });

    it("should scale a line-mode wheel into pixels", () => {
      render(<BasicTabs initialValue="a" />);
      const strip = stubStrip(500, 200);
      wheel(strip, { deltaY: 3, deltaMode: WheelEvent.DOM_DELTA_LINE });
      expect(strip.scrollLeft).toEqual(48);
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
      const { startDrag, onDragEnd } = Haul.useDrag({ type: "source", key: "source" });
      return (
        <button
          data-testid="drag-source"
          onClick={() => startDrag(items)}
          onDragEnd={onDragEnd}
        />
      );
    };

    interface DragTabsProps {
      haulType?: string;
      canDrop?: Haul.CanDrop;
      onDrop?: (params: Tabs.SelectorOnDropParams) => Haul.Item[];
      items?: Haul.Item[];
      // jsdom loads no stylesheet, so a strip needing a real gap sets one inline.
      style?: CSSProperties;
    }

    const DragTabs = ({
      haulType = TAB_TYPE,
      canDrop,
      onDrop,
      items = [{ type: TAB_TYPE, key: "x" }],
      style,
    }: DragTabsProps): ReactElement => (
      <Haul.Provider>
        <DragSource items={items} />
        <Tabs.Frame initialValue="a">
          <Tabs.Selector
            haulType={haulType}
            canDrop={canDrop}
            onDrop={onDrop}
            style={style}
          >
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

    const fireStripDrag = (type: "dragOver" | "drop", x: number): void =>
      fireDragEvent(screen.getByRole("tablist"), type, { x, y: 16 });

    const ghost = (): HTMLElement | null =>
      document.querySelector(".pluto-tabs__ghost");

    it("should render a ghost slot while dragging an accepted item", () => {
      render(<DragTabs />);
      expect(ghost()).toBeNull();
      beginDrag();
      fireStripDrag("dragOver", 150);
      expect(ghost()).not.toBeNull();
    });

    // An empty strip has no tab to measure the slot from, so it falls back to its own
    // leading padding and to the standard width the stylesheet gives the ghost.
    it("should render the ghost at the leading padding of an empty strip", () => {
      render(
        <Haul.Provider>
          <DragSource items={[{ type: TAB_TYPE, key: "x" }]} />
          <Tabs.Frame initialValue="a">
            <Tabs.Selector
              haulType={TAB_TYPE}
              onDrop={() => []}
              style={{ paddingLeft: 12 }}
            />
          </Tabs.Frame>
        </Haul.Provider>,
      );
      fireEvent.click(screen.getByTestId("drag-source"));
      fireStripDrag("dragOver", 150);
      expect(ghost()?.style.left).toEqual("12px");
      expect(ghost()?.style.width).toEqual("");
    });

    it("should report the resolved insertion index on drop", () => {
      const onDrop = vi.fn((_p: Tabs.SelectorOnDropParams): Haul.Item[] => []);
      render(<DragTabs onDrop={onDrop} />);
      beginDrag();
      fireStripDrag("drop", 90);
      expect(onDrop).toHaveBeenCalledTimes(1);
      expect(onDrop.mock.calls[0][0].index).toEqual(1);
    });

    it("should resolve the index from the cursor position across the strip", () => {
      const onDrop = vi.fn((_p: Tabs.SelectorOnDropParams): Haul.Item[] => []);
      render(<DragTabs onDrop={onDrop} />);
      // Tab centers sit at 50/150/250; a drop ends the drag, so restart it per slot.
      for (const x of [30, 90, 210, 400]) {
        beginDrag();
        fireStripDrag("drop", x);
      }
      expect(onDrop.mock.calls.map(([p]) => p.index)).toEqual([0, 1, 2, 3]);
    });

    it("should clear the ghost once a drop lands", () => {
      render(<DragTabs onDrop={() => []} />);
      beginDrag();
      fireStripDrag("dragOver", 150);
      expect(ghost()).not.toBeNull();
      fireStripDrag("drop", 150);
      expect(ghost()).toBeNull();
    });

    it("should clear the ghost when the drag leaves the strip", () => {
      render(<DragTabs />);
      beginDrag();
      fireStripDrag("dragOver", 150);
      expect(ghost()).not.toBeNull();
      fireEvent.dragLeave(screen.getByRole("tablist"));
      expect(ghost()).toBeNull();
    });

    it("should register no drop zone when haulType is empty", () => {
      const onDrop = vi.fn((_p: Tabs.SelectorOnDropParams): Haul.Item[] => []);
      render(<DragTabs haulType="" onDrop={onDrop} />);
      beginDrag();
      fireStripDrag("dragOver", 150);
      expect(ghost()).toBeNull();
      fireStripDrag("drop", 150);
      expect(onDrop).not.toHaveBeenCalled();
    });

    it("should reject items whose type does not match haulType", () => {
      const onDrop = vi.fn((_p: Tabs.SelectorOnDropParams): Haul.Item[] => []);
      render(<DragTabs onDrop={onDrop} items={[{ type: "other", key: "y" }]} />);
      beginDrag();
      fireStripDrag("dragOver", 150);
      expect(ghost()).toBeNull();
      fireStripDrag("drop", 150);
      expect(onDrop).not.toHaveBeenCalled();
    });

    it("should honor a custom canDrop predicate", () => {
      const onDrop = vi.fn((_p: Tabs.SelectorOnDropParams): Haul.Item[] => []);
      render(<DragTabs canDrop={() => false} onDrop={onDrop} />);
      beginDrag();
      fireStripDrag("dragOver", 150);
      expect(ghost()).toBeNull();
      fireStripDrag("drop", 150);
      expect(onDrop).not.toHaveBeenCalled();
    });

    describe("reorder preview", () => {
      const HAULED = "pluto-tabs__tab--hauled";

      const tabByKey = (key: string): HTMLElement =>
        screen
          .getByRole("tablist")
          .querySelector<HTMLElement>(`[data-tab-key="${key}"]`)!;

      // jsdom reports every offset as 0, so the shift distance would collapse and every
      // slot would resolve to the strip's start; lay the tabs out 100px wide from 0,
      // matching the stubbed strip rects.
      const beginReorder = (): void => {
        beginDrag();
        screen
          .getByRole("tablist")
          .querySelectorAll<HTMLElement>("[data-tab-key]")
          .forEach((t, i) => {
            Object.defineProperty(t, "offsetWidth", { configurable: true, value: 100 });
            Object.defineProperty(t, "offsetLeft", {
              configurable: true,
              value: i * 100,
            });
          });
      };

      const fireDragLeave = (relatedTarget: Node | null): void => {
        const target = screen.getByRole("tablist");
        const event = createEvent.dragLeave(target);
        Object.defineProperty(event, "relatedTarget", { value: relatedTarget });
        fireEvent(target, event);
      };

      const draggingTab = (key: string): Haul.Item[] => [{ type: TAB_TYPE, key }];

      it("should slide passed tabs aside and lift the source dragging right", () => {
        render(<DragTabs items={draggingTab("a")} onDrop={() => []} />);
        beginReorder();
        fireStripDrag("dragOver", 210);
        expect(tabByKey("a").classList.contains(HAULED)).toBe(true);
        expect(tabByKey("b").style.transform).toBe("translateX(-100px)");
        expect(tabByKey("c").style.transform).toBe("");
        // b slid back over a's slot, opening the one between b and c.
        expect(ghost()?.style.left).toEqual("100px");
      });

      it("should slide passed tabs aside dragging left", () => {
        render(<DragTabs items={draggingTab("c")} onDrop={() => []} />);
        beginReorder();
        fireStripDrag("dragOver", 30);
        expect(tabByKey("c").classList.contains(HAULED)).toBe(true);
        expect(tabByKey("a").style.transform).toBe("translateX(100px)");
        expect(tabByKey("b").style.transform).toBe("translateX(100px)");
        expect(ghost()?.style.left).toEqual("0px");
      });

      it("should not shift any tab while the drag hovers its own slot", () => {
        render(<DragTabs items={draggingTab("b")} onDrop={() => []} />);
        beginReorder();
        fireStripDrag("dragOver", 110);
        expect(tabByKey("a").style.transform).toBe("");
        expect(tabByKey("b").style.transform).toBe("");
        expect(tabByKey("c").style.transform).toBe("");
        // The slot the drag would land in is the one it came from.
        expect(ghost()?.style.left).toEqual("100px");
      });

      // A tab leaving the flow frees the gap that followed it too, so the tabs it
      // passes slide by both and the ghost keeps even gaps on either side.
      it("should slide the passed tabs by the strip's gap as well", () => {
        render(
          <DragTabs
            items={draggingTab("a")}
            style={{ columnGap: 10 }}
            onDrop={() => []}
          />,
        );
        beginReorder();
        fireStripDrag("dragOver", 210);
        expect(tabByKey("b").style.transform).toBe("translateX(-110px)");
        expect(ghost()?.style.left).toEqual("100px");
      });

      it("should leave a gap before a ghost landing past the last tab", () => {
        render(
          <DragTabs
            items={draggingTab("x")}
            style={{ columnGap: 10 }}
            onDrop={() => []}
          />,
        );
        beginReorder();
        fireStripDrag("dragOver", 400);
        expect(ghost()?.style.left).toEqual("310px");
      });

      it("should open a ghost slot for an item hauled in from elsewhere", () => {
        render(
          <DragTabs
            items={draggingTab("x")}
            style={{ columnGap: 10 }}
            onDrop={() => []}
          />,
        );
        beginReorder();
        fireStripDrag("dragOver", 150);
        expect(ghost()?.style.width).toEqual("100px");
        expect(ghost()?.style.left).toEqual("200px");
        expect(tabByKey("a").style.transform).toBe("");
        expect(tabByKey("b").style.transform).toBe("");
        expect(tabByKey("c").style.transform).toBe("translateX(110px)");
        expect(document.querySelector(`.${HAULED}`)).toBeNull();
      });

      it("should keep the preview when the cursor crosses onto a child tab", () => {
        render(<DragTabs items={draggingTab("a")} onDrop={() => []} />);
        beginReorder();
        fireStripDrag("dragOver", 210);
        fireDragLeave(tabByKey("c"));
        expect(tabByKey("b").style.transform).toBe("translateX(-100px)");
      });

      it("should reset the preview when the cursor truly leaves the strip", () => {
        render(<DragTabs items={draggingTab("a")} onDrop={() => []} />);
        beginReorder();
        fireStripDrag("dragOver", 210);
        fireDragLeave(document.body);
        expect(tabByKey("b").style.transform).toBe("");
        expect(document.querySelector(`.${HAULED}`)).toBeNull();
      });

      it("should clear the preview once a drop commits", () => {
        render(<DragTabs items={draggingTab("a")} onDrop={() => []} />);
        beginReorder();
        fireStripDrag("dragOver", 210);
        expect(tabByKey("b").style.transform).toBe("translateX(-100px)");
        fireStripDrag("drop", 210);
        expect(tabByKey("b").style.transform).toBe("");
        expect(document.querySelector(`.${HAULED}`)).toBeNull();
      });

      it("should reset the preview when the drag ends without a drop", () => {
        render(<DragTabs items={draggingTab("a")} />);
        beginReorder();
        fireStripDrag("dragOver", 210);
        expect(tabByKey("b").style.transform).toBe("translateX(-100px)");
        fireEvent.dragEnd(screen.getByTestId("drag-source"));
        expect(tabByKey("b").style.transform).toBe("");
        expect(document.querySelector(`.${HAULED}`)).toBeNull();
      });

      const SCROLLABLE = "pluto-tabs__selector--scrollable";

      const shiftOf = (tab: HTMLElement): number =>
        Number(/translateX\((-?[\d.]+)px\)/.exec(tab.style.transform)?.[1] ?? 0);

      // jsdom lays nothing out, so the strip reports no width of its own. Stand in
      // for the browser's scrollable overflow rule: the tabs are an exact fit, and
      // one translated toward the end pushes the scrollable width past the strip.
      const stubStripMetrics = (): void => {
        const strip = screen.getByRole("tablist");
        const tabs = (): HTMLElement[] =>
          Array.from(strip.querySelectorAll<HTMLElement>("[data-tab-key]"));
        Object.defineProperty(strip, "clientWidth", {
          configurable: true,
          get: () => tabs().length * 100,
        });
        Object.defineProperty(strip, "scrollWidth", {
          configurable: true,
          get: () => tabs().length * 100 + Math.max(0, ...tabs().map(shiftOf)),
        });
      };

      // Dragging left shifts the passed tabs toward the end, so the strip measures
      // as scrollable while the preview holds. Measuring before the reset leaves a
      // strip that fits wearing the fade and the scroll thumb.
      it("should report no overflow once a leftward reorder drops", () => {
        render(<DragTabs items={draggingTab("c")} onDrop={() => []} />);
        beginReorder();
        stubStripMetrics();
        fireStripDrag("dragOver", 30);
        fireStripDrag("drop", 30);
        expect(screen.getByRole("tablist").classList.contains(SCROLLABLE)).toBe(false);
      });

      it("should report no overflow once a leftward reorder is abandoned", () => {
        render(<DragTabs items={draggingTab("c")} />);
        beginReorder();
        stubStripMetrics();
        fireStripDrag("dragOver", 30);
        fireEvent.dragEnd(screen.getByTestId("drag-source"));
        expect(screen.getByRole("tablist").classList.contains(SCROLLABLE)).toBe(false);
      });
    });
  });
});
