// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, render, screen } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Dialog } from "@/dialog";
import { Icon } from "@/icon";
import { Menu } from "@/menu";
import { nodeText } from "@/menu/filter";
import { Triggers } from "@/triggers";

const ITEMS = (
  <Menu.Menu>
    <Menu.Item itemKey="open">
      <Icon.Search />
      Open
    </Menu.Item>
    <Menu.Item itemKey="rename">Rename</Menu.Item>
    <Menu.Divider />
    <Menu.Item itemKey="remove">
      <Icon.Delete />
      Remove
    </Menu.Item>
    <Menu.Item itemKey="favorite" textValue="Favorite">
      <Icon.StarFilled />
    </Menu.Item>
    <Menu.Sub itemKey="color">
      <Menu.SubTrigger>Color</Menu.SubTrigger>
      <Menu.Dialog>
        <Menu.Menu>
          <Menu.Item itemKey="red">Red</Menu.Item>
        </Menu.Menu>
      </Menu.Dialog>
    </Menu.Sub>
  </Menu.Menu>
);

const ContextMenu = (): ReactElement => {
  const ctx = Menu.useContextMenu();
  return (
    <Menu.ContextMenu menu={() => ITEMS} {...ctx}>
      <button onContextMenu={ctx.open}>Opener</button>
    </Menu.ContextMenu>
  );
};

const open = (): HTMLElement => {
  render(
    <Triggers.Provider>
      <ContextMenu />
    </Triggers.Provider>,
  );
  const opener = screen.getByText("Opener");
  opener.focus();
  fireEvent.contextMenu(opener);
  return opener;
};

const press = (key: string, init?: KeyboardEventInit): void => {
  if (document.activeElement == null) throw new Error("nothing has focus");
  fireEvent.keyDown(document.activeElement, { key, code: key, ...init });
};

const type = (text: string): void => {
  for (const key of text) press(key);
};

const shown = (): string[] =>
  screen.queryAllByRole("menuitem").map((el) => el.textContent ?? "");

const focused = (): string | null | undefined => document.activeElement?.textContent;

describe("menu filter", () => {
  describe("nodeText", () => {
    it("should read the text through elements and skip icons", () => {
      const node = (
        <>
          <Icon.Search />
          <span>Copy </span>
          link {2}
        </>
      );
      expect(nodeText(node)).toBe("Copy link 2");
    });
  });

  describe("context menu", () => {
    it("should show every item before a key is typed", () => {
      open();
      expect(shown()).toEqual(["Open", "Rename", "Remove", "", "Color"]);
      expect(screen.queryByText("No matches")).toBeNull();
    });

    it("should hide the items whose text does not match, ignoring case", () => {
      open();
      type("RE");
      expect(shown()).toEqual(["Rename", "Remove"]);
    });

    it("should match an item on its textValue", () => {
      open();
      type("fav");
      expect(screen.getAllByRole("menuitem")).toHaveLength(1);
      expect(screen.getByRole("menuitem").textContent).toBe("");
    });

    it("should match a submenu on the text of its item", () => {
      open();
      type("col");
      expect(shown()).toEqual(["Color"]);
      type("x");
      expect(shown()).toEqual([]);
    });

    it("should show the filter as a line at the top", () => {
      open();
      type("ren");
      const line = document.querySelector(".lyra-menu-filter");
      expect(line?.textContent).toBe("ren");
      expect(screen.getByRole("menu").firstElementChild).toBe(line);
    });

    it("should say No matches when no item matches", () => {
      open();
      type("zz");
      expect(shown()).toEqual([]);
      expect(screen.getByText("No matches")).toBeTruthy();
    });

    it("should give the focus to the first match", () => {
      open();
      type("re");
      expect(focused()).toBe("Rename");
      type("m");
      expect(focused()).toBe("Remove");
    });

    it("should move the focus over the items that show", () => {
      open();
      type("re");
      press("ArrowDown");
      expect(focused()).toBe("Remove");
      press("ArrowDown");
      expect(focused()).toBe("Rename");
    });

    it("should take capital letters typed with Shift", () => {
      open();
      press("N", { shiftKey: true });
      expect(shown()).toEqual(["Open", "Rename"]);
    });

    it("should edit the filter with Backspace", () => {
      open();
      type("rem");
      expect(shown()).toEqual(["Remove"]);
      press("Backspace");
      expect(shown()).toEqual(["Rename", "Remove"]);
    });

    it("should take a space only after the filter starts", () => {
      open();
      press(" ");
      expect(document.querySelector(".lyra-menu-filter")).toBeNull();
      type("o");
      press(" ");
      expect(document.querySelector(".lyra-menu-filter")?.textContent).toBe("o ");
    });

    it("should clear the filter on Escape, then close the menu", () => {
      const opener = open();
      type("rem");
      act(() => press("Escape"));
      expect(screen.getByRole("menu")).toBeTruthy();
      expect(shown()).toHaveLength(5);
      expect(document.querySelector(".lyra-menu-filter")).toBeNull();
      act(() => press("Escape"));
      expect(screen.queryByRole("menu")).toBeNull();
      expect(document.activeElement).toBe(opener);
    });

    it("should open with no filter after it closes", () => {
      open();
      type("rem");
      act(() => press("Escape"));
      act(() => press("Escape"));
      fireEvent.contextMenu(screen.getByText("Opener"));
      expect(shown()).toHaveLength(5);
    });

    it("should leave keys typed into a field to the field", () => {
      const FieldMenu = (): ReactElement => {
        const ctx = Menu.useContextMenu();
        return (
          <Menu.ContextMenu
            menu={() => (
              <Menu.Menu>
                <input aria-label="name" />
                <Menu.Item itemKey="rename">Rename</Menu.Item>
              </Menu.Menu>
            )}
            {...ctx}
          >
            <button onContextMenu={ctx.open}>Opener</button>
          </Menu.ContextMenu>
        );
      };
      render(
        <Triggers.Provider>
          <FieldMenu />
        </Triggers.Provider>,
      );
      fireEvent.contextMenu(screen.getByText("Opener"));
      screen.getByLabelText("name").focus();
      type("zz");
      expect(document.querySelector(".lyra-menu-filter")).toBeNull();
      expect(shown()).toEqual(["Rename"]);
    });
  });

  describe("menu dialog", () => {
    const renderDialog = (filterable: boolean): void => {
      render(
        <Triggers.Provider>
          <Dialog.Frame visible>
            <Menu.Dialog filterable={filterable}>{ITEMS}</Menu.Dialog>
          </Dialog.Frame>
        </Triggers.Provider>,
      );
    };

    it("should take the focus and filter its items when filterable", () => {
      renderDialog(true);
      expect(document.activeElement?.classList.contains("lyra-menu-dialog")).toBe(true);
      type("rem");
      expect(shown()).toEqual(["Remove"]);
    });

    it("should leave its items and the focus alone when not filterable", () => {
      renderDialog(false);
      expect(document.activeElement).toBe(document.body);
      expect(shown()).toHaveLength(5);
    });
  });
});
