// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, render, renderHook, screen } from "@testing-library/react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { type Haul } from "@/haul";
import { List } from "@/list";
import { mockGeometry } from "@/testutil/dom";
import { type Node } from "@/tree/base";
import { Item } from "@/tree/Item";
import {
  canDropHaulItem,
  createHaulItem,
  filterHaulItems,
  HAUL_TYPE,
  isHaulItem,
  type ToggleOn,
  Tree,
  use,
} from "@/tree/Tree";

const KEY = "node-1";
const OTHER: Haul.Item = { type: "other_type", key: "other" };

describe("tree haul utilities", () => {
  describe("createHaulItem", () => {
    it("creates an item with the tree HAUL_TYPE", () => {
      expect(createHaulItem(KEY).type).toEqual(HAUL_TYPE);
    });

    it("creates an item with the provided key", () => {
      expect(createHaulItem(KEY).key).toEqual(KEY);
    });
  });

  describe("isHaulItem", () => {
    it("returns true for an item of the tree kind", () => {
      expect(isHaulItem(createHaulItem(KEY))).toBe(true);
    });

    it("returns false for an item of another kind", () => {
      expect(isHaulItem(OTHER)).toBe(false);
    });
  });

  describe("filterHaulItems", () => {
    it("keeps tree items and drops items of other kinds", () => {
      const item = createHaulItem(KEY);
      expect(filterHaulItems([item, OTHER])).toEqual([item]);
    });
  });

  describe("canDropHaulItem", () => {
    it("returns true when at least one item is a tree item", () => {
      expect(
        canDropHaulItem({ source: OTHER, items: [createHaulItem(KEY), OTHER] }),
      ).toBe(true);
    });

    it("returns false when no item is a tree item", () => {
      expect(canDropHaulItem({ source: OTHER, items: [OTHER] })).toBe(false);
    });
  });
});

describe("Tree", () => {
  beforeAll(() => mockGeometry(100, 100));

  const NODES: Node[] = Array.from({ length: 200 }, (_, i) => ({
    key: `node-${String(i).padStart(3, "0")}`,
  }));

  const getItem = List.createGetItem(
    (key: string) => ({ key, name: key }),
    (keys: string[]) => keys.map((key) => ({ key, name: key })),
  );

  interface ComponentProps {
    pinned?: string[];
  }

  const Component = ({ pinned }: ComponentProps) => {
    const props = use({ nodes: NODES });
    return (
      <Tree {...props} getItem={getItem} pinned={pinned}>
        {({ key, ...rest }) => (
          <Item key={key} {...rest}>
            {key}
          </Item>
        )}
      </Tree>
    );
  };

  const renderTree = () => render(<Component />);

  it("should window its rows by default", () => {
    const { container } = renderTree();
    const rows = container.querySelectorAll(".lyra-tree__item");
    expect(rows.length).toBeGreaterThan(0);
    expect(rows.length).toBeLessThan(NODES.length);
  });

  it("should space its rows by the tree's item height", () => {
    const { container } = renderTree();
    const rows = Array.from(
      container.querySelectorAll<HTMLElement>(".lyra-tree__item"),
    );
    rows.forEach((row, index) => expect(row.style.top).toBe(`${index * 27}px`));
  });

  it("should reserve scroll height for every node", () => {
    const { container } = renderTree();
    const virtualizer = container.querySelector<HTMLElement>(".lyra-list__virtualizer");
    expect(virtualizer?.style.minHeight).toBe(`${NODES.length * 27}px`);
  });

  it("should keep a pinned node mounted outside the window", () => {
    const last = NODES[NODES.length - 1].key;
    const { rerender } = render(<Component />);
    expect(screen.queryByText(last)).toBeNull();
    rerender(<Component pinned={[last]} />);
    expect(screen.getByText(last)).toBeTruthy();
    rerender(<Component />);
    expect(screen.queryByText(last)).toBeNull();
  });
});

