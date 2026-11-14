// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { Dialog } from "@/dialog";
import { Menu } from "@/menu";
import { Select } from "@/select";
import { computed, renderThemed, resolve } from "@/styletest";
import { Triggers } from "@/triggers";

const Labels = (): ReactElement => (
  <Menu.Sub itemKey="labels">
    <Menu.SubTrigger>Labels</Menu.SubTrigger>
    <Select.Frame<string, undefined>
      multiple
      data={["alpha", "beta"]}
      value={[]}
      onChange={() => {}}
    >
      <Select.Dialog surface="menu" aria-label="Labels">
        <Select.Body<string> resourceName="label" onSearch={() => {}}>
          {({ key, ...p }) => (
            <Select.Item key={key} {...p}>
              {key}
            </Select.Item>
          )}
        </Select.Body>
      </Select.Dialog>
    </Select.Frame>
  </Menu.Sub>
);

const items = (onChange: (key: string) => void): ReactElement => (
  <Menu.Menu onChange={onChange}>
    <Menu.Item itemKey="copy">Copy</Menu.Item>
    <Menu.Divider />
    <Menu.Item itemKey="rename">Rename</Menu.Item>
    <Menu.Item itemKey="remove">Remove</Menu.Item>
    <Menu.Divider />
    <Menu.Item itemKey="favorite">Favorite</Menu.Item>
    <Labels />
  </Menu.Menu>
);

const ContextMenu = ({
  onChange,
}: {
  onChange: (key: string) => void;
}): ReactElement => {
  const ctx = Menu.useContextMenu();
  return (
    <Menu.ContextMenu menu={() => items(onChange)} {...ctx}>
      <button onContextMenu={ctx.open}>Opener</button>
    </Menu.ContextMenu>
  );
};

const openContextMenu = async (onChange = vi.fn()): Promise<HTMLElement> => {
  const { getByText } = await renderThemed(
    <Triggers.Provider>
      <ContextMenu onChange={onChange} />
    </Triggers.Provider>,
  );
  await userEvent.click(getByText("Opener"), { button: "right" });
  const menu = page.getByRole("menu").element() as HTMLElement;
  await expect.poll(() => document.activeElement).toBe(menu);
  return menu;
};

const displays = (selector: string): string[] =>
  Array.from(document.querySelectorAll(selector)).map((el) => computed(el, "display"));

// A filtered menu narrows to its items, but never below the menu floor.
const expectFloor = (menu: Element): void => {
  expect(computed(menu, "min-width")).toBe(resolve("24rem", "min-width"));
  expect(menu.getBoundingClientRect().width).toBe(
    parseFloat(computed(menu, "min-width")),
  );
};

const empty = (): Element | null => document.querySelector(".lyra-menu-filter__empty");

describe("menu filter", () => {
  beforeEach(async () => {
    await page.viewport(800, 600);
  });

  describe("context menu", () => {
    it("should show the filter as a line that spans the top of the menu", async () => {
      const menu = await openContextMenu();
      await userEvent.keyboard("ren");
      const line = document.querySelector(".lyra-menu-filter") as HTMLElement;
      await expect.poll(() => line?.textContent).toBe("ren");
      const lineBox = line.getBoundingClientRect();
      const menuBox = menu.getBoundingClientRect();
      expect(menuBox.right - lineBox.right).toBe(lineBox.left - menuBox.left);
      expect(lineBox.top - menuBox.top).toBe(1);
      expect(computed(line, "border-bottom-width")).toBe("1px");
    });

    it("should narrow no further than the menu floor as items hide", async () => {
      const menu = await openContextMenu();
      await userEvent.keyboard("ren");
      await expect.poll(() => page.getByRole("menuitem").elements().length).toBe(1);
      expectFloor(menu);
      await userEvent.keyboard("zz");
      await expect.poll(() => page.getByRole("menuitem").elements().length).toBe(0);
      expectFloor(menu);
    });

    it("should keep the dividers between groups that both match", async () => {
      await openContextMenu();
      await userEvent.keyboard("o");
      await expect.poll(() => page.getByRole("menuitem").elements().length).toBe(3);
      expect(displays(".lyra-menu__divider")).toEqual(["flex", "flex"]);
      expect(computed(empty() as Element, "display")).toBe("none");
    });

    it("should hide a divider with no match on one side", async () => {
      await openContextMenu();
      await userEvent.keyboard("ren");
      await expect.poll(() => page.getByRole("menuitem").elements().length).toBe(1);
      expect(displays(".lyra-menu__divider")).toEqual(["none", "none"]);
    });

    it("should show one muted line when nothing matches", async () => {
      await openContextMenu();
      await userEvent.keyboard("zz");
      await expect.poll(() => page.getByRole("menuitem").elements().length).toBe(0);
      const line = page.getByText("No matches").element();
      expect(computed(line, "display")).not.toBe("none");
      expect(displays(".lyra-menu__divider")).toEqual(["none", "none"]);
    });

    it("should pick the first match on Enter", async () => {
      const onChange = vi.fn();
      await openContextMenu(onChange);
      await userEvent.keyboard("rem{Enter}");
      expect(onChange).toHaveBeenCalledExactlyOnceWith("remove");
      expect(page.getByRole("menu").query()).toBeNull();
    });

    it("should clear the filter on Escape, then close", async () => {
      await openContextMenu();
      await userEvent.keyboard("rem{Escape}");
      await expect.poll(() => page.getByRole("menuitem").elements().length).toBe(5);
      expect(page.getByRole("menu").query()).not.toBeNull();
      await userEvent.keyboard("{Escape}");
      await expect.poll(() => page.getByRole("menu").query()).toBeNull();
    });

    it("should leave the keys typed into a picker's search to the picker", async () => {
      await openContextMenu();
      await userEvent.click(page.getByRole("menuitem", { name: "Labels" }));
      const search = page.getByPlaceholder("Search labels...");
      await userEvent.click(search);
      await userEvent.keyboard("alp");
      await expect.element(search).toHaveValue("alp");
      expect(document.querySelector(".lyra-menu-filter")).toBeNull();
      expect(page.getByRole("menuitem").elements()).toHaveLength(5);
    });
  });

  describe("filter menu", () => {
    it("should take the focus on open and filter as the user types", async () => {
      await renderThemed(
        <Triggers.Provider>
          <Dialog.Frame initialVisible>
            <Dialog.Trigger>Filter</Dialog.Trigger>
            <Menu.Dialog aria-label="Filter" filterable>
              {items(() => {})}
            </Menu.Dialog>
          </Dialog.Frame>
        </Triggers.Provider>,
      );
      const dialog = page.getByRole("dialog", { name: "Filter" });
      await expect.poll(() => document.activeElement).toBe(dialog.element());
      await userEvent.keyboard("lab");
      await expect
        .poll(() =>
          page
            .getByRole("menuitem")
            .elements()
            .map((el) => el.textContent),
        )
        .toEqual(["Labels"]);
      expectFloor(dialog.element());
      await userEvent.keyboard("{ArrowDown}");
      expect(document.activeElement?.textContent).toBe("Labels");
      await userEvent.keyboard("{Escape}");
      await expect.poll(() => page.getByRole("menuitem").elements().length).toBe(5);
      await userEvent.keyboard("{Escape}");
      await expect.element(dialog).not.toBeInTheDocument();
    });
  });
});
