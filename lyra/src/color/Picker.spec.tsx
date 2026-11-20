// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import {
  act,
  fireEvent,
  render,
  type RenderResult,
  within,
} from "@testing-library/react";
import {
  type PropsWithChildren,
  type ReactElement,
  useEffect,
  useRef,
  useState,
} from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Color } from "@/color";
import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { Menu } from "@/menu";
import { Theming } from "@/theming";
import { Triggers } from "@/triggers";

const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
  <Triggers.Provider>
    <Theming.Provider>{children}</Theming.Provider>
  </Triggers.Provider>
);

const renderPicker = (props: Color.PickerProps): RenderResult =>
  render(<Color.Picker {...props} />, { wrapper: Wrapper });

const hexInput = (c: RenderResult): HTMLInputElement =>
  c.getByLabelText("Hex") as HTMLInputElement;

const lastHex = (fn: ReturnType<typeof vi.fn>): string | undefined =>
  color.hex(fn.mock.calls.at(-1)?.[0]);

/** A picker that feeds every change back in as its value, like a real caller. */
const Controlled = ({
  initial,
  onChange,
}: {
  initial: color.Crude;
  onChange: (c?: color.Color) => void;
}): ReactElement => {
  const [value, setValue] = useState<color.Crude>(initial);
  return (
    <Color.Picker
      value={value}
      onChange={(c) => {
        if (c != null) setValue(c);
        onChange(c);
      }}
    />
  );
};