describe("Tree toggling", () => {
  beforeAll(() => mockGeometry(100, 100));

  const NODES: Node[] = [
    { key: "parent", children: [{ key: "child" }] },
    { key: "second", children: [{ key: "second-child" }] },
  ];

  const getItem = List.createGetItem(
    (key: string) => ({ key, name: key }),
    (keys: string[]) => keys.map((key) => ({ key, name: key })),
  );

  const Component = ({ toggleOn }: { toggleOn?: ToggleOn }) => {
    const props = use({ nodes: NODES, toggleOn });
    return (
      <Tree {...props} getItem={getItem}>
        {({ key, ...rest }) => (
          <Item key={key} {...rest}>
            {key}
          </Item>
        )}
      </Tree>
    );
  };

  const row = (name: string) => screen.getByRole("treeitem", { name });
  const parentRow = () => row("parent");
  const caret = (name = "parent") =>
    row(name).querySelector(".lyra-tree__expansion-indicator") as Element;

  describe("on the row", () => {
    it("should select and expand a node when its row is clicked", () => {
      render(<Component />);
      fireEvent.click(parentRow());
      expect(parentRow().getAttribute("aria-selected")).toBe("true");
      expect(parentRow().getAttribute("aria-expanded")).toBe("true");
      expect(screen.getByText("child")).toBeTruthy();
    });
  });

  describe("on the caret", () => {
    it("should expand a node without selecting it when its caret is clicked", () => {
      render(<Component toggleOn="caret" />);
      fireEvent.click(caret());
      expect(parentRow().getAttribute("aria-expanded")).toBe("true");
      expect(parentRow().getAttribute("aria-selected")).toBe("false");
      expect(screen.getByText("child")).toBeTruthy();
    });

    it("should collapse an expanded node when its caret is clicked again", () => {
      render(<Component toggleOn="caret" />);
      fireEvent.click(caret());
      fireEvent.click(caret());
      expect(parentRow().getAttribute("aria-expanded")).toBe("false");
      expect(screen.queryByText("child")).toBeNull();
    });

    it("should toggle the clicked node after an expand above it moves it", () => {
      render(<Component toggleOn="caret" />);
      fireEvent.click(caret());
      fireEvent.click(caret("second"));
      expect(row("second").getAttribute("aria-expanded")).toBe("true");
      expect(screen.getByText("second-child")).toBeTruthy();
    });

    it("should toggle the clicked node after an expand below it", () => {
      render(<Component toggleOn="caret" />);
      fireEvent.click(caret("second"));
      fireEvent.click(caret());
      expect(parentRow().getAttribute("aria-expanded")).toBe("true");
      expect(screen.getByText("child")).toBeTruthy();
    });

    it("should select a node without expanding it when its row is clicked", () => {
      render(<Component toggleOn="caret" />);
      fireEvent.click(parentRow());
      expect(parentRow().getAttribute("aria-selected")).toBe("true");
      expect(parentRow().getAttribute("aria-expanded")).toBe("false");
      expect(screen.queryByText("child")).toBeNull();
    });
  });
});

describe("Tree expansion", () => {
  const NODES: Node[] = [
    { key: "a", children: [{ key: "a1" }] },
    { key: "b", children: [{ key: "b1" }] },
  ];

  it("should report the expanded keys after an expand", () => {
    const onExpand = vi.fn();
    const { result } = renderHook(() => use({ nodes: NODES, onExpand }));
    act(() => result.current.expand("a"));
    act(() => result.current.expand("b"));
    expect(onExpand).toHaveBeenLastCalledWith({
      current: ["a", "b"],
      action: "expand",
      clicked: "b",
    });
  });

  it("should report the expanded keys after a contract", () => {
    const onExpand = vi.fn();
    const { result } = renderHook(() =>
      use({ nodes: NODES, onExpand, initialExpanded: ["a", "b"] }),
    );
    act(() => result.current.contract("a"));
    expect(onExpand).toHaveBeenLastCalledWith({
      current: ["b"],
      action: "contract",
      clicked: "a",
    });
  });
});
