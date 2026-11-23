// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// The pure entry leaves the act environment of the browser specs alone.
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react/pure";
import { type ReactElement, Suspense, use, useState } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { Dialog } from "@/dialog";
import { Icon } from "@/icon";
import { Menu } from "@/menu";
import { Select } from "@/select";
import { computed, renderThemed, resolve } from "@/styletest";
import { Triggers } from "@/triggers";

const Labels = (): ReactElement => (
  <Menu.Sub itemKey="labels" textValue="Labels">
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

// Three groups, where a submenu wraps its item in a frame.
const GROUPS = (
  <Menu.Menu>
    <Menu.Item itemKey="copy">Copy</Menu.Item>
    <Menu.Divider />
    <Menu.Item itemKey="rename">Rename</Menu.Item>
    <Menu.Divider />
    <Menu.Sub itemKey="paste" textValue="Paste">
      <Menu.SubTrigger>Paste</Menu.SubTrigger>
      <Menu.Dialog>
        <Menu.Menu>
          <Menu.Item itemKey="plain">Plain</Menu.Item>
        </Menu.Menu>
      </Menu.Dialog>
    </Menu.Sub>
  </Menu.Menu>
);

// Items that show once their data loads, as items that read Core data do.
const Loaded = ({ items }: { items: Promise<void> }): ReactElement => {
  use(items);
  return (
    <>
      <Menu.Item itemKey="rename">Rename</Menu.Item>
      <Menu.Item itemKey="remove">Remove</Menu.Item>
    </>
  );
};

const ContextMenu = ({
  onChange,
  menu,
}: {
  onChange: (key: string) => void;
  menu?: ReactElement;
}): ReactElement => {
  const ctx = Menu.useContextMenu();
  return (
    <Menu.ContextMenu menu={() => menu ?? items(onChange)} {...ctx}>
      <button onContextMenu={ctx.open}>Opener</button>
    </Menu.ContextMenu>
  );
};

const openContextMenu = async (
  onChange = vi.fn(),
  items?: ReactElement,
): Promise<HTMLElement> => {
  const { getByText } = await renderThemed(
    <Triggers.Provider>
      <ContextMenu onChange={onChange} menu={items} />
    </Triggers.Provider>,
  );
  await userEvent.click(getByText("Opener"), { button: "right" });
  const menu = page.getByRole("menu").element() as HTMLElement;
  await expect.poll(() => document.activeElement).toBe(menu);
  return menu;
};

const displays = (selector: string): string[] =>
  Array.from(document.querySelectorAll(selector)).map((el) => computed(el, "display"));

// A filtered menu keeps the width of the menu floor.
const expectFloor = (menu: Element): void => {
  expect(computed(menu, "min-width")).toBe(resolve("32rem", "min-width"));
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
    it("should open a search that spans the top of the menu", async () => {
      const menu = await openContextMenu();
      await userEvent.keyboard("ren");
      const search = page.getByRole("textbox");
      await expect.element(search).toHaveValue("ren");
      await expect.element(search).toHaveFocus();
      expect((search.element() as HTMLInputElement).selectionStart).toBe(3);
      const line = document.querySelector(".lyra-menu-filter") as HTMLElement;
      const lineBox = line.getBoundingClientRect();
      const menuBox = menu.getBoundingClientRect();
      expect(menuBox.right - lineBox.right).toBe(lineBox.left - menuBox.left);
      expect(lineBox.top - menuBox.top).toBe(1);
      expect(computed(line, "border-bottom-width")).toBe("1px");
    });

    it("should keep the menu floor as items hide", async () => {
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

    it("should hide the dividers after the last match", async () => {
      await openContextMenu();
      await userEvent.keyboard("cop");
      await expect.poll(() => page.getByRole("menuitem").elements().length).toBe(1);
      expect(displays(".lyra-menu__divider")).toEqual(["none", "none"]);
    });

    it("should show one divider between two groups that match around a hidden group", async () => {
      await openContextMenu(vi.fn(), GROUPS);
      await userEvent.keyboard("p");
      await expect.poll(() => page.getByRole("menuitem").elements().length).toBe(2);
      expect(displays(".lyra-menu__divider")).toEqual(["flex", "none"]);
    });

    it("should hide the dividers before the first match", async () => {
      await openContextMenu(vi.fn(), GROUPS);
      await userEvent.keyboard("pa");
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

    it("should take a click in the search, with the menu open", async () => {
      await openContextMenu();
      await userEvent.keyboard("ren");
      const search = page.getByRole("textbox");
      await userEvent.click(search);
      await userEvent.keyboard("x");
      await expect.element(search).toHaveValue("renx");
      expect(page.getByRole("menu").query()).not.toBeNull();
    });

    it("should light the first match while the search has the focus", async () => {
      await openContextMenu();
      await userEvent.keyboard("re");
      const [first, second] = page.getByRole("menuitem").elements();
      await expect
        .poll(() => computed(first, "background-color"))
        .not.toBe(computed(second, "background-color"));
    });

    it("should keep the first match lit under the pointer on a disabled item", async () => {
      await openContextMenu(
        vi.fn(),
        <Menu.Menu>
          <Menu.Item itemKey="rename">Rename</Menu.Item>
          <Menu.Item itemKey="remove" disabled>
            Remove
          </Menu.Item>
          <Menu.Item itemKey="reload">Reload</Menu.Item>
        </Menu.Menu>,
      );
      await userEvent.keyboard("re");
      const [first, , third] = page.getByRole("menuitem").elements();
      await userEvent.hover(page.getByRole("menuitem", { name: "Remove" }));
      await expect
        .poll(() => computed(first, "background-color"))
        .not.toBe(computed(third, "background-color"));
    });

    it("should light the first match of items that load after the search opens", async () => {
      const { promise: items, resolve: load } = Promise.withResolvers<void>();
      await openContextMenu(
        vi.fn(),
        <Menu.Menu>
          <Menu.Item itemKey="copy">Copy</Menu.Item>
          <Suspense fallback={null}>
            <Loaded items={items} />
          </Suspense>
        </Menu.Menu>,
      );
      await userEvent.keyboard("re");
      await expect.element(page.getByRole("textbox")).toHaveValue("re");
      load();
      await expect.poll(() => page.getByRole("menuitem").elements().length).toBe(2);
      const [first, second] = page.getByRole("menuitem").elements();
      await expect
        .poll(() => computed(first, "background-color"))
        .not.toBe(computed(second, "background-color"));
    });

    it("should pick the item under the pointer on Enter", async () => {
      const onChange = vi.fn();
      await openContextMenu(onChange);
      await userEvent.keyboard("re");
      await userEvent.hover(page.getByRole("menuitem", { name: "Remove" }));
      await userEvent.keyboard("{Enter}");
      expect(onChange).toHaveBeenCalledExactlyOnceWith("remove");
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
            <Menu.Dialog aria-label="Filter">{items(() => {})}</Menu.Dialog>
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

describe("menu filter", () => {
  afterEach(cleanup);

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
      <Menu.Sub itemKey="color" textValue="Color">
        <Menu.SubTrigger>Color</Menu.SubTrigger>
        <Menu.Dialog>
          <Menu.Menu>
            <Menu.Item itemKey="sky">Sky</Menu.Item>
          </Menu.Menu>
        </Menu.Dialog>
      </Menu.Sub>
    </Menu.Menu>
  );

  const StageTrigger = (): ReactElement => <Menu.SubTrigger>Stage</Menu.SubTrigger>;

  const ContextMenu = ({ items }: { items: ReactElement }): ReactElement => {
    const ctx = Menu.useContextMenu();
    return (
      <Menu.ContextMenu menu={() => items} {...ctx}>
        <button onContextMenu={ctx.open}>Opener</button>
      </Menu.ContextMenu>
    );
  };

  const open = (items: ReactElement = ITEMS): HTMLElement => {
    render(
      <Triggers.Provider>
        <ContextMenu items={items} />
      </Triggers.Provider>,
    );
    const opener = screen.getByText("Opener");
    opener.focus();
    fireEvent.contextMenu(opener);
    return opener;
  };

  // Presses a key as a browser does: a key the page does not take edits a focused
  // field.
  const press = (key: string, init?: KeyboardEventInit): void => {
    const el = document.activeElement;
    if (el == null) throw new Error("nothing has focus");
    const taken = !fireEvent.keyDown(el, { key, code: key, ...init });
    if (taken || !(el instanceof HTMLInputElement)) return;
    if (key === "Backspace")
      fireEvent.change(el, { target: { value: el.value.slice(0, -1) } });
    else if (key.length === 1)
      fireEvent.change(el, { target: { value: el.value + key } });
  };

  const type = (text: string): void => {
    for (const key of text) press(key);
  };

  const shown = (): string[] =>
    screen.queryAllByRole("menuitem").map((el) => el.textContent ?? "");

  const focused = (): string | null | undefined => document.activeElement?.textContent;

  const search = (): HTMLInputElement | null =>
    document.querySelector(".lyra-menu-filter input");

  const lit = (): string | null | undefined =>
    document.querySelector("[data-highlighted]")?.textContent;

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

    it("should match an item on the text inside its elements", () => {
      open(
        <Menu.Menu>
          <Menu.Item itemKey="copy">
            <Icon.Search />
            <span>Copy </span>
            link {2}
          </Menu.Item>
        </Menu.Menu>,
      );
      type("copy link 2");
      expect(shown()).toEqual(["Copy link 2"]);
    });

    it("should match an item on its textValue", () => {
      open();
      type("fav");
      expect(screen.getAllByRole("menuitem")).toHaveLength(1);
      expect(screen.getByRole("menuitem").textContent).toBe("");
    });

    it("should hide a copy item whose text does not match", () => {
      open(
        <Menu.Menu>
          <Menu.CopyItem itemKey="copy" text="hello">
            Copy name
          </Menu.CopyItem>
          <Menu.Item itemKey="rename">Rename</Menu.Item>
        </Menu.Menu>,
      );
      type("cop");
      expect(shown()).toEqual([" Copy name"]);
      type("x");
      expect(shown()).toEqual([]);
    });

    it("should match a submenu on its textValue when a component wraps its item", () => {
      open(
        <Menu.Menu>
          <Menu.Sub itemKey="stage" textValue="Stage">
            <StageTrigger />
            <Menu.Dialog>
              <Menu.Menu>
                <Menu.Item itemKey="done">Done</Menu.Item>
              </Menu.Menu>
            </Menu.Dialog>
          </Menu.Sub>
          <Menu.Item itemKey="rename">Rename</Menu.Item>
        </Menu.Menu>,
      );
      type("sta");
      expect(shown()).toEqual(["Stage"]);
    });

    it("should match a submenu on its textValue", () => {
      open();
      type("col");
      expect(shown()).toEqual(["Color"]);
      type("x");
      expect(shown()).toEqual([]);
    });

    it("should show every item of a submenu opened under the filter", () => {
      open();
      type("col");
      press("Enter");
      expect(shown()).toEqual(["Color", "Sky"]);
      expect(focused()).toBe("Sky");
    });

    describe("submenus", () => {
      const CopyMenu = ({
        onPick,
      }: {
        onPick?: (key: string) => void;
      }): ReactElement => (
        <Menu.Menu>
          <Menu.Item itemKey="rename">Rename</Menu.Item>
          <Menu.Sub itemKey="copy" textValue="Copy">
            <Menu.SubTrigger>Copy</Menu.SubTrigger>
            <Menu.Dialog aria-label="Copy">
              <Menu.Menu onChange={onPick}>
                <Menu.Item itemKey="link">
                  <Icon.Search />
                  Link
                </Menu.Item>
                <Menu.CopyItem itemKey="name" text="Stand B">
                  Name
                </Menu.CopyItem>
              </Menu.Menu>
            </Menu.Dialog>
          </Menu.Sub>
          <Menu.Sub itemKey="color" textValue="Color">
            <Menu.SubTrigger>Color</Menu.SubTrigger>
            <Menu.Dialog itemless>
              <input aria-label="Hex" />
            </Menu.Dialog>
          </Menu.Sub>
          <Menu.Sub itemKey="labels" textValue="Labels">
            <Menu.SubTrigger>Labels</Menu.SubTrigger>
            <Select.Frame<string, undefined> value={[]} onChange={() => {}} multiple>
              <Select.Dialog surface="menu" aria-label="Labels">
                <Select.List>
                  <Select.Item<string> itemKey="link">Linked</Select.Item>
                </Select.List>
              </Select.Dialog>
            </Select.Frame>
          </Menu.Sub>
        </Menu.Menu>
      );

      it("should show the items of an unmatched submenu after its name", () => {
        open(<CopyMenu />);
        type("link");
        expect(shown()).toEqual(["CopyLink"]);
        const path = document.querySelector(".lyra-menu-item__path");
        expect(path?.textContent).toBe("Copy");
        expect(path?.previousElementSibling?.tagName).toBe("svg");
      });

      it("should match the words of a filter across a submenu name and an item", () => {
        open(<CopyMenu />);
        type("copy link");
        expect(shown()).toEqual(["CopyLink"]);
      });

      it("should match the words of a filter in any order", () => {
        open(<CopyMenu />);
        type("link copy");
        expect(shown()).toEqual(["CopyLink"]);
      });

      it("should show a submenu whose name matches, and not its items", () => {
        open(<CopyMenu />);
        type("cop");
        expect(shown()).toEqual(["Copy"]);
      });

      it("should lead a copy item with the copy icon before the submenu name", () => {
        open(<CopyMenu />);
        type("copy na");
        const item = screen.getByRole("menuitem");
        expect(item.textContent).toBe(" CopyName");
        expect(item.firstElementChild?.tagName).toBe("svg");
        expect(item.querySelector(".lyra-menu-item__path")).toBeTruthy();
      });

      it("should pick an item of a submenu with Enter from the search", () => {
        const onPick = vi.fn();
        open(<CopyMenu onPick={onPick} />);
        type("link");
        press("Enter");
        expect(onPick).toHaveBeenCalledExactlyOnceWith("link");
        expect(screen.queryByRole("menu")).toBeNull();
      });

      it("should show nothing of an itemless dialog", () => {
        open(<CopyMenu />);
        type("hex");
        expect(screen.queryByLabelText("Hex")).toBeNull();
        expect(shown()).toEqual([]);
      });

      it("should close a submenu typed into and filter the menu", () => {
        open(<CopyMenu />);
        screen.getByRole("menuitem", { name: "Copy" }).focus();
        press("ArrowRight");
        expect(focused()).toBe("Link");
        expect(screen.getByRole("dialog", { name: "Copy" })).toBeTruthy();
        type("lin");
        expect(search()?.value).toBe("lin");
        expect(document.activeElement).toBe(search());
        expect(screen.queryByRole("dialog", { name: "Copy" })).toBeNull();
        expect(shown()).toEqual(["CopyLink"]);
        press("Escape");
        expect(screen.queryByRole("dialog", { name: "Copy" })).toBeNull();
      });

      it("should keep a submenu open on Space", () => {
        open(<CopyMenu />);
        screen.getByRole("menuitem", { name: "Copy" }).focus();
        press("ArrowRight");
        press(" ");
        expect(focused()).toBe("Link");
        expect(screen.getByRole("dialog", { name: "Copy" })).toBeTruthy();
        expect(search()).toBeNull();
      });

      it("should leave typed text to a field in a submenu", () => {
        open(<CopyMenu />);
        screen.getByRole("menuitem", { name: "Color" }).focus();
        press("ArrowRight");
        const hex = screen.getByLabelText<HTMLInputElement>("Hex");
        hex.focus();
        type("ff");
        expect(hex.value).toBe("ff");
        expect(search()).toBeNull();
      });

      describe("filtered selection", () => {
        const NAMES = ["Leak check", "Cold flow"];

        const LabelsMenu = ({
          onChange,
          pending = false,
        }: {
          onChange: (keys: string[], extra: { clicked: string | null }) => void;
          /** Whether the search of the list never answers, like a slow server. */
          pending?: boolean;
        }): ReactElement => {
          const [data, setData] = useState(NAMES);
          const handleSearch = (term: string): void => {
            if (pending) return;
            setData(NAMES.filter((n) => n.toLowerCase().includes(term.toLowerCase())));
          };
          return (
            <Menu.Menu>
              <Menu.Item itemKey="rename">Rename</Menu.Item>
              <Menu.Sub itemKey="labels" textValue="Labels">
                <Menu.SubTrigger>Labels</Menu.SubTrigger>
                <Select.Frame<string, undefined>
                  multiple
                  value={[]}
                  onChange={onChange}
                  data={data}
                >
                  <Select.Dialog surface="menu" filtered aria-label="Labels">
                    <Select.Body<string> resourceName="label" onSearch={handleSearch}>
                      {({ key, ...p }) => (
                        <Select.Item<string> key={key} {...p}>
                          <Icon.Label />
                          {p.itemKey}
                        </Select.Item>
                      )}
                    </Select.Body>
                  </Select.Dialog>
                </Select.Frame>
              </Menu.Sub>
            </Menu.Menu>
          );
        };

        it("should show the matching options after the name of the submenu", () => {
          open(<LabelsMenu onChange={vi.fn()} />);
          type("leak");
          expect(shown()).toEqual(["LabelsLeak check"]);
          expect(search()).not.toBeNull();
        });

        it("should match filter words across the submenu name and an option", () => {
          open(<LabelsMenu onChange={vi.fn()} />);
          type("label cold");
          expect(shown()).toEqual(["LabelsCold flow"]);
        });

        it("should pick an option with a click", () => {
          const onChange = vi.fn();
          open(<LabelsMenu onChange={onChange} />);
          type("cold");
          fireEvent.click(screen.getByRole("menuitem", { name: /Cold flow/ }));
          expect(onChange).toHaveBeenCalledExactlyOnceWith(["Cold flow"], {
            clicked: "Cold flow",
          });
          expect(screen.queryByRole("menu")).toBeNull();
        });

        it("should pick the lit option with Enter from the search", () => {
          const onChange = vi.fn();
          open(<LabelsMenu onChange={onChange} />);
          type("leak");
          expect(lit()).toBe("LabelsLeak check");
          press("Enter");
          expect(onChange).toHaveBeenCalledExactlyOnceWith(["Leak check"], {
            clicked: "Leak check",
          });
          expect(screen.queryByRole("menu")).toBeNull();
        });

        it("should hide the options that do not match before the search answers", () => {
          const onChange = vi.fn();
          open(<LabelsMenu onChange={onChange} pending />);
          type("cold");
          expect(shown()).toEqual(["LabelsCold flow"]);
          press("Enter");
          expect(onChange).toHaveBeenCalledExactlyOnceWith(["Cold flow"], {
            clicked: "Cold flow",
          });
        });

        it("should list every option again once the filter clears", () => {
          open(<LabelsMenu onChange={vi.fn()} />);
          type("leak");
          press("Escape");
          screen.getByRole("menuitem", { name: "Labels" }).focus();
          press("ArrowRight");
          expect(screen.getAllByRole("option").map((o) => o.textContent)).toEqual(
            NAMES,
          );
        });
      });

      it("should show nothing of a selection", () => {
        open(<CopyMenu />);
        type("linked");
        expect(screen.queryByText("Linked")).toBeNull();
        expect(shown()).toEqual([]);
      });
    });

    it("should open a search at the top on the first key, with the focus", () => {
      open();
      type("ren");
      expect(search()?.value).toBe("ren");
      expect(document.activeElement).toBe(search());
      expect(screen.getByRole("menu").firstElementChild).toBe(
        document.querySelector(".lyra-menu-filter"),
      );
    });

    it("should keep the menu open on a click in the search", () => {
      open();
      type("ren");
      fireEvent.click(search() as HTMLInputElement);
      expect(screen.getByRole("menu")).toBeTruthy();
    });

    it("should say No matches when no item matches", () => {
      open();
      type("zz");
      expect(shown()).toEqual([]);
      expect(screen.getByText("No matches")).toBeTruthy();
    });

    it("should light the first match", () => {
      open();
      type("re");
      expect(lit()).toBe("Rename");
      type("m");
      expect(lit()).toBe("Remove");
    });

    it("should move the focus from the search over the items that show", () => {
      open();
      type("re");
      press("ArrowDown");
      expect(focused()).toBe("Rename");
      press("ArrowDown");
      expect(focused()).toBe("Remove");
      press("ArrowDown");
      expect(focused()).toBe("Rename");
    });

    it("should move the focus from the search to the last item on ArrowUp", () => {
      open();
      type("re");
      press("ArrowUp");
      expect(focused()).toBe("Remove");
    });

    it("should go on typing in the search from an item", () => {
      open();
      type("re");
      press("ArrowDown");
      type("m");
      expect(search()?.value).toBe("rem");
      expect(document.activeElement).toBe(search());
      expect(search()?.selectionStart).toBe(3);
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

    it("should edit the search with Backspace from an item", () => {
      open();
      type("rem");
      press("ArrowDown");
      press("Backspace");
      expect(search()?.value).toBe("re");
      expect(document.activeElement).toBe(search());
    });

    it("should give the focus to an empty search on Backspace from an item", () => {
      open();
      type("r");
      press("Backspace");
      press("ArrowDown");
      press("Backspace");
      expect(search()?.value).toBe("");
      expect(document.activeElement).toBe(search());
    });

    it("should take a space only after the filter starts", () => {
      open();
      press(" ");
      expect(search()).toBeNull();
      type("o");
      press(" ");
      expect(search()?.value).toBe("o ");
    });

    it("should clear the filter on Escape, then close the menu", () => {
      const opener = open();
      type("rem");
      act(() => press("Escape"));
      expect(screen.getByRole("menu")).toBeTruthy();
      expect(shown()).toHaveLength(5);
      expect(search()).toBeNull();
      expect(document.activeElement).toBe(screen.getByRole("menu"));
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
    const renderDialog = (children: ReactElement): void => {
      render(
        <Triggers.Provider>
          <Dialog.Frame visible>
            <Menu.Dialog>{children}</Menu.Dialog>
          </Dialog.Frame>
        </Triggers.Provider>,
      );
    };

    it("should show nothing of a selection under a submenu that the filter passes", () => {
      renderDialog(
        <Menu.Menu>
          <Menu.Item itemKey="rename">Rename</Menu.Item>
          <Menu.Sub itemKey="parent" textValue="Parent">
            <Menu.SubTrigger>Parent</Menu.SubTrigger>
            <Select.Frame<string, undefined>
              value={undefined}
              onChange={vi.fn()}
              data={["Cold flow"]}
            >
              <Select.Dialog surface="menu" aria-label="Parent">
                <Select.Body<string> resourceName="range" onSearch={vi.fn()}>
                  {({ key, ...p }) => (
                    <Select.Item<string> key={key} {...p}>
                      {p.itemKey}
                    </Select.Item>
                  )}
                </Select.Body>
              </Select.Dialog>
            </Select.Frame>
          </Menu.Sub>
        </Menu.Menu>,
      );
      type("rena");
      expect(shown()).toEqual(["Rename"]);
      expect(screen.queryByText("Cold flow")).toBeNull();
      expect(screen.queryByRole("dialog", { name: "Parent" })).toBeNull();
    });

    it("should take the focus and filter its items", () => {
      renderDialog(ITEMS);
      expect(document.activeElement?.classList.contains("lyra-menu-dialog")).toBe(true);
      type("rem");
      expect(shown()).toEqual(["Remove"]);
    });

    it("should leave the focus and the keys to a field that takes the focus", () => {
      renderDialog(
        <>
          <input aria-label="Search" autoFocus />
          {ITEMS}
        </>,
      );
      const search = screen.getByLabelText("Search");
      expect(document.activeElement).toBe(search);
      type("rem");
      expect(shown()).toHaveLength(5);
      expect(document.querySelector(".lyra-menu-filter")).toBeNull();
    });
  });
});
