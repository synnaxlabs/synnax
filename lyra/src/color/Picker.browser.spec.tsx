// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, type ReactNode } from "react";
import { assert, describe, expect, it } from "vitest";
import { page, userEvent } from "vitest/browser";

import { Button } from "@/button";
import { Color } from "@/color";
import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Input } from "@/input";
import { Menu } from "@/menu";
import { renderThemed } from "@/styletest";
import { Triggers } from "@/triggers";

const COLORS: Color.Named[] = Array.from({ length: 10 }, (_, i) => ({
  key: `c${i}`,
  name: `Color ${i}`,
  light: `#${(i * 20).toString(16).padStart(2, "0")}8040`,
}));

interface RenderLibraryOptions
  extends
    Pick<Color.PickerProps, "style">,
    Pick<Color.Library, "paletteEnd" | "swatchMenu" | "footer"> {
  value?: string;
}

const renderLibrary = async ({
  value,
  style,
  paletteEnd,
  swatchMenu,
  footer,
}: RenderLibraryOptions = {}): Promise<HTMLElement> => {
  const { container } = await renderThemed(
    <Triggers.Provider>
      <Color.Picker
        value={value}
        style={style}
        onChange={() => {}}
        library={{
          palettes: [{ key: "p", name: "Palette", colors: COLORS }],
          onNamedChange: () => {},
          paletteEnd,
          swatchMenu,
          footer,
        }}
      />
    </Triggers.Provider>,
  );
  const picker = container.querySelector(`.${CSS.B("color-picker")}`);
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

const CREATE = (
  <Button.CreateListItem size="small">Create palette</Button.CreateListItem>
);

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
      await renderLibrary({
        paletteEnd: (
          <Button.Button variant="outlined" square aria-label="Add">
            <Icon.Add />
          </Button.Button>
        ),
      });
      const add = page.getByRole("button", { name: "Add" }).element();
      const s = swatchRect(0);
      expect(add.getBoundingClientRect().width).toBeCloseTo(s.width, 0);
      expect(add.getBoundingClientRect().height).toBeCloseTo(s.height, 0);
    });

    it("should focus a right-clicked swatch after an item opens a view and Back", async () => {
      await renderLibrary({ swatchMenu: <SwatchMenu /> });
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
      await renderLibrary({ value: "#cccccc" });
      const hex = page.getByRole("textbox", { name: "Hex" }).element();
      assert(hex instanceof HTMLInputElement);
      expect(hex.scrollWidth).toBeLessThanOrEqual(hex.clientWidth);
    });

    it("should draw the # of the Custom tab in the code font of the digits", async () => {
      await renderLibrary({ value: "#cccccc" });
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

    it("should keep an equal edge around the footer's create item, under a hairline across the picker", async () => {
      const picker = (await renderLibrary({ footer: CREATE })).getBoundingClientRect();
      const footer = document.querySelector(`.${CSS.BE("color-picker", "footer")}`);
      assert(footer instanceof HTMLElement);
      const f = footer.getBoundingClientRect();
      const hairline = Number.parseFloat(getComputedStyle(footer).borderTopWidth);
      expect(hairline).toBeGreaterThan(0);
      expect(f.left).toBeCloseTo(picker.left, 0);
      expect(f.right).toBeCloseTo(picker.right, 0);
      const item = page
        .getByRole("button", { name: "Create palette" })
        .element()
        .getBoundingClientRect();
      const left = item.left - picker.left;
      expect(left).toBeGreaterThan(0);
      expect(picker.right - item.right).toBeCloseTo(left, 0);
      expect(picker.bottom - item.bottom).toBeCloseTo(left, 0);
      expect(item.top - (f.top + hairline)).toBeCloseTo(left, 0);
    });

    it("should keep the footer at the bottom while the palettes scroll", async () => {
      const picker = await renderLibrary({
        footer: CREATE,
        style: { height: "26rem" },
      });
      const palettes = picker.querySelector(`.${CSS.BE("color-picker", "palettes")}`);
      assert(palettes instanceof HTMLElement);
      expect(palettes.scrollHeight).toBeGreaterThan(palettes.clientHeight);
      const footer = `.${CSS.BE("color-picker", "footer")}`;
      const bottom =
        picker.getBoundingClientRect().bottom -
        Number.parseFloat(getComputedStyle(picker).borderBottomWidth);
      expect(rectOf(footer).bottom).toBeCloseTo(bottom, 0);
      palettes.scrollTop = palettes.scrollHeight;
      await expect.poll(() => palettes.scrollTop).toBeGreaterThan(0);
      expect(rectOf(footer).bottom).toBeCloseTo(bottom, 0);
    });

    it("should give what replaces the footer's create item the same box", async () => {
      const box = async (footer: ReactNode): Promise<number[]> => {
        const picker = await renderLibrary({ footer });
        const p = picker.getBoundingClientRect();
        const child = picker.querySelector(`.${CSS.BE("color-picker", "footer")} > *`);
        assert(child instanceof HTMLElement);
        const c = child.getBoundingClientRect();
        return [c.left - p.left, p.right - c.right, p.bottom - c.bottom, c.height];
      };
      const entry = (
        <Flex.Box x align="center">
          <Input.Text
            value=""
            onChange={() => {}}
            size="small"
            aria-label="Palette name"
          />
        </Flex.Box>
      );
      expect(await box(entry)).toEqual(await box(CREATE));
    });
  });
});
