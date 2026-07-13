// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createEvent, fireEvent, render, screen } from "@testing-library/react";
import { type ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Haul } from "@/haul";
import { Mosaic } from "@/mosaic";
import { Tabs } from "@/tabs";
import { mockBoundingClientRect } from "@/testutil/dom";

const stubRect = (el: Element, x: number, y: number, w: number, h: number): void => {
  vi.spyOn(el, "getBoundingClientRect").mockReturnValue({
    x,
    y,
    width: w,
    height: h,
    top: y,
    bottom: y + h,
    left: x,
    right: x + w,
    toJSON: () => ({}),
  });
};

/**
 * Gives the leaf a 400x300 rect with a 32px tall strip across the top holding
 * 100px wide tabs, so cursor positions resolve to deterministic regions.
 */
const stubLeafRects = (leaf: HTMLElement): void => {
  stubRect(leaf, 0, 0, 400, 300);
  const tabList = leaf.querySelector('[role="tablist"]');
  if (tabList != null) stubRect(tabList, 0, 0, 400, 32);
  leaf
    .querySelectorAll("[data-tab-key]")
    .forEach((tab, i) => stubRect(tab, i * 100, 0, 100, 32));
};

interface DragSourceProps {
  items: Haul.Item[];
}

const DragSource = ({ items }: DragSourceProps): ReactElement => {
  const { startDrag } = Haul.useDrag({ type: "test-source", key: "source" });
  return <button data-testid="drag-source" onClick={() => startDrag(items)} />;
};

interface LeafTabsProps {
  nodeKey: number;
  tabs: string[];
}

const LeafTabs = ({ nodeKey, tabs }: LeafTabsProps): ReactElement => {
  const { startDrag, onDragEnd } = Mosaic.useDragTab();
  const selectorDropProps = Mosaic.useSelectorDropProps({ nodeKey, tabKeys: tabs });
  return (
    <Mosaic.Leaf nodeKey={nodeKey} data-testid={`leaf-${nodeKey}`}>
      <Tabs.Frame initialValue={tabs[0] ?? ""}>
        <Tabs.Selector {...selectorDropProps}>
          {tabs.map((tab) => (
            <Tabs.Tab
              key={tab}
              itemKey={tab}
              draggable
              onDragStart={(e) => startDrag(e, tab)}
              onDragEnd={onDragEnd}
            >
              Tab {tab}
            </Tabs.Tab>
          ))}
        </Tabs.Selector>
        {tabs.map((tab) => (
          <Tabs.Content key={tab} itemKey={tab}>
            <span>Content {tab}</span>
            <Mosaic.Shield />
          </Tabs.Content>
        ))}
      </Tabs.Frame>
    </Mosaic.Leaf>
  );
};

interface HarnessProps extends Omit<Mosaic.FrameProps, "children"> {
  tabs?: string[];
  items?: Haul.Item[];
}

const Harness = ({
  tabs = ["a", "b", "c"],
  items = [],
  ...rest
}: HarnessProps): ReactElement => (
  <Haul.Provider>
    <DragSource items={items} />
    <Mosaic.Frame {...rest}>
      <LeafTabs nodeKey={1} tabs={tabs} />
    </Mosaic.Frame>
  </Haul.Provider>
);

const leaf = (key: number = 1): HTMLElement => screen.getByTestId(`leaf-${key}`);

const strip = (key: number = 1): HTMLElement => {
  const el = leaf(key).querySelector<HTMLElement>('[role="tablist"]');
  if (el == null) throw new Error("tab strip not found");
  return el;
};

const beginDrag = (target: HTMLElement): void => {
  fireEvent.click(screen.getByTestId("drag-source"));
  stubLeafRects(target);
};

// jsdom has no DragEvent, so testing-library falls back to a plain Event for drag
// event types and the cursor coordinates in the init are lost. Define them on the
// event directly.
const fireDragEvent = (
  target: HTMLElement,
  type: "dragOver" | "drop",
  x: number,
  y: number,
): void => {
  const event = createEvent[type](target);
  Object.defineProperties(event, {
    clientX: { value: x },
    clientY: { value: y },
    screenX: { value: x },
    screenY: { value: y },
  });
  fireEvent(target, event);
};

