// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";
import { userEvent } from "vitest/browser";

import { Button } from "@/button";
import { Icon } from "@/icon";
import { List } from "@/list";
import { Menu } from "@/menu";
import { height, renderThemed, resolve } from "@/styletest";
import { Tree } from "@/tree";

const NODES: Tree.Node[] = [
  { key: "parent", children: [{ key: "child", children: [{ key: "grandchild" }] }] },
  { key: "sibling" },
];

const getItem = List.createGetItem(
  (key: string) => ({ key, name: key }),
  (keys: string[]) => keys.map((key) => ({ key, name: key })),
);

interface ComponentProps {
  showRules?: boolean;
  onDoubleClick?: (key: string) => void;
}

const Component = ({ showRules, onDoubleClick }: ComponentProps): ReactElement => {
  const props = Tree.use({ nodes: NODES, initialExpanded: ["parent", "child"] });
  return (
    <Tree.Tree {...props} getItem={getItem} showRules={showRules}>
      {({ key, ...rest }) => (
        <Tree.Item key={key} {...rest} onDoubleClick={() => onDoubleClick?.(key)}>
          {key}
        </Tree.Item>
      )}
    </Tree.Tree>
  );
};

const LEVEL_NODES: Tree.Node[] = [{ key: "p" }, { key: "small" }];

const LevelComponent = (): ReactElement => {
  const props = Tree.use({ nodes: LEVEL_NODES });
  return (
    <Tree.Tree {...props} getItem={getItem}>
      {({ key, ...rest }) => (
        <Tree.Item key={key} {...rest} level={key === "small" ? "small" : "p"}>
          <Icon.Add />
          {key}
        </Tree.Item>
      )}
    </Tree.Tree>
  );
};

const BUTTON_NODES: Tree.Node[] = [{ key: "row" }];

const ButtonComponent = (): ReactElement => {
  const props = Tree.use({ nodes: BUTTON_NODES });
  return (
    <>
      <Tree.Tree {...props} getItem={getItem}>
        {({ key, ...rest }) => (
          <Tree.Item key={key} {...rest}>
            <Icon.Add id="row-icon" />
            {key}
            <Button.Button size="tiny" variant="text">
              <Icon.Add id="nested-icon" />
            </Button.Button>
          </Tree.Item>
        )}
      </Tree.Tree>
      <Button.Button size="tiny" variant="text">
        <Icon.Add id="outside-icon" />
      </Button.Button>
    </>
  );
};

const renderTree = async (): Promise<(key: string) => DOMRect> => {
  const { getByText } = await renderThemed(<Component />);
  return (key) => {
    const row = getByText(key, { exact: true }).element();
    const text = Array.from(row.childNodes).find(
      (node) => node.nodeType === Node.TEXT_NODE,
    )!;
    const range = document.createRange();
    range.selectNode(text);
    return range.getBoundingClientRect();
  };
};

describe("Tree", () => {
  it("should indent a child's text past its parent's text", async () => {
    const rect = await renderTree();
    expect(rect("child").left).toBeGreaterThan(rect("parent").left);
  });

  it("should line up a leaf's text with its parent's text", async () => {
    const rect = await renderTree();
    expect(rect("grandchild").left).toBe(rect("child").left);
  });

  it("should stack rows at one pitch without gaps or overlaps", async () => {
    await renderTree();
    const rows = ["parent", "child", "grandchild", "sibling"].map((key) =>
      document.getElementById(key)!.getBoundingClientRect(),
    );
    for (let i = 1; i < rows.length; i++) {
      expect(rows[i].height).toBe(rows[0].height);
      expect(rows[i].top).toBe(rows[i - 1].bottom);
    }
  });

  it("should size an item's icons by its text level", async () => {
    await renderThemed(<LevelComponent />);
    const icon = (key: string): number =>
      height(document.getElementById(key)!.querySelector(".lyra-icon")!);
    expect(icon("small")).toBeLessThan(icon("p"));
    expect(`${icon("small")}px`).toBe(resolve("var(--pluto-small-size)", "font-size"));
  });

  it("should let a button in a row size its own icon", async () => {
    await renderThemed(<ButtonComponent />);
    const icon = (id: string): number => height(document.getElementById(id)!);
    expect(icon("nested-icon")).toBe(icon("outside-icon"));
    expect(icon("row-icon")).toBeGreaterThan(icon("nested-icon"));
  });

  it("should draw a rule under each ancestor's caret on hover", async () => {
    const { container } = await renderThemed(<Component showRules />);
    const row = (key: string): HTMLElement => document.getElementById(key)!;
    const caretCenter = (key: string): number => {
      const caret = row(key).querySelector(".lyra-tree__expansion-indicator")!;
      const { left, width } = caret.getBoundingClientRect();
      return left + width / 2;
    };
    const rules = (key: string): { start: number; end: number; opacity: string } => {
      const after = getComputedStyle(row(key), "::after");
      const start = row(key).getBoundingClientRect().left + parseFloat(after.left);
      return { start, end: start + parseFloat(after.width), opacity: after.opacity };
    };
    expect(rules("grandchild").opacity).toBe("0");
    await userEvent.hover(container.querySelector(".lyra-tree")!);
    const grandchild = rules("grandchild");
    expect(grandchild.opacity).toBe("1");
    expect(grandchild.start + 0.5).toBe(caretCenter("parent"));
    expect(grandchild.end).toBeGreaterThan(caretCenter("child"));
    expect(rules("child").end).toBeLessThan(caretCenter("child"));
  });
});

const MANY_NODES: Tree.Node[] = Array.from({ length: 200 }, (_, i) => ({
  key: `node-${i}`,
}));

