// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { page, userEvent } from "vitest/browser";

import { Dialog } from "@/dialog";
import { Menu } from "@/menu";
import { computed, renderThemed, resolve } from "@/styletest";
import { Triggers } from "@/triggers";

const Values = ({ name }: { name: string }): ReactElement => (
  <Menu.Dialog>
    <Menu.Menu>
      <Menu.Item itemKey={`${name}-1`}>{name} one</Menu.Item>
      <Menu.Item itemKey={`${name}-2`}>{name} two</Menu.Item>
    </Menu.Menu>
  </Menu.Dialog>
);

const PROPERTIES = ["Labels", "Stage", "Owner"];

// The menu opens in a dialog, as a filter menu does, so Escape has a level to keep.
const renderProperties = async (): Promise<(name: string) => HTMLElement> => {
  await renderThemed(
    <Triggers.Provider>
      <Dialog.Frame initialVisible>
        <Dialog.Trigger>Filter</Dialog.Trigger>
        <Menu.Dialog>
          <Menu.Menu>
            {PROPERTIES.map((name) => (
              <Menu.Submenu key={name} itemKey={name}>
                <Menu.SubmenuTrigger>{name}</Menu.SubmenuTrigger>
                <Values name={name} />
              </Menu.Submenu>
            ))}
            <Menu.Item itemKey="clear">Clear</Menu.Item>
          </Menu.Menu>
        </Menu.Dialog>
      </Dialog.Frame>
    </Triggers.Provider>,
  );
  const item = (name: string): HTMLElement =>
    page.getByRole("menuitem", { name, exact: true }).element() as HTMLElement;
  await expect.poll(() => computed(item("Labels"), "opacity")).toBe("1");
  return item;
};

const expanded = (el: Element): string | null => el.getAttribute("aria-expanded");

const surface = (name: string): HTMLElement =>
  page
    .getByRole("menuitem", { name: `${name} one` })
    .element()
    .closest(".lyra-menu-dialog") as HTMLElement;

const pointer = (
  type: string,
  target: Element,
  x: number,
  y: number,
  relatedTarget: Element | null = null,
): void => {
  target.dispatchEvent(
    new PointerEvent(type, {
      bubbles: true,
      clientX: x,
      clientY: y,
      pointerType: "mouse",
      relatedTarget,
    }),
  );
};

const wait = (ms: number): Promise<void> =>
  new Promise((resolve) => setTimeout(resolve, ms));

