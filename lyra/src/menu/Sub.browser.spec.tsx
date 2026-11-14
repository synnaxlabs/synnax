// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";
import { beforeEach, describe, expect, it } from "vitest";
import { page, userEvent } from "vitest/browser";

import { Dialog } from "@/dialog";
import { Flex } from "@/flex";
import { Menu } from "@/menu";
import { Select } from "@/select";
import { computed, renderThemed, resolve } from "@/styletest";

const OPTIONS = ["alpha", "beta"];

const Labels = (): ReactElement => (
  <Menu.Sub itemKey="labels">
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
  <Menu.Sub itemKey="stage">
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
  <Menu.Sub itemKey="move">
    <Menu.SubTrigger>Move</Menu.SubTrigger>
    <Menu.Dialog aria-label="Move">
      <Menu.Menu>
        <Menu.Item itemKey="up">Up</Menu.Item>
        <Menu.Item itemKey="down">Down</Menu.Item>
      </Menu.Menu>
    </Menu.Dialog>
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

    it("should open with its first item level with its own item on the left", async () => {
      await open("Move", true);
      expect(top(item("Up"))).toBe(top(item("Move")));
    });
  });

  describe("light", () => {
    it("should keep its item lit while the pointer is in the submenu", async () => {
      await open("Move");
      await userEvent.hover(item("Up"));
      expect(computed(item("Move"), "background-color")).toBe(lit());
    });

    it("should not light the items the pointer crosses toward the submenu", async () => {
      await open("Labels");
      const labels = item("Labels");
      const stage = item("Stage");
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
