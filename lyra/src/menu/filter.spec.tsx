// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, render, screen } from "@testing-library/react";
import { type ReactElement, useState } from "react";
import { describe, expect, it, vi } from "vitest";

import { Dialog } from "@/dialog";
import { Icon } from "@/icon";
import { Menu } from "@/menu";
import { Select } from "@/select";
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

// Presses a key as a browser does: a key the page does not take edits a focused field.
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

describe("menu filter", () => {
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