describe("Submenu", () => {
  describe("open", () => {
    it("should open its submenu once the pointer rests on its item", async () => {
      const item = await renderProperties();
      await userEvent.hover(item("Labels"));
      await expect.poll(() => expanded(item("Labels"))).toBe("true");
      expect(surface("Labels")).not.toBeNull();
    });

    it("should open its submenu at once on a click", async () => {
      const item = await renderProperties();
      await userEvent.click(item("Stage"));
      expect(expanded(item("Stage"))).toBe("true");
    });

    it("should open beside its item, with its first item level", async () => {
      const item = await renderProperties();
      await userEvent.click(item("Stage"));
      const trigger = item("Stage").getBoundingClientRect();
      const dialog = surface("Stage");
      await expect.poll(() => computed(dialog, "opacity")).toBe("1");
      const first = page
        .getByRole("menuitem", { name: "Stage one" })
        .element()
        .getBoundingClientRect();
      expect(first.top).toBeCloseTo(trigger.top, 0);
      expect(dialog.getBoundingClientRect().left).toBeCloseTo(trigger.right + 1, 0);
    });

    it("should keep its item lit while the submenu is open", async () => {
      const item = await renderProperties();
      await userEvent.click(item("Labels"));
      await userEvent.hover(surface("Labels"));
      expect(computed(item("Labels"), "background-color")).toBe(
        resolve("var(--pluto-fill-hover)"),
      );
    });

    it("should light one item at a time", async () => {
      const item = await renderProperties();
      await userEvent.click(item("Labels"));
      await userEvent.hover(item("Clear"));
      expect(computed(item("Labels"), "background-color")).toBe(resolve("transparent"));
      expect(computed(item("Clear"), "background-color")).toBe(
        resolve("var(--pluto-fill-hover)"),
      );
    });

    it("should close once the pointer rests on an item with no submenu", async () => {
      const item = await renderProperties();
      await userEvent.click(item("Labels"));
      await userEvent.hover(item("Clear"));
      await expect.poll(() => expanded(item("Labels"))).toBe("false");
      expect(page.getByRole("menuitem", { name: "Labels one" }).query()).toBeNull();
    });
  });

  describe("swap", () => {
    it("should mark a submenu that replaces another, so it shows at once", async () => {
      const item = await renderProperties();
      await userEvent.click(item("Labels"));
      expect(surface("Labels").classList).not.toContain("lyra-menu-dialog--swap");
      await userEvent.hover(item("Stage"));
      await expect.poll(() => expanded(item("Stage"))).toBe("true");
      expect(expanded(item("Labels"))).toBe("false");
      expect(surface("Stage").classList).toContain("lyra-menu-dialog--swap");
    });

    it("should not mark a submenu that opens after the others close", async () => {
      const item = await renderProperties();
      await userEvent.click(item("Labels"));
      await userEvent.keyboard("{Escape}");
      await expect.poll(() => expanded(item("Labels"))).toBe("false");
      await userEvent.click(item("Stage"));
      expect(surface("Stage").classList).not.toContain("lyra-menu-dialog--swap");
    });
  });

  describe("aim", () => {
    const leaveToward = async (
      item: (name: string) => HTMLElement,
      from: string = "Labels",
      to: string = "Stage",
    ): Promise<{ x: number; y: number }> => {
      await userEvent.click(item(from));
      await expect.poll(() => computed(surface(from), "opacity")).toBe("1");
      const next = item(to).getBoundingClientRect();
      const exit = { x: next.right - 20, y: next.top + 4 };
      pointer("pointerout", item(from), exit.x, exit.y, item(to));
      return exit;
    };

    it("should keep the open submenu while the pointer crosses an item with none", async () => {
      const item = await renderProperties();
      const exit = await leaveToward(item, "Owner", "Clear");
      pointer("pointermove", item("Clear"), exit.x + 4, exit.y + 1);
      expect(item("Clear").hasAttribute("data-aiming")).toBe(true);
      await wait(200);
      expect(expanded(item("Owner"))).toBe("true");
      await expect.poll(() => expanded(item("Owner"))).toBe("false");
    });

    it("should keep the open submenu while the pointer travels toward it", async () => {
      const item = await renderProperties();
      const exit = await leaveToward(item);
      pointer("pointermove", item("Stage"), exit.x + 4, exit.y + 1);
      expect(item("Stage").hasAttribute("data-aiming")).toBe(true);
      await wait(200);
      expect(expanded(item("Labels"))).toBe("true");
      expect(expanded(item("Stage"))).toBe("false");
    });

    it("should open the item under the pointer once the aim runs out", async () => {
      const item = await renderProperties();
      const exit = await leaveToward(item);
      pointer("pointermove", item("Stage"), exit.x + 4, exit.y + 1);
      await expect.poll(() => expanded(item("Stage"))).toBe("true");
      expect(expanded(item("Labels"))).toBe("false");
      expect(item("Stage").hasAttribute("data-aiming")).toBe(false);
    });

    it("should end the aim when the pointer moves away from the submenu", async () => {
      const item = await renderProperties();
      const exit = await leaveToward(item);
      pointer("pointermove", item("Stage"), exit.x - 6, exit.y + 8);
      expect(item("Stage").hasAttribute("data-aiming")).toBe(false);
      await expect.poll(() => expanded(item("Stage"))).toBe("true");
    });

    it("should end the aim when the pointer reaches the submenu", async () => {
      const item = await renderProperties();
      await leaveToward(item);
      const one = page.getByRole("menuitem", { name: "Labels one" }).element();
      const rect = one.getBoundingClientRect();
      pointer("pointermove", one, rect.left + 4, rect.top + 4);
      expect(item("Stage").hasAttribute("data-aiming")).toBe(false);
      await wait(400);
      expect(expanded(item("Labels"))).toBe("true");
    });
  });

  describe("keys", () => {
    it("should open its submenu on ArrowRight", async () => {
      const item = await renderProperties();
      item("Owner").focus();
      await userEvent.keyboard("{ArrowRight}");
      expect(expanded(item("Owner"))).toBe("true");
    });

    it("should close on ArrowLeft and give the focus back to its item", async () => {
      const item = await renderProperties();
      item("Owner").focus();
      await userEvent.keyboard("{ArrowRight}");
      const one = page.getByRole("menuitem", { name: "Owner one" }).element();
      (one as HTMLElement).focus();
      await userEvent.keyboard("{ArrowLeft}");
      await expect.poll(() => expanded(item("Owner"))).toBe("false");
      expect(document.activeElement).toBe(item("Owner"));
    });

    it("should close one level on Escape", async () => {
      const item = await renderProperties();
      item("Owner").focus();
      await userEvent.keyboard("{ArrowRight}");
      (
        page.getByRole("menuitem", { name: "Owner one" }).element() as HTMLElement
      ).focus();
      await userEvent.keyboard("{Escape}");
      await expect.poll(() => expanded(item("Owner"))).toBe("false");
      expect(page.getByRole("button", { name: "Filter" }).element()).toHaveAttribute(
        "aria-expanded",
        "true",
      );
    });
  });
});
