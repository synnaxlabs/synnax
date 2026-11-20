// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { ReactElement, ReactNode } from "react";
import { assert, describe, expect, it } from "vitest";
import { page, userEvent } from "vitest/browser";

import { Button } from "@/button";
import { Color } from "@/color";
import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { Icon } from "@/icon";
import { Menu } from "@/menu";
import { renderThemed } from "@/styletest";
import { Triggers } from "@/triggers";

const COLORS: Color.Named[] = Array.from({ length: 10 }, (_, i) => ({
  key: `c${i}`,
  name: `Color ${i}`,
  light: `#${(i * 20).toString(16).padStart(2, "0")}8040`,
}));

const renderLibrary = async (
  value?: string,
  paletteEnd?: ReactNode,
  swatchMenu?: ReactNode,
): Promise<HTMLElement> => {
  await renderThemed(
    <Triggers.Provider>
      <Color.Picker
        value={value}
        onChange={() => {}}
        library={{
          palettes: [{ key: "p", name: "Palette", colors: COLORS }],
          onNamedChange: () => {},
          paletteEnd,
          swatchMenu,
        }}
      />
    </Triggers.Provider>,
  );
  const picker = document.querySelector(`.${CSS.B("color-picker")}`);
  assert(picker instanceof HTMLElement);
  return picker;
};

const rectOf = (selector: string): DOMRect => {
  const el = document.querySelector(selector);
  assert(el != null);
  return el.getBoundingClientRect();
};

const swatchRect = (i: number): DOMRect =>
  page
    .getByRole("button", { name: `Color ${i}`, exact: true })
    .element()
    .getBoundingClientRect();

const SwatchMenu = (): ReactElement | null => {
  const { swatch } = Color.usePalette();
  const { open } = Dialog.useViews();
  if (swatch == null) return null;
  return (
    <Menu.Menu>
      <Menu.Item
        itemKey="edit"
        onClick={() => open("Edit color", <span>{`editing ${swatch.name}`}</span>)}
      >
        Edit
      </Menu.Item>
    </Menu.Menu>
  );
};

describe("Picker", () => {
  describe("library", () => {
    it("should draw square swatches", async () => {
      await renderLibrary();
      const s = swatchRect(0);
      expect(s.width).toBeCloseTo(s.height, 0);
    });

    it("should size a slotted button as a swatch", async () => {
      await renderLibrary(
        undefined,
        <Button.Button variant="outlined" square aria-label="Add">
          <Icon.Add />
        </Button.Button>,
      );
      const add = page.getByRole("button", { name: "Add" }).element();
      const s = swatchRect(0);
      expect(add.getBoundingClientRect().width).toBeCloseTo(s.width, 0);
      expect(add.getBoundingClientRect().height).toBeCloseTo(s.height, 0);
    });

    it("should focus a right-clicked swatch after an item opens a view and Back", async () => {
      await renderLibrary(undefined, undefined, <SwatchMenu />);
      const swatch = page.getByRole("button", { name: "Color 2", exact: true });
      await userEvent.click(swatch, { button: "right" });
      await userEvent.click(page.getByRole("menuitem", { name: "Edit" }));
      await expect.element(page.getByText("editing Color 2")).toBeVisible();
      await userEvent.click(page.getByRole("button", { name: "Back" }));
      expect(document.activeElement).toBe(swatch.element());
    });

    it("should take the width of an open view, not its own", async () => {
      const picker = await renderLibrary(undefined, undefined, <SwatchMenu />);
      const swatch = page.getByRole("button", { name: "Color 2", exact: true });
      await userEvent.click(swatch, { button: "right" });
      await userEvent.click(page.getByRole("menuitem", { name: "Edit" }));
      await expect.element(page.getByText("editing Color 2")).toBeVisible();
      picker.style.setProperty("--pluto-dialog-views-width", "300px");
      expect(picker.getBoundingClientRect().width).toBeCloseTo(300, 0);
    });

    it("should draw no focus ring around the picker", async () => {
      const picker = await renderLibrary();
      await userEvent.keyboard("{Tab}");
      picker.focus();
      expect(picker.matches(":focus-visible")).toBe(true);
      expect(getComputedStyle(picker).outlineStyle).toBe("none");
    });

    it("should space the swatches equally across and down", async () => {
      await renderLibrary();
      const across = swatchRect(1).left - swatchRect(0).right;
      const down = swatchRect(8).top - swatchRect(0).bottom;
      expect(across).toBeGreaterThan(0);
      expect(across).toBeCloseTo(down, 0);
    });

    it("should fit eight swatches between equal insets", async () => {
      const picker = (await renderLibrary()).getBoundingClientRect();
      const left = swatchRect(0).left - picker.left;
      const right = picker.right - swatchRect(7).right;
      expect(left).toBeGreaterThan(0);
      expect(left).toBeCloseTo(right, 0);
      expect(swatchRect(8).left).toBeCloseTo(swatchRect(0).left, 0);
    });

    it("should fit the widest hex code in the Custom tab", async () => {
      await renderLibrary("#cccccc");
      const hex = page.getByRole("textbox", { name: "Hex" }).element();
      assert(hex instanceof HTMLInputElement);
      expect(hex.scrollWidth).toBeLessThanOrEqual(hex.clientWidth);
    });

    it("should draw the # of the Custom tab in the code font of the digits", async () => {
      await renderLibrary("#cccccc");
      const hex = page.getByRole("textbox", { name: "Hex" }).element();
      const hash = hex
        .closest(`.${CSS.B("color-hex-text")}`)
        ?.querySelector(`.${CSS.BE("input", "start-content")}`);
      assert(hash != null);
      expect(hash.textContent).toBe("#");
      expect(getComputedStyle(hash).fontFamily).toBe(getComputedStyle(hex).fontFamily);
    });

    it("should start the tabs, the search icon, and the swatches on one inset", async () => {
      const picker = (await renderLibrary()).getBoundingClientRect();
      const swatch = swatchRect(0).left - picker.left;
      const tabs = rectOf(`.${CSS.BE("color-picker", "tabs")}`).left - picker.left;
      const icon =
        rectOf(`.${CSS.BE("color-picker", "search")} svg`).left - picker.left;
      expect(tabs).toBeCloseTo(swatch, 0);
      expect(icon).toBeCloseTo(swatch, 0);
    });
  });
});