const dragOver = (target: HTMLElement, x: number, y: number): void =>
  fireDragEvent(target, "dragOver", x, y);

const drop = (target: HTMLElement, x: number, y: number): void =>
  fireDragEvent(target, "drop", x, y);

const mask = (): HTMLElement | null => document.querySelector(".pluto-mosaic__mask");

const shield = (): HTMLElement | null =>
  document.querySelector(".pluto-mosaic__shield");

const insertionIndicator = (): HTMLElement | null =>
  document.querySelector(".pluto-tabs__insertion");

describe("Mosaic", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe("Leaf", () => {
    it("should throw when rendered outside a Frame", () => {
      expect(() => render(<Mosaic.Leaf nodeKey={1}>content</Mosaic.Leaf>)).toThrow(
        "Mosaic.Leaf must be used within Mosaic.Frame",
      );
    });

    it("should report a center drop of a hauled tab", () => {
      const onDrop = vi.fn();
      render(<Harness onDrop={onDrop} items={[Mosaic.createTabDropHaulItem("x")]} />);
      beginDrag(leaf());
      drop(leaf(), 200, 150);
      expect(onDrop).toHaveBeenCalledTimes(1);
      expect(onDrop).toHaveBeenCalledWith({
        nodeKey: 1,
        tabKey: "x",
        location: "center",
        index: undefined,
      });
    });

    it.each<[string, number, number]>([
      ["left", 40, 150],
      ["right", 360, 150],
      ["top", 200, 40],
      ["bottom", 200, 280],
    ])("should report a %s drop at the matching edge", (location, x, y) => {
      const onDrop = vi.fn();
      render(<Harness onDrop={onDrop} items={[Mosaic.createTabDropHaulItem("x")]} />);
      beginDrag(leaf());
      drop(leaf(), x, y);
      expect(onDrop).toHaveBeenCalledWith(expect.objectContaining({ location }));
    });

    it("should resolve strip drops to a center drop at the insertion index, claimed before the leaf multi-fires", () => {
      const onDrop = vi.fn();
      render(<Harness onDrop={onDrop} items={[Mosaic.createTabDropHaulItem("x")]} />);
      beginDrag(leaf());
      // The drop bubbles from the strip to the leaf, which is its own drop
      // target: a single onDrop call proves the strip claimed the event first.
      drop(strip(), 140, 16);
      expect(onDrop).toHaveBeenCalledTimes(1);
      expect(onDrop).toHaveBeenCalledWith({
        nodeKey: 1,
        tabKey: "x",
        location: "center",
        index: 1,
      });
    });

    it("should reject dropping a leaf's sole tab back onto its own strip", () => {
      const onDrop = vi.fn();
      render(
        <Harness
          tabs={["a"]}
          onDrop={onDrop}
          items={[Mosaic.createTabDropHaulItem("a")]}
        />,
      );
      beginDrag(leaf());
      drop(strip(), 50, 16);
      expect(onDrop).not.toHaveBeenCalled();
    });

    it("should resolve a drop past the last tab to the end of the strip", () => {
      const onDrop = vi.fn();
      render(<Harness onDrop={onDrop} items={[Mosaic.createTabDropHaulItem("x")]} />);
      beginDrag(leaf());
      drop(strip(), 380, 16);
      expect(onDrop).toHaveBeenCalledTimes(1);
      expect(onDrop).toHaveBeenCalledWith(expect.objectContaining({ index: 3 }));
    });

    it("should report creation drops on the strip with the insertion index", () => {
      const onCreate = vi.fn();
      render(
        <Harness
          onCreate={onCreate}
          items={[Mosaic.createTabCreateHaulItem("ontology:1")]}
        />,
      );
      beginDrag(leaf());
      drop(strip(), 140, 16);
      expect(onCreate).toHaveBeenCalledTimes(1);
      expect(onCreate).toHaveBeenCalledWith({
        nodeKey: 1,
        location: "center",
        tabKeys: ["ontology:1"],
        index: 1,
      });
    });

    it("should treat any drop on an empty leaf as a center drop", () => {
      const onDrop = vi.fn();
      render(
        <Harness
          tabs={[]}
          onDrop={onDrop}
          items={[Mosaic.createTabDropHaulItem("x")]}
        />,
      );
      beginDrag(leaf());
      drop(leaf(), 40, 150);
      expect(onDrop).toHaveBeenCalledWith(
        expect.objectContaining({ location: "center" }),
      );
    });

    it("should refuse a drop containing all of the leaf's own tabs", () => {
      const onDrop = vi.fn();
      render(
        <Harness
          tabs={["a"]}
          onDrop={onDrop}
          items={[Mosaic.createTabDropHaulItem("a")]}
        />,
      );
      beginDrag(leaf());
      drop(leaf(), 200, 150);
      expect(onDrop).not.toHaveBeenCalled();
    });

    it("should accept a drop of the leaf's own tab when others remain", () => {
      const onDrop = vi.fn();
      render(
        <Harness
          tabs={["a", "b"]}
          onDrop={onDrop}
          items={[Mosaic.createTabDropHaulItem("a")]}
        />,
      );
      beginDrag(leaf());
      drop(leaf(), 200, 150);
      expect(onDrop).toHaveBeenCalledWith(expect.objectContaining({ tabKey: "a" }));
    });

    it("should report creation drops with every created key", () => {
      const onCreate = vi.fn();
      render(
        <Harness
          onCreate={onCreate}
          items={[
            Mosaic.createTabCreateHaulItem("ontology:1"),
            Mosaic.createTabCreateHaulItem("ontology:2"),
          ]}
        />,
      );
      beginDrag(leaf());
      drop(leaf(), 40, 150);
      expect(onCreate).toHaveBeenCalledWith({
        nodeKey: 1,
        location: "left",
        tabKeys: ["ontology:1", "ontology:2"],
        index: undefined,
      });
    });

    it("should hand file drops to onFileDrop with the drop location", () => {
      const onFileDrop = vi.fn();
      render(<Harness onFileDrop={onFileDrop} items={[Haul.FILE]} />);
      beginDrag(leaf());
      drop(leaf(), 200, 150);
      expect(onFileDrop).toHaveBeenCalledWith(
        expect.objectContaining({ nodeKey: 1, location: "center" }),
      );
    });

    it("should ignore file drops when the frame has no onFileDrop", () => {
      const onDrop = vi.fn();
      const onCreate = vi.fn();
      render(<Harness onDrop={onDrop} onCreate={onCreate} items={[Haul.FILE]} />);
      beginDrag(leaf());
      drop(leaf(), 200, 150);
      expect(onDrop).not.toHaveBeenCalled();
      expect(onCreate).not.toHaveBeenCalled();
    });

    it("should mask the affected half while dragging over an edge", () => {
      render(<Harness onDrop={vi.fn()} items={[Mosaic.createTabDropHaulItem("x")]} />);
      beginDrag(leaf());
      dragOver(leaf(), 40, 150);
      const el = mask();
      expect(el).not.toBeNull();
      expect(el?.style.left).toEqual("0%");
      expect(el?.style.width).toEqual("50%");
      expect(el?.style.height).toEqual("100%");
    });

    it("should swap the mask for the strip's insertion indicator over the strip", () => {
      render(<Harness onDrop={vi.fn()} items={[Mosaic.createTabDropHaulItem("x")]} />);
      beginDrag(leaf());
      dragOver(leaf(), 40, 150);
      expect(mask()).not.toBeNull();
      // Crossing from the leaf body into the strip fires dragleave on the leaf;
      // jsdom does not synthesize it, so fire it by hand before the strip dragover.
      fireEvent.dragLeave(leaf(), { relatedTarget: strip() });
      dragOver(strip(), 150, 16);
      expect(mask()).toBeNull();
      expect(insertionIndicator()).not.toBeNull();
    });

    it("should not re-mask the leaf while dragging over the strip", () => {
      render(<Harness onDrop={vi.fn()} items={[Mosaic.createTabDropHaulItem("x")]} />);
      beginDrag(leaf());
      dragOver(strip(), 150, 16);
      expect(insertionIndicator()).not.toBeNull();
      expect(mask()).toBeNull();
    });

    it("should clear the mask when the drag leaves the leaf", () => {
      render(<Harness onDrop={vi.fn()} items={[Mosaic.createTabDropHaulItem("x")]} />);
      beginDrag(leaf());
      dragOver(leaf(), 40, 150);
      expect(mask()).not.toBeNull();
      fireEvent.dragLeave(leaf());
      expect(mask()).toBeNull();
    });

    it("should clear the mask and indicator once a drop lands", () => {
      render(<Harness onDrop={vi.fn()} items={[Mosaic.createTabDropHaulItem("x")]} />);
      beginDrag(leaf());
      dragOver(strip(), 150, 16);
      expect(insertionIndicator()).not.toBeNull();
      drop(strip(), 150, 16);
      expect(insertionIndicator()).toBeNull();
      expect(mask()).toBeNull();
    });

    it("should shield the leaf's content while a tab drag is in flight", () => {
      render(<Harness onDrop={vi.fn()} items={[Mosaic.createTabDropHaulItem("x")]} />);
      expect(shield()).toBeNull();
      beginDrag(leaf());
      expect(shield()).not.toBeNull();
      drop(leaf(), 200, 150);
      expect(shield()).toBeNull();
    });

    it("should not shield the leaf for unrelated drags", () => {
      render(<Harness items={[{ type: "unrelated", key: "u" }]} />);
      beginDrag(leaf());
      expect(shield()).toBeNull();
    });
  });

  describe("useDragTab", () => {
    it("should haul a tab dragged from one leaf onto another", () => {
      const onDrop = vi.fn();
      render(
        <Haul.Provider>
          <Mosaic.Frame onDrop={onDrop}>
            <LeafTabs nodeKey={1} tabs={["a", "b"]} />
            <LeafTabs nodeKey={2} tabs={["c"]} />
          </Mosaic.Frame>
        </Haul.Provider>,
      );
      fireEvent.dragStart(screen.getByRole("tab", { name: "Tab a" }));
      stubLeafRects(leaf(2));
      drop(leaf(2), 200, 150);
      expect(onDrop).toHaveBeenCalledWith({
        nodeKey: 2,
        tabKey: "a",
        location: "center",
        index: undefined,
      });
    });
  });

  describe("Split", () => {
    beforeEach(() => {
      vi.spyOn(Element.prototype, "getBoundingClientRect").mockImplementation(
        mockBoundingClientRect(0, 0, 1000, 1000),
      );
    });

    it("should render both children", () => {
      render(
        <Mosaic.Frame>
          <Mosaic.Split nodeKey={1}>
            <p>First</p>
            <p>Last</p>
          </Mosaic.Split>
        </Mosaic.Frame>,
      );
      expect(screen.getByText("First")).toBeTruthy();
      expect(screen.getByText("Last")).toBeTruthy();
    });

    it("should report the committed ratio to the frame's onResize", () => {
      const onResize = vi.fn();
      const c = render(
        <Mosaic.Frame onResize={onResize}>
          <Mosaic.Split nodeKey={2}>
            <p>First</p>
            <p>Last</p>
          </Mosaic.Split>
        </Mosaic.Frame>,
      );
      const handle = c.container.querySelector<HTMLElement>(".pluto-resize__handle");
      if (handle == null) throw new Error("resize handle not found");
      fireEvent.pointerDown(handle, {
        pointerId: 1,
        button: 0,
        isPrimary: true,
        clientX: 500,
        clientY: 0,
      });
      fireEvent.pointerMove(window, { pointerId: 1, clientX: 750, clientY: 0 });
      fireEvent.pointerUp(window, { pointerId: 1, clientX: 750, clientY: 0 });
      expect(onResize).toHaveBeenCalledTimes(1);
      const [splitKey, size] = onResize.mock.lastCall as [number, number];
      expect(splitKey).toEqual(2);
      expect(size).toBeCloseTo(0.75, 2);
    });
  });

  describe("Frame", () => {
    it("should render children inside a mosaic container", () => {
      render(
        <Mosaic.Frame>
          <p>child</p>
        </Mosaic.Frame>,
      );
      expect(document.querySelector(".pluto-mosaic")).not.toBeNull();
      expect(screen.getByText("child")).toBeTruthy();
    });
  });
});
