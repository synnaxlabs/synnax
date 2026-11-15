// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, Suspense, use } from "react";
import { describe, expect, it } from "vitest";
import { page, userEvent } from "vitest/browser";

import { Menu } from "@/menu";
import { computed, renderThemed, resolve } from "@/styletest";

const ITEMS = (
  <Menu.Menu value="copy">
    <Menu.Divider />
    <Menu.Item itemKey="copy">Copy</Menu.Item>
    <Menu.Divider />
    <Menu.Divider />
    <Menu.Item itemKey="rename">Rename</Menu.Item>
    <Menu.Divider />
  </Menu.Menu>
);

const ContextMenu = (): ReactElement => {
  const ctx = Menu.useContextMenu();
  return (
    <Menu.ContextMenu menu={() => ITEMS} {...ctx}>
      <div className={Menu.CONTEXT_TARGET} onContextMenu={ctx.open}>
        Target
      </div>
    </Menu.ContextMenu>
  );
};

const FocusMenu = (): ReactElement => {
  const ctx = Menu.useContextMenu();
  return (
    <Menu.ContextMenu menu={() => ITEMS} {...ctx}>
      <button onContextMenu={ctx.open}>Opener</button>
      {/* Takes no focus on a press, as a canvas does. */}
      <div onMouseDown={(e) => e.preventDefault()}>Canvas</div>
    </Menu.ContextMenu>
  );
};

const Loaded = ({ items }: { items: Promise<void> }): ReactElement => {
  use(items);
  return ITEMS;
};

// Renders nothing while its items load, as a menu that reads Core data does.
const LoadingMenu = ({ items }: { items: Promise<void> }): ReactElement => {
  const ctx = Menu.useContextMenu();
  return (
    <Menu.ContextMenu
      menu={() => (
        <Suspense fallback={null}>
          <Loaded items={items} />
        </Suspense>
      )}
      {...ctx}
    >
      <button onContextMenu={ctx.open}>Opener</button>
      <input id="field" aria-label="field" />
    </Menu.ContextMenu>
  );
};

describe("Menu", () => {
  it("should fill a selected item with the list selection tier", async () => {
    const { getByRole } = await renderThemed(ITEMS);
    const item = getByRole("menuitem", { name: "Copy" }).element();
    expect(computed(item, "background-color")).toBe(
      resolve("var(--pluto-selected-list-bg)"),
    );
    expect(computed(item, "color")).toBe(resolve("var(--pluto-text-color)"));
  });

  it("should leave an unselected item transparent", async () => {
    const { getByRole } = await renderThemed(ITEMS);
    const item = getByRole("menuitem", { name: "Rename" }).element();
    expect(computed(item, "background-color")).toBe(resolve("transparent"));
  });

  it("should show only the divider between two items", async () => {
    const { container } = await renderThemed(ITEMS);
    const shown = Array.from(container.children).filter(
      (el) =>
        el.getAttribute("role") !== "menuitem" && computed(el, "display") !== "none",
    );
    expect(shown).toHaveLength(1);
    expect(shown[0].previousElementSibling?.textContent).toBe("Copy");
  });

  it("should stretch items to the width of the menu", async () => {
    const { getByRole } = await renderThemed(<div style={{ width: 300 }}>{ITEMS}</div>);
    const item = getByRole("menuitem", { name: "Rename" }).element();
    expect(item.getBoundingClientRect().width).toBe(300);
  });

  describe("context menu", () => {
    it("should float over the page on the overlay surface", async () => {
      const { getByText } = await renderThemed(<ContextMenu />);
      await userEvent.click(getByText("Target"), { button: "right" });
      const menu = page.getByRole("menu").element();
      expect(computed(menu, "position")).toBe("fixed");
      expect(computed(menu, "background-color")).toBe(
        resolve("var(--pluto-gray-l1-l2)"),
      );
      expect(computed(menu, "box-shadow")).toBe(
        resolve("var(--pluto-shadow-overlay)", "box-shadow"),
      );
    });

    it("should fill a focused item as a hovered item, with no ring", async () => {
      const { getByText } = await renderThemed(<ContextMenu />);
      await userEvent.click(getByText("Target"), { button: "right" });
      await userEvent.keyboard("{ArrowDown}{ArrowDown}");
      const rename = page.getByRole("menuitem", { name: "Rename" }).element();
      await expect.poll(() => document.activeElement).toBe(rename);
      expect(computed(rename, "background-color")).toBe(
        resolve(computed(rename, "--pluto-hover-bg")),
      );
      expect(computed(rename, "background-color")).not.toBe(resolve("transparent"));
      expect(computed(rename, "outline-style")).toBe("none");
    });

    const openLoading = async () => {
      let load = (): void => {};
      const items = new Promise<void>((resolve) => {
        load = resolve;
      });
      const { getByText } = await renderThemed(<LoadingMenu items={items} />);
      const opener = getByText("Opener").element() as HTMLElement;
      opener.focus();
      await userEvent.click(opener, { button: "right" });
      return { opener, load };
    };

    it("should take the focus when its items load", async () => {
      const { load } = await openLoading();
      load();
      const copy = page.getByRole("menuitem", { name: "Copy" });
      await expect
        .poll(() => document.activeElement?.getAttribute("role"))
        .toBe("menu");
      await userEvent.keyboard("{ArrowDown}");
      await expect.poll(() => document.activeElement).toBe(copy.element());
    });

    it("should leave the focus that moved while its items load", async () => {
      const { load } = await openLoading();
      const field = document.getElementById("field") as HTMLElement;
      field.focus();
      load();
      await expect.poll(() => page.getByRole("menu").query()).not.toBeNull();
      expect(document.activeElement).toBe(field);
    });

    it("should leave the focus to a click outside", async () => {
      const { getByText } = await renderThemed(<FocusMenu />);
      const opener = getByText("Opener").element() as HTMLElement;
      opener.focus();
      await userEvent.click(opener, { button: "right" });
      await expect
        .poll(() => document.activeElement?.getAttribute("role"))
        .toBe("menu");
      await userEvent.click(getByText("Canvas"));
      await expect.poll(() => document.querySelector('[role="menu"]')).toBeNull();
      expect(document.activeElement).not.toBe(opener);
    });
  });
});
