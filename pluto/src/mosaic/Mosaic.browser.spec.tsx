// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Haul } from "@synnaxlabs/lyra/haul";
import { computed, renderThemed, resolve } from "@synnaxlabs/lyra/styletest";
import { Tabs } from "@synnaxlabs/lyra/tabs";
import { fireDragEvent } from "@synnaxlabs/lyra/testutil";
import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { CSS } from "@/css";
import { Mosaic } from "@/mosaic";

const WIDTH = 800;
const HEIGHT = 500;

const Leaf = ({ nodeKey }: { nodeKey: number }): ReactElement => (
  <Mosaic.Leaf nodeKey={nodeKey} data-testid={`leaf-${nodeKey}`}>
    <Tabs.Frame initialValue="a" data-testid={`tabs-${nodeKey}`}>
      <Tabs.Selector>
        <Tabs.Tab itemKey="a">Tab a</Tabs.Tab>
      </Tabs.Selector>
      <Tabs.Content itemKey="a">Content {nodeKey}</Tabs.Content>
    </Tabs.Frame>
  </Mosaic.Leaf>
);

const renderMosaic = async (): Promise<Element> => {
  const { container } = await renderThemed(
    <div style={{ width: WIDTH, height: HEIGHT }}>
      <Haul.Provider>
        <Mosaic.Frame>
          <Mosaic.Split nodeKey={1}>
            <Leaf nodeKey={2} />
            <Leaf nodeKey={3} />
          </Mosaic.Split>
        </Mosaic.Frame>
      </Haul.Provider>
    </div>,
  );
  const frame = container.querySelector(`.${CSS.B("mosaic")}`);
  if (frame == null) throw new Error("mosaic frame not found");
  return frame;
};

const leaf = (key: number): Element => {
  const el = document.querySelector(`[data-testid="leaf-${key}"]`);
  if (el == null) throw new Error(`leaf ${key} not found`);
  return el;
};

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
  const tabList = leaf.querySelector(Tabs.LIST_SELECTOR);
  if (tabList != null) stubRect(tabList, 0, 0, 400, 32);
  leaf
    .querySelectorAll("[data-tab-key]")
    .forEach((tab, i) => stubRect(tab, i * 100, 0, 100, 32));
};

const DRAG = { dataTransfer: new DataTransfer() };

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

describe("Mosaic", () => {
  it("should fill its container", async () => {
    const rect = (await renderMosaic()).getBoundingClientRect();
    expect(rect.width).toBe(WIDTH);
    expect(rect.height).toBe(HEIGHT);
  });

  it("should lay out as a block, not a flex box", async () => {
    expect(computed(await renderMosaic(), "display")).toBe("block");
  });

  it("should lift off the page with the lift shadow", async () => {
    expect(computed(await renderMosaic(), "box-shadow")).toBe(
      resolve("var(--pluto-shadow-lift)", "box-shadow"),
    );
  });

  it("should split its width between two full height leaves", async () => {
    const frame = (await renderMosaic()).getBoundingClientRect();
    const first = leaf(2).getBoundingClientRect();
    const second = leaf(3).getBoundingClientRect();
    expect(first.left).toBe(frame.left);
    expect(second.right).toBeCloseTo(frame.right, 0);
    expect(second.left).toBeGreaterThanOrEqual(first.right);
    for (const pane of [first, second]) {
      expect(pane.height).toBe(HEIGHT);
      expect(pane.width).toBeCloseTo(WIDTH / 2, -1);
    }
  });

  it("should fill each leaf with its tabs", async () => {
    await renderMosaic();
    const tabs = leaf(2).querySelector('[data-testid="tabs-2"]');
    if (tabs == null) throw new Error("tabs not found");
    const outer = leaf(2).getBoundingClientRect();
    const inner = tabs.getBoundingClientRect();
    expect(inner.width).toBe(outer.width);
    expect(inner.height).toBe(outer.height);
  });

  describe("useDragTab", () => {
    it("should picture a dragged tab without its close button", () => {
      const ClosableTab = (): ReactElement => {
        const { startDrag, onDragEnd } = Mosaic.useDragTab();
        return (
          <Tabs.Frame initialValue="a">
            <Tabs.Selector>
              <Tabs.Tab
                itemKey="a"
                draggable
                onDragStart={(e) => startDrag(e, "a")}
                onDragEnd={onDragEnd}
              >
                Tab a
                <Tabs.Close />
              </Tabs.Tab>
            </Tabs.Selector>
          </Tabs.Frame>
        );
      };
      render(
        <Haul.Provider>
          <ClosableTab />
        </Haul.Provider>,
      );
      fireEvent.dragStart(screen.getByRole("tab", { name: /Tab a/ }), DRAG);
      const layer = document.querySelector(".lyra-haul__layer");
      expect(layer?.textContent).toBe("Tab a");
      expect(layer?.querySelector('[aria-label="Close"]')).toBeNull();
      expect(screen.getByRole("button", { name: "Close" })).toBeDefined();
    });

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
      fireEvent.dragStart(screen.getByRole("tab", { name: "Tab a" }), DRAG);
      const target = screen.getByTestId("leaf-2");
      stubLeafRects(target);
      fireDragEvent(target, "drop", { x: 200, y: 150 });
      expect(onDrop).toHaveBeenCalledWith({
        nodeKey: 2,
        tabKey: "a",
        location: "center",
        index: undefined,
      });
    });
  });
});
