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
import { afterEach, assert, beforeEach, describe, expect, it, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { Dialog } from "@/dialog";
import { Flex } from "@/flex";
import { Menu } from "@/menu";
import { Select } from "@/select";
import { computed, renderThemed, resolve } from "@/styletest";
import { Triggers } from "@/triggers";

const OPTIONS = ["alpha", "beta"];

const Labels = (): ReactElement => (
  <Menu.Sub itemKey="labels" textValue="Labels">
    <Menu.SubTrigger>Labels</Menu.SubTrigger>
    <Select.Frame<string, undefined>
      multiple
      data={OPTIONS}
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

const Stage = (): ReactElement => (
  <Menu.Sub itemKey="stage" textValue="Stage">
    <Menu.SubTrigger>Stage</Menu.SubTrigger>
    <Select.Frame<string, undefined> multiple value={[]} onChange={() => {}}>
      <Select.Dialog surface="menu" aria-label="Stage">
        <Select.List>
          {OPTIONS.map((key) => (
            <Select.Item<string> key={key} itemKey={key}>
              {key}
            </Select.Item>
          ))}
        </Select.List>
      </Select.Dialog>
    </Select.Frame>
  </Menu.Sub>
);

const Move = (): ReactElement => (
  <Menu.Sub itemKey="move" textValue="Move">
    <Menu.SubTrigger>Move</Menu.SubTrigger>
    <Menu.Dialog aria-label="Move">
      <Menu.Menu>
        <Menu.Item itemKey="up">Up</Menu.Item>
        <Menu.Item itemKey="down">Down</Menu.Item>
      </Menu.Menu>
    </Menu.Dialog>
  </Menu.Sub>
);

const LONG = Array.from({ length: 60 }, (_, i) => `option ${i}`);

const Long = (): ReactElement => (
  <Menu.Sub itemKey="long" textValue="Long">
    <Menu.SubTrigger>Long</Menu.SubTrigger>
    <Select.Frame<string, undefined>
      multiple
      virtual
      data={LONG}
      value={[]}
      onChange={() => {}}
    >
      <Select.Dialog surface="menu" aria-label="Long">
        <Select.Body<string> resourceName="option" onSearch={() => {}}>
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

/**
 * A menu at the left or the right edge of the window, low enough that a submenu has
 * room to rise and level its first item.
 */
const Dropdown = ({ right = false }: { right?: boolean }): ReactElement => (
  <Flex.Box
    x
    justify={right ? "end" : "start"}
    style={{ width: "100vw", paddingTop: 100 }}
  >
    <Dialog.Frame initialVisible>
      <Dialog.Trigger>Filter</Dialog.Trigger>
      <Menu.Dialog aria-label="Filter">
        <Menu.Menu>
          <Labels />
          <Stage />
          <Move />
          <Menu.Item itemKey="rename">Rename</Menu.Item>
        </Menu.Menu>
      </Menu.Dialog>
    </Dialog.Frame>
  </Flex.Box>
);

const item = (name: string): HTMLElement =>
  page.getByRole("menuitem", { name, exact: true }).element() as HTMLElement;

const dialog = (name: string): HTMLElement =>
  page.getByRole("dialog", { name }).element() as HTMLElement;

const shown = async (name: string): Promise<HTMLElement> => {
  await expect.poll(() => computed(dialog(name), "opacity")).toBe("1");
  return dialog(name);
};

const open = async (name: string, right = false): Promise<HTMLElement> => {
  await renderThemed(<Dropdown right={right} />);
  await shown("Filter");
  await userEvent.click(item(name));
  return await shown(name);
};

const top = (el: Element): number => el.getBoundingClientRect().top;

// The theme sets its tokens on render, so a spec resolves the token after it renders.
const lit = (): string => resolve("var(--pluto-gray-l4)", "background-color");

describe("Menu.Sub", () => {
  // The default window is too narrow for a menu and a submenu side by side, so the
  // submenu would clamp to the window edge over the menu.
  beforeEach(async () => {
    await page.viewport(800, 600);
  });

  describe("place", () => {
    it("should open with its first item level with its own item", async () => {
      await open("Move");
      expect(top(item("Up"))).toBe(top(item("Move")));
    });

    it("should open with its first option level with its own item", async () => {
      await open("Stage");
      expect(top(page.getByRole("option", { name: "alpha" }).element())).toBe(
        top(item("Stage")),
      );
    });

    it("should open with its first option under the search level with its item", async () => {
      await open("Labels");
      expect(top(page.getByRole("option", { name: "alpha" }).element())).toBe(
        top(item("Labels")),
      );
    });

    it("should lap 4 px over the right edge of the menu", async () => {
      const sub = await open("Move");
      expect(
        dialog("Filter").getBoundingClientRect().right -
          sub.getBoundingClientRect().left,
      ).toBe(4);
    });

    it("should lap 4 px over the left edge of a menu with no room on its right", async () => {
      const sub = await open("Move", true);
      expect(
        sub.getBoundingClientRect().right -
          dialog("Filter").getBoundingClientRect().left,
      ).toBe(4);
    });

    it("should keep the first visible option of a scrolled list level with its item", async () => {
      await renderThemed(
        <Dialog.Frame initialVisible>
          <Dialog.Trigger>Filter</Dialog.Trigger>
          <Menu.Dialog aria-label="Filter" style={{ marginTop: 100 }}>
            <Menu.Menu>
              <Long />
            </Menu.Menu>
          </Menu.Dialog>
        </Dialog.Frame>,
      );
      await shown("Filter");
      await userEvent.click(item("Long"));
      const list = (await shown("Long")).querySelector('[role="listbox"]');
      assert(list instanceof HTMLElement);
      list.scrollTop = list.scrollHeight / 3;
      const row = page.getByRole("option", { name: "option 20", exact: true });
      await expect.element(row).toBeInTheDocument();
      list.scrollTop += top(row.element()) - top(list);
      await expect.poll(() => top(row.element())).toBe(top(list));
      // A resize of the window places the submenu again.
      await page.viewport(820, 600);
      await expect.poll(() => top(row.element())).toBe(top(item("Long")));
    });

    it("should open with its first item level with its own item on the left", async () => {
      await open("Move", true);
      expect(top(item("Up"))).toBe(top(item("Move")));
    });
  });

  describe("light", () => {
    afterEach(() => {
      vi.useRealTimers();
    });

    it("should keep its item lit while the pointer is in the submenu", async () => {
      await open("Move");
      await userEvent.hover(item("Up"));
      expect(computed(item("Move"), "background-color")).toBe(lit());
    });

    it("should not light the items the pointer crosses toward the submenu", async () => {
      await open("Labels");
      const labels = item("Labels");
      const stage = item("Stage");
      // The aim lasts 300 ms, and a slow runner can take longer to move the pointer.
      vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout", "performance"] });
      await userEvent.hover(labels, { position: { x: 10, y: 14 } });
      await userEvent.hover(stage, {
        position: { x: stage.getBoundingClientRect().width - 4, y: 4 },
      });
      expect(computed(stage, "background-color")).toBe(resolve("transparent"));
      expect(computed(labels, "background-color")).toBe(lit());
    });
  });

  describe("keyboard", () => {
    it("should open on Enter", async () => {
      await renderThemed(<Dropdown />);
      await shown("Filter");
      item("Stage").focus();
      await userEvent.keyboard("{Enter}");
      await shown("Stage");
    });

    it("should open on Space", async () => {
      await renderThemed(<Dropdown />);
      await shown("Filter");
      item("Stage").focus();
      await userEvent.keyboard(" ");
      await shown("Stage");
    });
  });
});

describe("Menu.Sub", () => {
  interface OptionsProps {
    name: string;
    search?: boolean;
    onCreate?: (name: string) => void;
  }

  /** The content of a submenu: a selection, with a search or without one. */
  const Options = ({ name, search = false, onCreate }: OptionsProps): ReactElement => {
    const [value, setValue] = useState<string[]>([]);
    if (search)
      return (
        <Select.Frame<string, undefined>
          multiple
          data={OPTIONS}
          value={value}
          onChange={setValue}
        >
          <Select.Dialog surface="menu" aria-label={name}>
            <Select.Body<string>
              resourceName="label"
              onSearch={() => {}}
              footer={
                onCreate && (
                  <Select.CreateItem resourceName="label" onCreate={onCreate} />
                )
              }
            >
              {({ key, ...p }) => (
                <Select.Item key={key} {...p}>
                  {key}
                </Select.Item>
              )}
            </Select.Body>
          </Select.Dialog>
        </Select.Frame>
      );
    return (
      <Select.Frame<string, undefined> multiple value={value} onChange={setValue}>
        <Select.Dialog surface="menu" aria-label={name}>
          <Select.List>
            {OPTIONS.map((key) => (
              <Select.Item<string> key={key} itemKey={key}>
                {key}
              </Select.Item>
            ))}
          </Select.List>
        </Select.Dialog>
      </Select.Frame>
    );
  };

  interface ItemsProps {
    onPick?: (key: string) => void;
    onCreate?: (name: string) => void;
  }

  /** A submenu that holds menu items. */
  const Move = ({ onPick }: ItemsProps): ReactElement => (
    <Menu.Sub itemKey="move" textValue="Move">
      <Menu.SubTrigger>Move</Menu.SubTrigger>
      <Menu.Dialog aria-label="Move">
        <Menu.Menu onChange={onPick}>
          <Menu.Item itemKey="up">Up</Menu.Item>
          <Menu.Item itemKey="down">Down</Menu.Item>
        </Menu.Menu>
      </Menu.Dialog>
    </Menu.Sub>
  );

  const Items = ({ onPick, onCreate }: ItemsProps): ReactElement => (
    <Menu.Menu>
      <Menu.Sub itemKey="labels" textValue="Labels">
        <Menu.SubTrigger>Labels</Menu.SubTrigger>
        <Options name="Labels" search onCreate={onCreate} />
      </Menu.Sub>
      <Menu.Sub itemKey="stage" textValue="Stage">
        <Menu.SubTrigger>Stage</Menu.SubTrigger>
        <Options name="Stage" />
      </Menu.Sub>
      <Menu.Sub itemKey="time" textValue="Time">
        <Menu.SubTrigger>Time</Menu.SubTrigger>
        <Options name="Time" />
      </Menu.Sub>
      <Move onPick={onPick} />
      <Menu.Item itemKey="rename">Rename</Menu.Item>
    </Menu.Menu>
  );

  const Dropdown = (): ReactElement => (
    <Dialog.Frame initialVisible>
      <Dialog.Trigger>Filter</Dialog.Trigger>
      <Menu.Dialog>
        <Items />
      </Menu.Dialog>
    </Dialog.Frame>
  );

  const ContextMenu = ({ onPick, onCreate }: ItemsProps): ReactElement => {
    const ctx = Menu.useContextMenu();
    return (
      <Menu.ContextMenu
        menu={() => <Items onPick={onPick} onCreate={onCreate} />}
        {...ctx}
      >
        <button onContextMenu={ctx.open}>Opener</button>
      </Menu.ContextMenu>
    );
  };

  /** A context menu opened over a dialog. */
  const ContextMenuInDialog = (): ReactElement => (
    <Dialog.Frame initialVisible>
      <Dialog.Trigger>Panel</Dialog.Trigger>
      <Dialog.Dialog aria-label="Panel">
        <ContextMenu />
      </Dialog.Dialog>
    </Dialog.Frame>
  );

  const SWAP = "lyra-menu-dialog--swap";
  const LIT = "lyra-menu-sub__trigger--lit";

  /** A fixed spot beside the menu, so the pointer moves of a spec aim at a submenu. */
  const BESIDE: DOMRect = {
    left: 150,
    top: 0,
    width: 200,
    height: 200,
    right: 350,
    bottom: 200,
    x: 150,
    y: 0,
    toJSON: () => "",
  };

  const renderWithTriggers = (element: ReactElement): void => {
    render(<Triggers.Provider>{element}</Triggers.Provider>);
  };

  /** @returns the item of the closed submenu with the name. */
  const closed = (name: string): HTMLElement =>
    screen.getByRole("menuitem", { name, expanded: false });

  /** @returns the item of the open submenu with the name. */
  const opened = (name: string): HTMLElement =>
    screen.getByRole("menuitem", { name, expanded: true });

  /** @returns the content of the open submenu with the name. */
  const content = (name: string): HTMLElement => screen.getByRole("dialog", { name });

  const queryContent = (name: string): HTMLElement | null =>
    screen.queryByRole("dialog", { name });

  const move = (target: HTMLElement, clientX: number, clientY: number): void => {
    fireEvent.pointerMove(target, { pointerType: "mouse", clientX, clientY });
  };

  const advance = (ms: number): void => {
    act(() => {
      vi.advanceTimersByTime(ms);
    });
  };

  const press = (
    key: string,
    target: Element | null = document.activeElement,
  ): void => {
    if (target == null) throw new Error("nothing has focus");
    fireEvent.keyDown(target, { key, code: key });
  };

  const escape = (): void => {
    act(() => {
      fireEvent.keyDown(window, { key: "Escape", code: "Escape" });
    });
    fireEvent.keyUp(window, { key: "Escape", code: "Escape" });
  };

  /** Opens the dropdown menu, then holds the clock so its timers run on demand. */
  const openDropdown = (): void => {
    renderWithTriggers(<Dropdown />);
    vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout", "performance"] });
  };

  /** Opens the labels, then moves the pointer across stage toward their content. */
  const aimAtLabels = (): void => {
    openDropdown();
    press("ArrowRight", closed("Labels"));
    vi.spyOn(content("Labels"), "getBoundingClientRect").mockReturnValue(BESIDE);
    move(opened("Labels"), 100, 14);
    move(closed("Stage"), 140, 32);
  };

  /** Opens the context menu, then moves the focus to its first item. */
  const openContextMenu = (element: ReactElement = <ContextMenu />): HTMLElement => {
    renderWithTriggers(element);
    const opener = screen.getByText("Opener");
    opener.focus();
    fireEvent.contextMenu(opener);
    return opener;
  };

  describe("pointer", () => {
    afterEach(() => {
      act(() => {
        vi.runOnlyPendingTimers();
      });
      vi.useRealTimers();
    });

    it("should open once the pointer rests on its item", () => {
      openDropdown();
      move(closed("Time"), 50, 70);
      advance(99);
      expect(queryContent("Time")).toBeNull();
      advance(1);
      expect(content("Time").className).not.toContain(SWAP);
    });

    it("should not open a submenu the pointer only passes", () => {
      openDropdown();
      move(closed("Stage"), 50, 42);
      advance(60);
      move(closed("Time"), 50, 70);
      advance(100);
      // A submenu that opened over the submenu of stage would carry the swap.
      expect(content("Time").className).not.toContain(SWAP);
      expect(queryContent("Stage")).toBeNull();
    });

    it("should ignore a pointer that is not a mouse", () => {
      openDropdown();
      fireEvent.pointerMove(closed("Time"), { pointerType: "touch" });
      advance(100);
      expect(queryContent("Time")).toBeNull();
    });

    it("should close the open submenu once the pointer rests on another item", () => {
      openDropdown();
      fireEvent.click(closed("Stage"));
      move(screen.getByRole("menuitem", { name: "Rename" }), 50, 120);
      advance(100);
      expect(queryContent("Stage")).toBeNull();
    });

    it("should go dark while the pointer rests on another item", () => {
      openDropdown();
      fireEvent.click(closed("Stage"));
      expect(opened("Stage").className).toContain(LIT);
      move(screen.getByRole("menuitem", { name: "Rename" }), 50, 120);
      expect(opened("Stage").className).not.toContain(LIT);
    });

    it("should keep the open submenu while the pointer crosses another item toward it", () => {
      aimAtLabels();
      advance(250);
      move(screen.getByPlaceholderText("Search labels..."), 200, 50);
      advance(300);
      expect(content("Labels")).toBeTruthy();
      expect(queryContent("Stage")).toBeNull();
    });

    it("should swap in the submenu under the pointer once the aim lapses", () => {
      aimAtLabels();
      advance(300);
      expect(content("Stage").className).toContain(SWAP);
      expect(queryContent("Labels")).toBeNull();
    });

    it("should end the aim when the pointer heads past the submenu", () => {
      aimAtLabels();
      move(closed("Stage"), 145, 180);
      advance(100);
      expect(content("Stage").className).toContain(SWAP);
    });

    it("should end the aim when the pointer drops without moving toward the submenu", () => {
      aimAtLabels();
      move(closed("Stage"), 141, 50);
      advance(100);
      expect(content("Stage").className).toContain(SWAP);
    });
  });

  describe("press", () => {
    it("should open at once on a click", () => {
      renderWithTriggers(<Dropdown />);
      fireEvent.click(closed("Stage"));
      expect(content("Stage")).toBeTruthy();
    });

    it("should open at once on a tap", () => {
      renderWithTriggers(<Dropdown />);
      const stage = closed("Stage");
      fireEvent.pointerDown(stage, { pointerType: "touch" });
      fireEvent.pointerUp(stage, { pointerType: "touch" });
      fireEvent.click(stage);
      expect(content("Stage")).toBeTruthy();
    });
  });

  describe("keyboard", () => {
    it("should open on ArrowRight and close on ArrowLeft, giving the focus back", () => {
      renderWithTriggers(<Dropdown />);
      const labels = closed("Labels");
      labels.focus();
      press("ArrowRight");
      const search = screen.getByPlaceholderText("Search labels...");
      expect(document.activeElement).toBe(search);
      press("ArrowLeft", search);
      expect(queryContent("Labels")).toBeNull();
      expect(document.activeElement).toBe(labels);
    });

    it("should leave ArrowLeft to a search that holds text", () => {
      renderWithTriggers(<Dropdown />);
      press("ArrowRight", closed("Labels"));
      const search = screen.getByPlaceholderText("Search labels...");
      fireEvent.change(search, { target: { value: "al" } });
      press("ArrowLeft", search);
      expect(content("Labels")).toBeTruthy();
    });

    it("should give the focus to a submenu without a search", () => {
      renderWithTriggers(<Dropdown />);
      press("ArrowRight", closed("Stage"));
      expect(document.activeElement).toBe(content("Stage"));
    });

    it("should give the focus to the first item of a submenu of items", () => {
      renderWithTriggers(<Dropdown />);
      press("ArrowRight", closed("Move"));
      expect(document.activeElement).toBe(screen.getByRole("menuitem", { name: "Up" }));
    });

    it("should close one level on Escape", () => {
      renderWithTriggers(<Dropdown />);
      press("ArrowRight", closed("Stage"));
      escape();
      expect(queryContent("Stage")).toBeNull();
      expect(closed("Stage")).toBeTruthy();
    });
  });

  describe("context menu", () => {
    it("should close only the submenu on Escape", () => {
      openContextMenu();
      press("ArrowRight", closed("Stage"));
      escape();
      expect(queryContent("Stage")).toBeNull();
      expect(screen.getByRole("menu")).toBeTruthy();
      escape();
      expect(screen.queryByRole("menu")).toBeNull();
    });

    it("should close only the submenu on Escape over a dialog", () => {
      openContextMenu(<ContextMenuInDialog />);
      press("ArrowRight", closed("Stage"));
      escape();
      expect(queryContent("Stage")).toBeNull();
      expect(screen.getByRole("menu")).toBeTruthy();
      expect(content("Panel")).toBeTruthy();
    });

    it("should stay open on a click inside a submenu", () => {
      openContextMenu();
      fireEvent.click(closed("Labels"));
      fireEvent.click(screen.getByPlaceholderText("Search labels..."));
      fireEvent.click(content("Labels"));
      fireEvent.click(screen.getByText("alpha"));
      expect(screen.getByRole("menu")).toBeTruthy();
      expect(content("Labels")).toBeTruthy();
    });

    it("should close once a submenu option that leaves the menu is picked", () => {
      const onCreate = vi.fn();
      openContextMenu(<ContextMenu onCreate={onCreate} />);
      fireEvent.click(closed("Labels"));
      fireEvent.click(screen.getByRole("option", { name: "New label" }));
      expect(onCreate).toHaveBeenCalledExactlyOnceWith("");
      expect(screen.queryByRole("menu")).toBeNull();
    });

    it("should stay open once a submenu option creates from the search", () => {
      const onCreate = vi.fn();
      openContextMenu(<ContextMenu onCreate={onCreate} />);
      fireEvent.click(closed("Labels"));
      fireEvent.change(screen.getByPlaceholderText("Search labels..."), {
        target: { value: "gamma" },
      });
      fireEvent.click(screen.getByRole("option", { name: 'Create label "gamma"' }));
      expect(onCreate).toHaveBeenCalledExactlyOnceWith("gamma");
      expect(screen.getByRole("menu")).toBeTruthy();
    });

    it("should close and give the focus back once an item of a submenu is picked", () => {
      const onPick = vi.fn();
      const opener = openContextMenu(<ContextMenu onPick={onPick} />);
      press("ArrowRight", closed("Move"));
      fireEvent.click(screen.getByRole("menuitem", { name: "Down" }));
      expect(onPick).toHaveBeenCalledExactlyOnceWith("down");
      expect(screen.queryByRole("menu")).toBeNull();
      expect(document.activeElement).toBe(opener);
    });
  });
});
