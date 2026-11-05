// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, render, renderHook, screen } from "@testing-library/react";
import { type KeyboardEvent, type MouseEvent } from "react";
import { beforeAll, describe, expect, it, onTestFinished, vi } from "vitest";

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
import { Triggers } from "@/triggers";

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

  it("should toggle a node that its collapsed parent hides", () => {
    const onExpand = vi.fn();
    const { result } = renderHook(() =>
      use({ nodes: [{ key: "a", children: [{ key: "a1", children: [] }] }], onExpand }),
    );
    act(() => result.current.toggle("a1"));
    expect(onExpand).toHaveBeenLastCalledWith({
      current: ["a1"],
      action: "expand",
      clicked: "a1",
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

describe("Tree keys", () => {
  beforeAll(() => mockGeometry(100, 100));

  const NODES: Node[] = [
    { key: "a", children: [{ key: "a1" }, { key: "a2" }] },
    { key: "b" },
    { key: "c" },
  ];

  const getItem = List.createGetItem(
    (key: string) => ({ key, name: key }),
    (keys: string[]) => keys.map((key) => ({ key, name: key })),
  );

  interface ComponentProps {
    nodes?: Node[];
    onKeyDown?: (e: KeyboardEvent<HTMLDivElement>) => void;
    onDoubleClick?: (key: string) => void;
    onContextMenu?: (e: MouseEvent<HTMLDivElement>) => void;
  }

  const Component = ({
    nodes = NODES,
    onKeyDown,
    onDoubleClick,
    onContextMenu,
  }: ComponentProps) => {
    const props = use({ nodes });
    return (
      <Triggers.Provider>
        <Tree
          {...props}
          getItem={getItem}
          onKeyDown={onKeyDown}
          onContextMenu={onContextMenu}
        >
          {({ key, ...rest }) => (
            <Item key={key} {...rest} onDoubleClick={() => onDoubleClick?.(key)}>
              {key}
              {key === "b" && <input aria-label="b name" />}
            </Item>
          )}
        </Tree>
      </Triggers.Provider>
    );
  };

  const tree = () => screen.getByRole("tree");
  const row = (name: string) => screen.getByRole("treeitem", { name });
  const press = (key: string, init?: KeyboardEventInit) => {
    fireEvent.keyDown(tree(), { key, code: key, ...init });
    fireEvent.keyUp(tree(), { key, code: key, ...init });
  };
  const selected = () =>
    screen
      .queryAllByRole("treeitem", { selected: true })
      .map(({ textContent }) => textContent);

  it("should take focus for its items", () => {
    render(<Component />);
    expect(tree().tabIndex).toBe(0);
  });

  it("should select the first item on the first ArrowDown", () => {
    render(<Component />);
    press("ArrowDown");
    expect(selected()).toEqual(["a"]);
  });

  it("should move the selection down and up", () => {
    render(<Component />);
    press("ArrowDown");
    press("ArrowDown");
    expect(selected()).toEqual(["b"]);
    press("ArrowUp");
    expect(selected()).toEqual(["a"]);
  });

  it("should move the selection to the ends on Home and End", () => {
    render(<Component />);
    press("End");
    expect(selected()).toEqual(["c"]);
    press("Home");
    expect(selected()).toEqual(["a"]);
  });

  it("should wrap around on a new press at either end", () => {
    render(<Component />);
    press("End");
    press("ArrowDown");
    expect(selected()).toEqual(["a"]);
    press("ArrowUp");
    expect(selected()).toEqual(["c"]);
  });

  it("should stop a held key at the last item", () => {
    vi.useFakeTimers();
    onTestFinished(() => {
      vi.useRealTimers();
    });
    render(<Component />);
    fireEvent.keyDown(tree(), { key: "ArrowDown", code: "ArrowDown" });
    act(() => {
      vi.advanceTimersByTime(400);
    });
    expect(selected()).toEqual(["c"]);
    act(() => {
      vi.advanceTimersByTime(200);
    });
    expect(selected()).toEqual(["c"]);
  });

  it("should double-click the selected item on Enter", () => {
    const onDoubleClick = vi.fn();
    render(<Component onDoubleClick={onDoubleClick} />);
    press("ArrowDown");
    press("ArrowDown");
    press("Enter");
    expect(onDoubleClick).toHaveBeenCalledExactlyOnceWith("b");
  });

  it("should not expand a node that the selection moves onto", () => {
    render(<Component />);
    press("ArrowDown");
    expect(row("a").getAttribute("aria-expanded")).toBe("false");
  });

  it("should expand a collapsed node on ArrowRight, then move into it", () => {
    render(<Component />);
    press("ArrowDown");
    press("ArrowRight");
    expect(row("a").getAttribute("aria-expanded")).toBe("true");
    expect(selected()).toEqual(["a"]);
    press("ArrowRight");
    expect(selected()).toEqual(["a1"]);
  });

  it("should collapse an expanded node on ArrowLeft", () => {
    render(<Component />);
    press("ArrowDown");
    press("ArrowRight");
    press("ArrowLeft");
    expect(row("a").getAttribute("aria-expanded")).toBe("false");
    expect(selected()).toEqual(["a"]);
  });

  it("should move to the parent on ArrowLeft", () => {
    render(<Component />);
    press("ArrowDown");
    press("ArrowRight");
    press("ArrowRight");
    press("ArrowDown");
    expect(selected()).toEqual(["a2"]);
    press("ArrowLeft");
    expect(selected()).toEqual(["a"]);
    expect(row("a").getAttribute("aria-expanded")).toBe("true");
  });

  it("should stop a held key when the window loses focus", () => {
    vi.useFakeTimers();
    onTestFinished(() => {
      vi.useRealTimers();
    });
    render(<Component />);
    fireEvent.keyDown(tree(), { key: "ArrowDown", code: "ArrowDown" });
    fireEvent.blur(window);
    act(() => {
      vi.advanceTimersByTime(1000);
    });
    expect(selected()).toEqual(["a"]);
  });

  it("should double-click a selected item on Enter after a click deselects", () => {
    const onDoubleClick = vi.fn();
    render(<Component onDoubleClick={onDoubleClick} />);
    fireEvent.click(row("b"));
    fireEvent.keyDown(window, { code: "Control" });
    fireEvent.click(row("c"));
    fireEvent.click(row("c"));
    fireEvent.keyUp(window, { code: "Control" });
    expect(selected()).toEqual(["b"]);
    press("Enter");
    expect(onDoubleClick).toHaveBeenCalledExactlyOnceWith("b");
  });

  it("should open the context menu of the selected item on Shift+F10", () => {
    const onContextMenu = vi.fn();
    render(<Component onContextMenu={onContextMenu} />);
    fireEvent.click(row("b"));
    const shiftF10 = { key: "F10", code: "F10", shiftKey: true };
    expect(fireEvent.keyDown(tree(), shiftF10)).toBe(false);
    expect(onContextMenu).toHaveBeenCalledOnce();
    expect(onContextMenu.mock.calls[0][0].target).toBe(row("b"));
  });

  it("should open the selected item's context menu on the Menu key release", () => {
    const onContextMenu = vi.fn();
    render(<Component onContextMenu={onContextMenu} />);
    fireEvent.click(row("b"));
    const menuKey = { key: "ContextMenu", code: "ContextMenu" };
    expect(fireEvent.keyDown(tree(), menuKey)).toBe(false);
    expect(onContextMenu).not.toHaveBeenCalled();
    expect(fireEvent.keyUp(tree(), menuKey)).toBe(false);
    expect(onContextMenu).toHaveBeenCalledOnce();
    expect(onContextMenu.mock.calls[0][0].target).toBe(row("b"));
  });

  it("should open no context menu from the keys with nothing selected", () => {
    const onContextMenu = vi.fn();
    render(<Component onContextMenu={onContextMenu} />);
    fireEvent.keyDown(tree(), { key: "F10", code: "F10", shiftKey: true });
    expect(onContextMenu).not.toHaveBeenCalled();
  });

  it("should move from the item clicked last", () => {
    render(<Component />);
    fireEvent.click(row("c"));
    press("ArrowUp");
    expect(selected()).toEqual(["b"]);
  });

  it("should move from the item a modifier click deselected", () => {
    render(
      <Component nodes={[{ key: "a" }, { key: "b" }, { key: "c" }, { key: "d" }]} />,
    );
    fireEvent.click(row("a"));
    fireEvent.keyDown(window, { code: "Control" });
    fireEvent.click(row("c"));
    fireEvent.click(row("c"));
    fireEvent.keyUp(window, { code: "Control" });
    expect(selected()).toEqual(["a"]);
    press("ArrowDown");
    expect(selected()).toEqual(["d"]);
  });

  it("should open the selection out of view after a modifier click deselects", () => {
    const onDoubleClick = vi.fn();
    const nodes = Array.from({ length: 300 }, (_, i) => ({ key: `${i}` }));
    const c = render(<Component nodes={nodes} onDoubleClick={onDoubleClick} />);
    fireEvent.click(row("0"));
    const scroller = c.container.querySelector<HTMLElement>(".lyra-list__scroll");
    if (scroller == null) throw new Error("scroll container not found");
    const scrollTo = (index: number) =>
      act(() => {
        scroller.scrollTop = index * 27;
        fireEvent.scroll(scroller);
      });
    scrollTo(200);
    fireEvent.keyDown(window, { code: "Control" });
    fireEvent.click(row("200"));
    fireEvent.click(row("200"));
    fireEvent.keyUp(window, { code: "Control" });
    scrollTo(201);
    expect(screen.queryByText("1", { exact: true })).toBeNull();
    press("Enter");
    expect(onDoubleClick).toHaveBeenCalledExactlyOnceWith("0");
  });

  it("should extend a shift-click range from the item the keys moved to", () => {
    render(
      <Component nodes={[{ key: "a" }, { key: "b" }, { key: "c" }, { key: "d" }]} />,
    );
    fireEvent.click(row("a"));
    press("ArrowDown");
    press("ArrowDown");
    fireEvent.keyDown(window, { code: "Shift" });
    fireEvent.click(row("d"), { shiftKey: true });
    expect(selected()).toEqual(["c", "d"]);
  });

  it("should leave a key pressed with a modifier alone", () => {
    render(<Component />);
    press("ArrowDown");
    press("ArrowDown", { shiftKey: true });
    press("ArrowDown", { metaKey: true });
    expect(selected()).toEqual(["a"]);
  });

  it("should leave a key typed into an element inside an item alone", () => {
    render(<Component />);
    fireEvent.keyDown(screen.getByLabelText("b name"), {
      key: "ArrowDown",
      code: "ArrowDown",
    });
    expect(selected()).toEqual([]);
  });

  it("should leave a key alone when the given handler prevents its default", () => {
    const onKeyDown = vi.fn((e: KeyboardEvent<HTMLDivElement>) => e.preventDefault());
    render(<Component onKeyDown={onKeyDown} />);
    press("ArrowDown");
    expect(onKeyDown).toHaveBeenCalledOnce();
    expect(selected()).toEqual([]);
  });
});