describe("Picker", () => {
  describe("hex input", () => {
    it("should show the color as six hex digits without a hash", () => {
      const c = renderPicker({ value: "#3e8bff80", onChange: vi.fn() });
      expect(hexInput(c).value).toEqual("3e8bff");
    });

    it("should apply a complete hex as the user types", () => {
      const onChange = vi.fn();
      const c = renderPicker({ value: "#ff0000", onChange });
      fireEvent.change(hexInput(c), { target: { value: "00ff00" } });
      expect(lastHex(onChange)).toEqual("#00ff00");
    });

    it("should wait for Enter to apply a three digit hex", () => {
      const onChange = vi.fn();
      const c = renderPicker({ value: "#ff0000", onChange });
      fireEvent.change(hexInput(c), { target: { value: "0f0" } });
      expect(onChange).not.toHaveBeenCalled();
      fireEvent.keyDown(hexInput(c), { key: "Enter" });
      expect(lastHex(onChange)).toEqual("#00ff00");
    });

    it("should keep the current alpha when the hex has no alpha digits", () => {
      const onChange = vi.fn();
      const c = renderPicker({ value: [255, 0, 0, 128], onChange });
      fireEvent.change(hexInput(c), { target: { value: "0000ff" } });
      expect(onChange.mock.calls[0][0]).toEqual([0, 0, 255, 128]);
    });

    it("should make a typed hex opaque when the color is transparent", () => {
      const onChange = vi.fn();
      const c = renderPicker({
        value: undefined,
        fallback: color.TRANSPARENT,
        onChange,
      });
      fireEvent.change(hexInput(c), { target: { value: "0000ff" } });
      expect(color.opacity(onChange.mock.calls[0][0])).toEqual(1);
    });

    it("should take the alpha from an eight digit hex", () => {
      const onChange = vi.fn();
      const c = renderPicker({ value: "#ff0000", onChange });
      fireEvent.change(hexInput(c), { target: { value: "0000ff00" } });
      expect(onChange.mock.calls[0][0]).toEqual([0, 0, 255, 0]);
    });

    it("should apply a pasted CSS color", () => {
      const onChange = vi.fn();
      const c = renderPicker({ value: "#ff0000", onChange });
      fireEvent.paste(hexInput(c), {
        clipboardData: { getData: () => "rgb(0 0 255 / 50%)" },
      });
      expect(onChange.mock.calls[0][0]).toEqual([0, 0, 255, 128]);
    });

    it("should restore the current color when the input is not a color", () => {
      const onChange = vi.fn();
      const c = renderPicker({ value: "#ff0000", onChange });
      fireEvent.change(hexInput(c), { target: { value: "zz" } });
      fireEvent.blur(hexInput(c));
      expect(onChange).not.toHaveBeenCalled();
      expect(hexInput(c).value).toEqual("ff0000");
    });
  });

  describe("alpha input", () => {
    it("should set the alpha as a percentage", () => {
      const onChange = vi.fn();
      const c = renderPicker({ value: "#ff0000", onChange });
      const input = c.getByLabelText("Alpha percentage") as HTMLInputElement;
      fireEvent.change(input, { target: { value: "25" } });
      fireEvent.keyDown(input, { key: "Enter", code: "Enter" });
      expect(onChange.mock.calls.at(-1)?.[0]).toEqual([255, 0, 0, 64]);
    });
  });

  describe("swatches", () => {
    it("should pick a preset with one click", () => {
      const onChange = vi.fn();
      const c = renderPicker({ value: "#ff0000", onChange });
      const [first] = c.container.querySelectorAll<HTMLElement>(
        `.${CSS.BE("color-swatches", "grid")} .${CSS.B("color-swatch")}`,
      );
      fireEvent.click(first);
      expect(lastHex(onChange)).toEqual(first.getAttribute("aria-label"));
    });
  });

  describe("auto", () => {
    it("should clear the value when the user picks Auto", () => {
      const onChange = vi.fn();
      const c = renderPicker({ value: "#ff0000", fallback: "#888888", onChange });
      fireEvent.click(c.getByLabelText("Auto"));
      expect(onChange).toHaveBeenCalledExactlyOnceWith(undefined);
    });

    it("should mark Auto and show the fallback while the value is absent", () => {
      const c = renderPicker({
        value: undefined,
        fallback: "#888888",
        onChange: vi.fn(),
      });
      expect(c.getByLabelText("Auto").className).toContain(CSS.M("selected"));
      expect(hexInput(c).value).toEqual("888888");
    });
  });

  describe("sliders", () => {
    it("should move the hue with the arrow keys", () => {
      const onChange = vi.fn();
      const c = renderPicker({ value: "#ff0000", onChange });
      fireEvent.keyDown(c.getByRole("slider", { name: "Hue" }), {
        key: "ArrowRight",
        shiftKey: true,
      });
      const [h] = color.hsva(onChange.mock.calls[0][0]);
      expect(h).toBeCloseTo(36, 0);
    });

    it("should move the alpha with the arrow keys", () => {
      const onChange = vi.fn();
      const c = renderPicker({ value: "#ff0000", onChange });
      fireEvent.keyDown(c.getByRole("slider", { name: "Alpha" }), {
        key: "ArrowLeft",
      });
      expect(color.opacity(onChange.mock.calls[0][0])).toBeCloseTo(0.99);
    });

    it("should make a transparent color opaque when the plane moves", () => {
      const onChange = vi.fn();
      const c = renderPicker({ value: color.TRANSPARENT, onChange });
      fireEvent.keyDown(c.getByRole("slider", { name: "Saturation and brightness" }), {
        key: "ArrowUp",
      });
      expect(color.opacity(onChange.mock.calls[0][0])).toEqual(1);
    });

    it("should keep the hue through a move to gray", () => {
      const onChange = vi.fn();
      const c = render(<Controlled initial="#0000ff" onChange={onChange} />, {
        wrapper: Wrapper,
      });
      const plane = c.getByRole("slider", { name: "Saturation and brightness" });
      for (let i = 0; i < 10; i++)
        fireEvent.keyDown(plane, { key: "ArrowLeft", shiftKey: true });
      expect(lastHex(onChange)).toEqual("#ffffff");
      fireEvent.keyDown(plane, { key: "ArrowRight", shiftKey: true });
      const [h] = color.hsva(onChange.mock.calls.at(-1)?.[0]);
      expect(h).toBeCloseTo(240, 0);
    });
  });

  describe("recent colors", () => {
    it("should show a color the user picked after the debounce", () => {
      vi.useFakeTimers();
      const c = render(
        <Color.Provider>
          <Controlled initial="#ff0000" onChange={vi.fn()} />
        </Color.Provider>,
        { wrapper: Wrapper },
      );
      expect(c.queryByLabelText("#123456")).toBeNull();
      fireEvent.change(hexInput(c), { target: { value: "123456" } });
      act(() => {
        vi.advanceTimersByTime(1500);
      });
      expect(c.getByLabelText("#123456")).toBeTruthy();
      vi.useRealTimers();
    });

    it("should show a picked preset once", () => {
      vi.useFakeTimers();
      const c = render(
        <Color.Provider presets={["#20A4F3"]}>
          <Controlled initial="#ff0000" onChange={vi.fn()} />
        </Color.Provider>,
        { wrapper: Wrapper },
      );
      fireEvent.click(c.getByLabelText("#20a4f3"));
      act(() => {
        vi.advanceTimersByTime(1500);
      });
      expect(c.getAllByLabelText("#20a4f3")).toHaveLength(1);
      vi.useRealTimers();
    });
  });

  describe("eyedropper", () => {
    afterEach(() => {
      vi.unstubAllGlobals();
    });

    it("should not render where the platform has no EyeDropper", () => {
      const c = renderPicker({ value: "#ff0000", onChange: vi.fn() });
      expect(c.queryByLabelText("Pick a color from the screen")).toBeNull();
    });

    it("should apply the color the platform picks", async () => {
      vi.stubGlobal(
        "EyeDropper",
        class {
          open = async () => ({ sRGBHex: "#00ff00" });
        },
      );
      const onChange = vi.fn();
      const c = renderPicker({ value: "#ff0000", onChange });
      await act(async () => {
        fireEvent.click(c.getByLabelText("Pick a color from the screen"));
      });
      expect(lastHex(onChange)).toEqual("#00ff00");
    });
  });
  describe("library", () => {
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
    const SWATCH_VAR = CSS.variable("swatch", "color");

    const renderLibrary = (
      props: Partial<Color.PickerProps> & { named?: string },
      wrapper = Wrapper,
    ): RenderResult & {
      onChange: ReturnType<typeof vi.fn>;
      onNamedChange: ReturnType<typeof vi.fn>;
    } => {
      const onChange = vi.fn();
      const onNamedChange = vi.fn();
      const { named, ...rest } = props;
      const c = render(
        <Color.Picker
          value={undefined}
          onChange={onChange}
          library={{ palettes: PALETTES, named, onNamedChange }}
          {...rest}
        />,
        { wrapper },
      );
      return { ...c, onChange, onNamedChange };
    };

    const themed =
      (key: string) =>
      ({ children }: PropsWithChildren): ReactElement => (
        <Triggers.Provider>
          <Theming.Provider theme={{ key }}>{children}</Theming.Provider>
        </Triggers.Provider>
      );

    const search = (c: RenderResult, term: string): void => {
      fireEvent.change(c.getByPlaceholderText("Search colors"), {
        target: { value: term },
      });
    };

    describe("tabs", () => {
      it("should not show tabs without a library", () => {
        const c = renderPicker({ value: "#ff0000", onChange: vi.fn() });
        expect(c.queryByText("Library")).toBeNull();
        expect(c.queryByText("Custom")).toBeNull();
      });

      it("should open on Library while the value follows a named color", () => {
        const c = renderLibrary({ value: "#3e8bff", named: "ocean" });
        expect(c.getByLabelText("Ocean")).toBeTruthy();
        expect(c.queryByLabelText("Hex")).toBeNull();
      });

      it("should open on Library while the value is absent", () => {
        const c = renderLibrary({ value: undefined });
        expect(c.getByLabelText("Ocean")).toBeTruthy();
      });

      it("should open on Custom while the value is a fixed color", () => {
        const c = renderLibrary({ value: "#123456" });
        expect(hexInput(c).value).toEqual("123456");
        expect(c.queryByLabelText("Ocean")).toBeNull();
      });

      it("should pick a recent color in Custom", () => {
        vi.useFakeTimers();
        const Recent = ({ children }: PropsWithChildren): ReactElement => (
          <Wrapper>
            <Color.Provider>{children}</Color.Provider>
          </Wrapper>
        );
        const c = renderLibrary({ value: "#123456" }, Recent);
        fireEvent.change(hexInput(c), { target: { value: "abcdef" } });
        act(() => {
          vi.advanceTimersByTime(1500);
        });
        c.onChange.mockClear();
        fireEvent.click(c.getByLabelText("#abcdef"));
        expect(lastHex(c.onChange)).toEqual("#abcdef");
        vi.useRealTimers();
      });

      describe("recent colors", () => {
        /** A caller that unlinks the value on each fixed color, as a real one does. */
        const Linked = (): ReactElement => {
          const [value, setValue] = useState<color.Crude | undefined>("#3e8bff");
          const [named, setNamed] = useState<string | undefined>("ocean");
          return (
            <Color.Picker
              value={value}
              onChange={(c) => {
                setValue(c);
                setNamed(undefined);
              }}
              library={{ palettes: PALETTES, named, onNamedChange: setNamed }}
            />
          );
        };

        const renderWithRecent = (): RenderResult => {
          vi.useFakeTimers();
          const c = render(
            <Color.Provider>
              <Linked />
            </Color.Provider>,
            { wrapper: Wrapper },
          );
          fireEvent.click(c.getByText("Custom"));
          fireEvent.change(hexInput(c), { target: { value: "abcdef" } });
          act(() => {
            vi.advanceTimersByTime(1500);
          });
          return c;
        };

        afterEach(() => {
          vi.useRealTimers();
        });

        it("should mark the recent color equal to a fixed value", () => {
          const c = renderWithRecent();
          expect(c.getByLabelText("#abcdef").className).toContain(CSS.M("selected"));
        });

        it("should not mark a recent color while the value is linked", () => {
          const c = renderWithRecent();
          fireEvent.click(c.getByText("Library"));
          fireEvent.click(c.getByLabelText("Ocean"));
          fireEvent.click(c.getByText("Custom"));
          expect(c.getByLabelText("#abcdef").className).not.toContain(
            CSS.M("selected"),
          );
        });
      });

      it("should edit a fixed color in Custom", () => {
        const c = renderLibrary({ value: "#3e8bff", named: "ocean" });
        fireEvent.click(c.getByText("Custom"));
        fireEvent.change(hexInput(c), { target: { value: "00ff00" } });
        expect(lastHex(c.onChange)).toEqual("#00ff00");
        expect(c.onNamedChange).not.toHaveBeenCalled();
      });
    });

    describe("named colors", () => {
      it("should link the value with one click on a named color", () => {
        const c = renderLibrary({ value: "#3e8bff", named: "ocean" });
        fireEvent.click(c.getByLabelText("Moss"));
        expect(c.onNamedChange).toHaveBeenCalledExactlyOnceWith("moss");
        expect(c.onChange).not.toHaveBeenCalled();
      });

      it("should mark the linked color", () => {
        const c = renderLibrary({ value: "#3e8bff", named: "ocean" });
        expect(c.getByLabelText("Ocean").className).toContain(CSS.M("selected"));
        expect(c.getByLabelText("Moss").className).not.toContain(CSS.M("selected"));
      });

      it("should paint a color with its light value in a light theme", () => {
        const c = renderLibrary({ named: "ocean" }, themed("synnaxLight"));
        expect(c.getByLabelText("Ocean").style.getPropertyValue(SWATCH_VAR)).toEqual(
          color.cssString("#3e8bff"),
        );
      });

      it("should paint a color with its dark value in a dark theme", () => {
        const c = renderLibrary({ named: "ocean" }, themed("synnaxDark"));
        expect(c.getByLabelText("Ocean").style.getPropertyValue(SWATCH_VAR)).toEqual(
          color.cssString("#1a4f99"),
        );
      });
    });

    describe("auto", () => {
      it("should not offer Auto without a fallback", () => {
        const c = renderLibrary({});
        expect(c.queryByLabelText("Auto")).toBeNull();
      });

      it("should clear the value when the user picks Auto", () => {
        const c = renderLibrary({
          value: "#3e8bff",
          named: "ocean",
          fallback: "#888888",
        });
        fireEvent.click(c.getByLabelText("Auto"));
        expect(c.onChange).toHaveBeenCalledExactlyOnceWith(undefined);
        expect(c.onNamedChange).not.toHaveBeenCalled();
      });

      it("should mark Auto while the value is absent", () => {
        const c = renderLibrary({ fallback: "#888888" });
        expect(c.getByLabelText("Auto").className).toContain(CSS.M("selected"));
      });

      it("should offer Auto on the Custom tab", () => {
        const c = renderLibrary({ value: "#123456", fallback: "#888888" });
        expect(hexInput(c).value).toEqual("123456");
        fireEvent.click(c.getByLabelText("Auto"));
        expect(c.onChange).toHaveBeenCalledExactlyOnceWith(undefined);
      });
    });

    describe("palette titles", () => {
      it("should end the title with the name of the linked color", () => {
        const c = renderLibrary({ value: "#3e8bff", named: "ocean" });
        expect(c.getByText("Ocean")).toBeTruthy();
      });

      it("should show the hovered color in place of the linked one", () => {
        const c = renderLibrary({ value: "#3e8bff", named: "ocean" });
        fireEvent.pointerEnter(c.getByLabelText("Moss"));
        expect(c.getByText("Moss")).toBeTruthy();
        expect(c.queryByText("Ocean")).toBeNull();
      });

      it("should keep the hovered color while the pointer stays in the palette", () => {
        const c = renderLibrary({ value: "#3e8bff", named: "ocean" });
        fireEvent.pointerEnter(c.getByLabelText("Moss"));
        fireEvent.pointerLeave(c.getByLabelText("Moss"), {
          relatedTarget: c.getByRole("group", { name: "Series" }),
        });
        expect(c.getByText("Moss")).toBeTruthy();
      });

      it("should return to the linked color when the pointer leaves the palette", () => {
        const c = renderLibrary({ value: "#3e8bff", named: "ocean" });
        fireEvent.pointerEnter(c.getByLabelText("Moss"));
        fireEvent.pointerLeave(c.getByRole("group", { name: "Series" }));
        expect(c.getByText("Ocean")).toBeTruthy();
      });

      it("should name a color that takes the keyboard focus", () => {
        const c = renderLibrary({ value: "#3e8bff", named: "ocean" });
        fireEvent.focus(c.getByLabelText("Moss"));
        expect(c.getByText("Moss")).toBeTruthy();
      });

      it("should keep the focused color after the pointer leaves the palette", () => {
        const c = renderLibrary({ value: "#3e8bff", named: "ocean" });
        fireEvent.focus(c.getByLabelText("Moss"));
        fireEvent.pointerLeave(c.getByRole("group", { name: "Series" }));
        expect(c.getByText("Moss")).toBeTruthy();
      });

      it("should keep the hovered color after the focus leaves the palette", () => {
        const c = renderLibrary({ value: "#3e8bff", named: "ocean" });
        fireEvent.pointerEnter(c.getByLabelText("Moss"));
        fireEvent.focus(c.getByLabelText("Moss"));
        fireEvent.blur(c.getByLabelText("Moss"), {
          relatedTarget: c.getByLabelText("Alarm"),
        });
        expect(c.getByText("Moss")).toBeTruthy();
      });

      it("should return to the linked color when the focus leaves the palette", () => {
        const c = renderLibrary({ value: "#3e8bff", named: "ocean" });
        fireEvent.focus(c.getByLabelText("Moss"));
        fireEvent.blur(c.getByLabelText("Moss"), {
          relatedTarget: c.getByLabelText("Alarm"),
        });
        expect(c.getByText("Ocean")).toBeTruthy();
      });
    });

    describe("slots", () => {
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

    describe("search", () => {
      it("should filter colors by name", () => {
        const c = renderLibrary({});
        search(c, "mos");
        expect(c.getByLabelText("Moss")).toBeTruthy();
        expect(c.queryByLabelText("Ocean")).toBeNull();
        expect(c.queryByText("Status")).toBeNull();
      });

      it("should keep every color of a palette whose name matches", () => {
        const c = renderLibrary({});
        search(c, "status");
        expect(c.getByLabelText("Alarm")).toBeTruthy();
        expect(c.queryByLabelText("Ocean")).toBeNull();
      });

      it("should filter colors by the hex they show", () => {
        const c = renderLibrary({});
        search(c, "#4caf");
        expect(c.getByLabelText("Moss")).toBeTruthy();
        expect(c.queryByLabelText("Ocean")).toBeNull();
      });

      it("should say when the library has no colors", () => {
        const c = render(
          <Color.Picker
            value={undefined}
            onChange={vi.fn()}
            library={{ palettes: [], onNamedChange: vi.fn() }}
          />,
          { wrapper: Wrapper },
        );
        expect(c.getByText("No colors found")).toBeTruthy();
      });

      it("should say when no color matches", () => {
        const c = renderLibrary({});
        search(c, "zzz");
        expect(c.getByText('No colors match "zzz"')).toBeTruthy();
      });
    });
  });
});
