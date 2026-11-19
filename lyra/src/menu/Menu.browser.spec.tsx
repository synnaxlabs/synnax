// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, Suspense, use } from "react";
import { describe, expect, it, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { Menu } from "@/menu";
import { Select } from "@/select";
import { computed, renderThemed, resolve } from "@/styletest";
import { Triggers } from "@/triggers";

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

const ChooseMenu = ({
  onChoose,
}: {
  onChoose: (focused: Element | null) => void;
}): ReactElement => {
  const ctx = Menu.useContextMenu();
  return (
    <Menu.ContextMenu
      menu={() => (
        <Menu.Menu>
          <Menu.Item itemKey="edit" onClick={() => onChoose(document.activeElement)}>
            Edit
          </Menu.Item>
        </Menu.Menu>
      )}
      {...ctx}
    >
      <button onContextMenu={ctx.open}>Opener</button>
    </Menu.ContextMenu>
  );
};

// A right click in WebKit moves the focus to the body, so the menu opens as the menu
// keys open it: on the focused element, with no press.
const openFromKeyboard = (element: Element): HTMLElement => {
  if (!(element instanceof HTMLElement)) throw new Error("not an HTML element");
  element.focus();
  element.dispatchEvent(
    new MouseEvent("contextmenu", { bubbles: true, cancelable: true }),
  );
  return element;
};

interface TargetMenuProps {
  onOpen: (focused: Element | null) => void;
  onChoose: (focused: Element | null) => void;
}

const TargetMenu = ({ onOpen, onChoose }: TargetMenuProps): ReactElement => {
  const ctx = Menu.useContextMenu();
  return (
    <Menu.ContextMenu
      menu={() => (
        <Menu.Menu>
          <Menu.Item itemKey="edit" onClick={() => onChoose(document.activeElement)}>
            Edit
          </Menu.Item>
        </Menu.Menu>
      )}
      {...ctx}
    >
      <button
        className={Menu.CONTEXT_TARGET}
        onContextMenu={(e) => {
          onOpen(document.activeElement);
          ctx.open(e);
        }}
      >
        Target
      </button>
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

// Fills the window, so a click at any point of the window opens the menu there.
const AreaMenu = ({ count }: { count: number }): ReactElement => {
  const ctx = Menu.useContextMenu();
  return (
    <Menu.ContextMenu
      menu={() => (
        <Menu.Menu>
          {Array.from({ length: count }, (_, i) => (
            <Menu.Item key={i} itemKey={`item-${i}`}>
              {`Item ${i}`}
            </Menu.Item>
          ))}
        </Menu.Menu>
      )}
      {...ctx}
    >
      <div
        style={{ position: "fixed", inset: 0 }}
        onContextMenu={ctx.open}
        onMouseDown={(e) => e.preventDefault()}
      >
        Area
      </div>
    </Menu.ContextMenu>
  );
};

// A menu item and a submenu whose options show in the menu search.
const SearchMenu = (): ReactElement => {
  const ctx = Menu.useContextMenu();
  return (
    <Menu.ContextMenu
      menu={() => (
        <Menu.Menu>
          <Menu.Item itemKey="report">Leak report</Menu.Item>
          <Menu.Divider />
          <Menu.Sub itemKey="labels" textValue="Labels">
            <Menu.SubTrigger>Labels</Menu.SubTrigger>
            <Select.Frame<string, undefined>
              multiple
              value={[]}
              onChange={() => {}}
              data={["Leak check"]}
            >
              <Select.Dialog surface="menu" filtered aria-label="Labels">
                <Select.Body<string> resourceName="label" onSearch={() => {}}>
                  {({ key, ...p }) => (
                    <Select.Item<string> key={key} {...p}>
                      {p.itemKey}
                    </Select.Item>
                  )}
                </Select.Body>
              </Select.Dialog>
            </Select.Frame>
          </Menu.Sub>
          <Menu.Divider />
          <Menu.Item itemKey="rename">Rename</Menu.Item>
        </Menu.Menu>
      )}
      {...ctx}
    >
      <div className={Menu.CONTEXT_TARGET} onContextMenu={ctx.open}>
        Target
      </div>
    </Menu.ContextMenu>
  );
};

const menuBox = (): DOMRect | undefined =>
  document.querySelector('[role="menu"]')?.getBoundingClientRect();

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

    it("should size and light an option in the search as a menu item", async () => {
      const { getByText } = await renderThemed(<SearchMenu />);
      await userEvent.click(getByText("Target"), { button: "right" });
      await userEvent.keyboard("leak");
      const item = page.getByRole("menuitem", { name: "Leak report" }).element();
      const option = page.getByRole("menuitem", { name: /Leak check/ }).element();
      for (const property of [
        "height",
        "padding-left",
        "padding-right",
        "border-radius",
      ])
        expect(computed(option, property)).toBe(computed(item, property));
      const lit = computed(item, "background-color");
      expect(lit).not.toBe(resolve("transparent"));
      await userEvent.hover(option);
      await expect.poll(() => computed(option, "background-color")).toBe(lit);
      expect(computed(item, "background-color")).toBe(resolve("transparent"));
    });

    it("should hide the dividers around a selection with no option that matches", async () => {
      const { getByText } = await renderThemed(<SearchMenu />);
      await userEvent.click(getByText("Target"), { button: "right" });
      // The selection's search never answers, so its stale option stays listed.
      await userEvent.keyboard("rena");
      await expect
        .element(page.getByRole("menuitem", { name: "Rename" }))
        .toBeVisible();
      const shown = Array.from(document.querySelectorAll(".lyra-menu__divider")).filter(
        (el) => computed(el, "display") !== "none",
      );
      expect(shown).toEqual([]);
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

    it("should keep the top of a menu taller than the window in the window", async () => {
      const count = Math.ceil((window.innerHeight * 1.3) / 28);
      const { getByText } = await renderThemed(<AreaMenu count={count} />);
      const y = window.innerHeight / 2;
      await userEvent.click(getByText("Area"), {
        button: "right",
        position: { x: 20, y },
      });
      await expect
        .poll(() => menuBox()?.height ?? 0)
        .toBeGreaterThan(window.innerHeight);
      expect(menuBox()?.top).toBe(0);
    });

    it("should fit the window again when it opens a second time at one spot", async () => {
      // About 60% of the window high, so it fits only above a cursor near the bottom.
      const count = Math.ceil((window.innerHeight * 0.6) / 28);
      const { getByText } = await renderThemed(<AreaMenu count={count} />);
      const area = getByText("Area");
      const spot = { x: 20, y: window.innerHeight * 0.8 };
      const outside = { x: window.innerWidth - 10, y: 10 };
      for (let i = 0; i < 2; i++) {
        await userEvent.click(area, { button: "right", position: spot });
        await expect.poll(() => menuBox()?.height ?? 0).toBeGreaterThan(0);
        await expect
          .poll(() => menuBox()?.bottom)
          .toBeLessThanOrEqual(window.innerHeight);
        await userEvent.click(area, { position: outside });
        await expect.poll(() => menuBox()).toBeUndefined();
      }
    });

    it("should leave the focus to a click outside", async () => {
      const { getByText } = await renderThemed(<FocusMenu />);
      const opener = getByText("Opener").element() as HTMLElement;
      opener.focus();
      await userEvent.click(opener, { button: "right" });
      await expect
        .poll(() => document.activeElement?.getAttribute("role"))
        .toBe("menu");
      // The menu covers the left of the canvas.
      const canvas = getByText("Canvas");
      const { width } = canvas.element().getBoundingClientRect();
      await userEvent.click(canvas, { position: { x: width - 4, y: 4 } });
      await expect.poll(() => document.querySelector('[role="menu"]')).toBeNull();
      expect(document.activeElement).not.toBe(opener);
    });

    const rightClick = async (onChoose = vi.fn()) => {
      const onOpen = vi.fn();
      const { getByText } = await renderThemed(
        <Triggers.Provider>
          <TargetMenu onOpen={onOpen} onChoose={onChoose} />
        </Triggers.Provider>,
      );
      const target = getByText("Target").element();
      await userEvent.click(target, { button: "right" });
      await expect
        .poll(() => document.activeElement?.getAttribute("role"))
        .toBe("menu");
      return { target, found: onOpen.mock.calls[0][0], getByText };
    };

    it("should focus the right-clicked target before an item acts", async () => {
      const onChoose = vi.fn();
      const { target, getByText } = await rightClick(onChoose);
      await userEvent.click(getByText("Edit"));
      await expect.poll(() => document.querySelector('[role="menu"]')).toBeNull();
      expect(onChoose).toHaveBeenCalledExactlyOnceWith(target);
      expect(document.activeElement).toBe(target);
      expect(target.matches(":focus-visible")).toBe(false);
    });

    it("should keep the focus of a right click on Escape, with no ring", async () => {
      const { found } = await rightClick();
      await userEvent.keyboard("{Escape}");
      await expect.poll(() => document.querySelector('[role="menu"]')).toBeNull();
      expect(document.activeElement).toBe(found);
      expect(document.activeElement?.matches(":focus-visible")).toBe(false);
    });

    it("should give the focus back before a clicked item acts", async () => {
      const onChoose = vi.fn();
      const { getByText } = await renderThemed(<ChooseMenu onChoose={onChoose} />);
      const opener = openFromKeyboard(getByText("Opener").element());
      await userEvent.click(getByText("Edit"));
      await expect.poll(() => document.querySelector('[role="menu"]')).toBeNull();
      expect(onChoose).toHaveBeenCalledExactlyOnceWith(opener);
    });

    it("should give the focus back before an item chosen with Enter acts", async () => {
      const onChoose = vi.fn();
      const { getByText } = await renderThemed(<ChooseMenu onChoose={onChoose} />);
      const opener = openFromKeyboard(getByText("Opener").element());
      await userEvent.keyboard("{ArrowDown}");
      await expect.poll(() => document.activeElement?.textContent).toBe("Edit");
      await userEvent.keyboard("{Enter}");
      await expect.poll(() => document.querySelector('[role="menu"]')).toBeNull();
      expect(onChoose).toHaveBeenCalledExactlyOnceWith(opener);
    });
  });
});