const ManyComponent = ({ virtual }: { virtual?: boolean }): ReactElement => {
  const props = Tree.use({ nodes: MANY_NODES });
  return (
    <div style={{ height: 200 }}>
      <Tree.Tree {...props} getItem={getItem} virtual={virtual}>
        {({ key, ...rest }) => (
          <Tree.Item key={key} {...rest}>
            {key}
          </Tree.Item>
        )}
      </Tree.Tree>
    </div>
  );
};

interface MenuComponentProps {
  onChange: (key: string) => void;
}

const MenuComponent = ({ onChange }: MenuComponentProps): ReactElement => {
  const props = Tree.use({ nodes: NODES });
  const ctx = Menu.useContextMenu();
  return (
    <Menu.ContextMenu
      {...ctx}
      menu={({ keys }) => (
        <Menu.Menu onChange={onChange}>
          <Menu.Item itemKey="open">Open {keys.join(",")}</Menu.Item>
        </Menu.Menu>
      )}
    >
      <Tree.Tree {...props} getItem={getItem} onContextMenu={ctx.open}>
        {({ key, ...rest }) => (
          <Tree.Item key={key} {...rest}>
            {key}
          </Tree.Item>
        )}
      </Tree.Tree>
    </Menu.ContextMenu>
  );
};

describe("Tree keys", () => {
  const tree = (): HTMLElement => document.querySelector(".lyra-tree")!;
  const selected = (key: string): boolean =>
    document.getElementById(key)!.getAttribute("aria-selected") === "true";

  it("should move the selection with the arrow keys after a click", async () => {
    const { getByText } = await renderThemed(<Component />);
    await userEvent.click(getByText("sibling", { exact: true }));
    expect(document.activeElement).toBe(tree());
    expect(getComputedStyle(document.getElementById("sibling")!).outlineStyle).toBe(
      "none",
    );
    await userEvent.keyboard("{ArrowUp}");
    await expect.poll(() => selected("grandchild")).toBe(true);
    expect(selected("sibling")).toBe(false);
  });

  it("should bring the selection into view", async () => {
    await renderThemed(<ManyComponent />);
    await userEvent.click(document.getElementById("node-0")!);
    await userEvent.keyboard("{End}");
    await expect.poll(() => document.getElementById("node-199")).not.toBeNull();
    const last = document.getElementById("node-199")!;
    await expect
      .poll(() => last.getBoundingClientRect().bottom)
      .toBeLessThanOrEqual(tree().getBoundingClientRect().bottom);
    expect(selected("node-199")).toBe(true);
  });

  it("should bring the selection into view in a tree that mounts every item", async () => {
    await renderThemed(<ManyComponent virtual={false} />);
    await userEvent.click(document.getElementById("node-0")!);
    await userEvent.keyboard("{End}");
    await expect.poll(() => selected("node-199")).toBe(true);
    const last = document.getElementById("node-199")!;
    await expect
      .poll(() => last.getBoundingClientRect().bottom, { timeout: 5000 })
      .toBeLessThanOrEqual(tree().getBoundingClientRect().bottom);
  });

  it("should ring the tree when it takes keyboard focus with nothing selected", async () => {
    await renderThemed(<Component />);
    await userEvent.tab();
    const style = getComputedStyle(tree());
    expect(style.outlineStyle).toBe("solid");
    expect(style.outlineColor).toBe(resolve("var(--pluto-primary-z)"));
  });

  it("should ring the selection while the tree has keyboard focus", async () => {
    await renderThemed(<Component />);
    await userEvent.tab();
    expect(document.activeElement).toBe(tree());
    await userEvent.keyboard("{ArrowDown}");
    const item = getComputedStyle(document.getElementById("parent")!);
    await expect.poll(() => item.outlineStyle).toBe("solid");
    expect(item.outlineColor).toBe(resolve("var(--pluto-primary-z)"));
    expect(getComputedStyle(tree()).outlineStyle).toBe("none");
  });

  it("should open one context menu of the selection from either menu key", async () => {
    const onChange = vi.fn();
    await renderThemed(<MenuComponent onChange={onChange} />);
    await userEvent.click(document.getElementById("sibling")!);
    await userEvent.keyboard("{Shift>}{F10}{/Shift}");
    await expect.poll(() => document.querySelectorAll('[role="menu"]').length).toBe(1);
    const item = document.querySelector<HTMLElement>('[role="menuitem"]')!;
    expect(item.textContent).toBe("Open sibling");
    await userEvent.keyboard("{ArrowDown}");
    await expect.poll(() => document.activeElement).toBe(item);
    await userEvent.keyboard("{Enter}");
    await expect.poll(() => onChange.mock.calls.length).toBe(1);
    expect(document.querySelector('[role="menu"]')).toBeNull();
    expect(document.activeElement).toBe(tree());
    await userEvent.keyboard("{ContextMenu}");
    await expect.poll(() => document.querySelectorAll('[role="menu"]').length).toBe(1);
    expect(document.querySelector('[role="menuitem"]')?.textContent).toBe(
      "Open sibling",
    );
  });

  it("should open the selection on Enter as a double-click does", async () => {
    const onDoubleClick = vi.fn();
    const { getByText } = await renderThemed(
      <Component onDoubleClick={onDoubleClick} />,
    );
    await userEvent.click(getByText("sibling", { exact: true }));
    await userEvent.keyboard("{ArrowUp}");
    await expect.poll(() => selected("grandchild")).toBe(true);
    await userEvent.keyboard("{Enter}");
    expect(onDoubleClick).toHaveBeenCalledExactlyOnceWith("grandchild");
  });
});
