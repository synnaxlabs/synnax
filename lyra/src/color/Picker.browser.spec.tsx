// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
// The pure entry leaves the act environment of the browser specs alone.
import {
  act,
  cleanup,
  fireEvent,
  render,
  type RenderResult,
  within,
} from "@testing-library/react/pure";
import {
  type PropsWithChildren,
  type ReactElement,
  type ReactNode,
  useEffect,
  useRef,
} from "react";
import { afterEach, assert, describe, expect, it, vi } from "vitest";
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
import { Theming } from "@/theming";
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
  afterEach(cleanup);

  describe("library", () => {
    describe("slots", () => {
      const PALETTES: Color.Palette[] = [
        {
          key: "series",
          name: "Series",
          colors: [
            { key: "ocean", name: "Ocean", light: "#3e8bff", dark: "#1a4f99" },
            { key: "moss", name: "Moss", light: "#4caf50" },
          ],
        },
        {
          key: "status",
          name: "Status",
          colors: [{ key: "alarm", name: "Alarm", light: "#ff0000" }],
        },
      ];

      const search = (c: RenderResult, term: string): void => {
        fireEvent.change(c.getByPlaceholderText("Search colors"), {
          target: { value: term },
        });
      };

      const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
        <Triggers.Provider>
          <Theming.Provider>{children}</Theming.Provider>
        </Triggers.Provider>
      );

      const PaletteEnd = (): ReactElement => {
        const { palette } = Color.usePalette();
        return <span>{`end of ${palette.name}`}</span>;
      };

      const SwatchMenu = (): ReactElement => {
        const { palette, swatch } = Color.usePalette();
        return <span>{`menu for ${swatch?.name} in ${palette.name}`}</span>;
      };

      const ViewOpener = (): ReactElement => {
        const { value } = Color.usePicker();
        const { open } = Dialog.useViews();
        return (
          <button onClick={() => open("Edit color", <span>view body</span>)}>
            {`open from ${color.hex(value)}`}
          </button>
        );
      };

      const EditMenu = (): ReactElement => {
        const { open } = Dialog.useViews();
        return (
          <Menu.Menu onChange={() => open("Edit color", <span>view body</span>)}>
            <Menu.Item itemKey="edit">Edit</Menu.Item>
          </Menu.Menu>
        );
      };

      const renderSlots = (
        library: Partial<Color.Library>,
        props: Partial<Color.PickerProps> = {},
      ): RenderResult =>
        render(
          <Color.Picker
            value={undefined}
            onChange={vi.fn()}
            library={{ palettes: PALETTES, onNamedChange: vi.fn(), ...library }}
            {...props}
          />,
          { wrapper: Wrapper },
        );

      it("should end each palette with paletteEnd, given that palette", () => {
        const c = renderSlots({ paletteEnd: <PaletteEnd /> });
        const series = c.getByRole("group", { name: "Series" });
        const status = c.getByRole("group", { name: "Status" });
        expect(within(series).getByText("end of Series")).toBeTruthy();
        expect(within(status).getByText("end of Status")).toBeTruthy();
      });

      it("should show a palette with no colors only while it has a paletteEnd", () => {
        const palettes = [...PALETTES, { key: "empty", name: "Empty", colors: [] }];
        const without = renderSlots({ palettes });
        expect(without.queryByRole("group", { name: "Empty" })).toBeNull();
        without.unmount();
        const c = renderSlots({ palettes, paletteEnd: <PaletteEnd /> });
        expect(c.getByText("end of Empty")).toBeTruthy();
        search(c, "ocean");
        expect(c.queryByRole("group", { name: "Empty" })).toBeNull();
      });

      it("should open swatchMenu on a right click, given its swatch", () => {
        const c = renderSlots({ named: "ocean", swatchMenu: <SwatchMenu /> });
        fireEvent.contextMenu(c.getByLabelText("Alarm"));
        expect(c.getByRole("menu")).toBeTruthy();
        expect(c.getByText("menu for Alarm in Status")).toBeTruthy();
      });

      it("should name the swatch of an open menu in the palette title", () => {
        const c = renderSlots({ swatchMenu: <SwatchMenu /> });
        fireEvent.contextMenu(c.getByLabelText("Alarm"));
        expect(c.getByText("Alarm")).toBeTruthy();
      });

      it("should open swatchMenu on a press of Shift+F10", () => {
        const c = renderSlots({ swatchMenu: <SwatchMenu /> });
        const alarm = c.getByLabelText("Alarm");
        const opened = fireEvent.keyDown(alarm, {
          key: "F10",
          code: "F10",
          shiftKey: true,
        });
        expect(opened).toBe(false);
        expect(c.getByText("menu for Alarm in Status")).toBeTruthy();
      });

      it("should open swatchMenu on a release of the Menu key", () => {
        const c = renderSlots({ swatchMenu: <SwatchMenu /> });
        const moss = c.getByLabelText("Moss");
        const key = { key: "ContextMenu", code: "ContextMenu" };
        expect(fireEvent.keyDown(moss, key)).toBe(false);
        expect(c.queryByRole("menu")).toBeNull();
        fireEvent.keyUp(moss, key);
        expect(c.getByText("menu for Moss in Series")).toBeTruthy();
      });

      const OPENERS: [string, (swatch: HTMLElement) => boolean][] = [
        ["a right click", (swatch) => fireEvent.contextMenu(swatch)],
        [
          "Shift+F10",
          (swatch) =>
            fireEvent.keyDown(swatch, { key: "F10", code: "F10", shiftKey: true }),
        ],
        [
          "the Menu key",
          (swatch) => {
            const key = { key: "ContextMenu", code: "ContextMenu" };
            const down = fireEvent.keyDown(swatch, key);
            return fireEvent.keyUp(swatch, key) && down;
          },
        ],
      ];

      const onlyAlarm = (key: string): boolean => key === "alarm";

      it.each(OPENERS)(
        "should open swatchMenu on %s where hasSwatchMenu allows",
        (_, open) => {
          const c = renderSlots({
            swatchMenu: <SwatchMenu />,
            hasSwatchMenu: onlyAlarm,
          });
          expect(open(c.getByLabelText("Alarm"))).toBe(false);
          expect(c.getByText("menu for Alarm in Status")).toBeTruthy();
        },
      );

      it.each(OPENERS)(
        "should leave %s to the platform where hasSwatchMenu refuses",
        (_, open) => {
          const c = renderSlots({
            swatchMenu: <SwatchMenu />,
            hasSwatchMenu: onlyAlarm,
          });
          const moss = c.getByLabelText("Moss");
          act(() => moss.focus());
          expect(open(moss)).toBe(true);
          expect(c.queryByRole("menu")).toBeNull();
          expect(document.activeElement).toBe(moss);
        },
      );

      it("should leave a right click to the platform without swatchMenu", () => {
        const c = renderSlots({});
        expect(fireEvent.contextMenu(c.getByLabelText("Alarm"))).toBe(true);
        expect(c.queryByRole("menu")).toBeNull();
      });

      it("should end the palettes with the footer", () => {
        const c = renderSlots({ footer: <span>Create palette</span> });
        expect(c.getByText("Create palette")).toBeTruthy();
      });

      it("should end the custom editor with customFooter, and show Library on a link", () => {
        const Linker = (): ReactElement => {
          const { link } = Color.usePicker();
          return <button onClick={() => link("moss")}>link Moss</button>;
        };
        const c = renderSlots(
          { named: "ocean", customFooter: <Linker /> },
          { value: "#3e8bff" },
        );
        expect(c.queryByText("link Moss")).toBeNull();
        fireEvent.click(c.getByText("Custom"));
        fireEvent.click(c.getByText("link Moss"));
        expect(c.queryByLabelText("Hex")).toBeNull();
        expect(document.activeElement).toBe(c.getByLabelText("Moss"));
      });

      it("should give slots the fallback while the value is absent", () => {
        const c = renderSlots({ footer: <ViewOpener /> }, { fallback: "#123456" });
        expect(c.getByText("open from #123456")).toBeTruthy();
      });

      it("should clear the search and focus a hidden swatch that a slot links", () => {
        const Linker = (): ReactElement => {
          const { link } = Color.usePicker();
          return <button onClick={() => link("moss")}>link Moss</button>;
        };
        const c = renderSlots({ named: "ocean", footer: <Linker /> });
        search(c, "oc");
        expect(c.queryByLabelText("Moss")).toBeNull();
        fireEvent.click(c.getByText("link Moss"));
        expect(
          (c.getByPlaceholderText("Search colors") as HTMLInputElement).value,
        ).toBe("");
        expect(document.activeElement).toBe(c.getByLabelText("Moss"));
      });

      it("should focus a shown swatch that a slot links", () => {
        const Linker = (): ReactElement => {
          const { link } = Color.usePicker();
          return <button onClick={() => link("moss")}>link Moss</button>;
        };
        const c = renderSlots({ named: "ocean", footer: <Linker /> });
        fireEvent.click(c.getByText("link Moss"));
        expect(document.activeElement).toBe(c.getByLabelText("Moss"));
      });

      it("should move the focus into the end of a palette that a slot names", () => {
        const End = (): ReactElement => {
          const { palette, focused } = Color.usePalette();
          const ref = useRef<HTMLButtonElement>(null);
          useEffect(() => {
            if (focused) ref.current?.focus();
          }, [focused]);
          return (
            <button ref={ref}>
              {focused ? `${palette.name} end, focused` : `${palette.name} end`}
            </button>
          );
        };
        const Focuser = (): ReactElement => {
          const { focusPalette } = Color.usePicker();
          return <button onClick={() => focusPalette("status")}>focus Status</button>;
        };
        const c = renderSlots({ paletteEnd: <End />, footer: <Focuser /> });
        fireEvent.click(c.getByText("focus Status"));
        // The focus entered the palette, so the flag is clear again.
        expect(document.activeElement).toBe(c.getByText("Status end"));
      });

      it("should clear the search from a slot", () => {
        const Clearer = (): ReactElement => {
          const { clearSearch } = Color.usePicker();
          return <button onClick={clearSearch}>clear</button>;
        };
        const c = renderSlots({ named: "ocean", footer: <Clearer /> });
        search(c, "oc");
        fireEvent.click(c.getByText("clear"));
        expect(
          (c.getByPlaceholderText("Search colors") as HTMLInputElement).value,
        ).toBe("");
        expect(c.getByLabelText("Moss")).toBeTruthy();
      });

      it("should keep the search under a view", () => {
        const c = renderSlots(
          { named: "ocean", footer: <ViewOpener /> },
          { value: "#3e8bff" },
        );
        search(c, "o");
        fireEvent.click(c.getByText("open from #3e8bff"));
        fireEvent.click(c.getByLabelText("Back"));
        expect(
          (c.getByPlaceholderText("Search colors") as HTMLInputElement).value,
        ).toBe("o");
      });

      it("should give a slot the linked color and link the value through the library", () => {
        const onNamedChange = vi.fn();
        const Linker = (): ReactElement => {
          const { linked, link } = Color.usePicker();
          return (
            <button onClick={() => link("moss")}>{`linked ${linked?.name}`}</button>
          );
        };
        const c = renderSlots(
          { named: "ocean", onNamedChange, footer: <Linker /> },
          { value: "#3e8bff" },
        );
        fireEvent.click(c.getByText("linked Ocean"));
        expect(onNamedChange).toHaveBeenCalledExactlyOnceWith("moss");
      });

      it("should focus the swatch after Back from a view its menu opened", () => {
        const c = renderSlots({ named: "ocean", swatchMenu: <EditMenu /> });
        const alarm = c.getByLabelText("Alarm");
        act(() => alarm.focus());
        fireEvent.contextMenu(alarm);
        fireEvent.click(c.getByRole("menuitem", { name: "Edit" }));
        expect(c.queryByRole("menu")).toBeNull();
        expect(document.activeElement).toBe(c.getByLabelText("Back"));
        fireEvent.click(c.getByLabelText("Back"));
        expect(document.activeElement).toBe(alarm);
      });
    });

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
      const picker = await renderLibrary({ swatchMenu: <SwatchMenu /> });
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
