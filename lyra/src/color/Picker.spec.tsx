// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { act, fireEvent, render, type RenderResult } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement, useState } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Color } from "@/color";
import { CSS } from "@/css";
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
      const c = renderPicker({ value: [255, 0, 0, 0.5], onChange });
      fireEvent.change(hexInput(c), { target: { value: "0000ff" } });
      expect(onChange.mock.calls[0][0]).toEqual([0, 0, 255, 0.5]);
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
      expect(onChange.mock.calls[0][0]).toEqual([0, 0, 255, 0.5]);
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
      expect(onChange.mock.calls.at(-1)?.[0]).toEqual([255, 0, 0, 0.25]);
    });
  });

  describe("swatches", () => {
    it("should pick a preset with one click", () => {
      const onChange = vi.fn();
      const c = renderPicker({ value: "#ff0000", onChange });
      const [first] = c.container.querySelectorAll<HTMLElement>(
        `.${CSS.BE("color-picker", "swatch")}`,
      );
      fireEvent.click(first);
      expect(lastHex(onChange)).toEqual(first.getAttribute("aria-label"));
    });

    it("should mark the swatch that matches the value", () => {
      const c = renderPicker({ value: "#DC136C", onChange: vi.fn() });
      expect(c.getByLabelText("#dc136c").className).toContain(CSS.M("selected"));
    });
  });

  describe("auto", () => {
    it("should not offer Auto without a fallback", () => {
      const c = renderPicker({ value: "#ff0000", onChange: vi.fn() });
      expect(c.queryByLabelText("Auto")).toBeNull();
    });

    it("should clear the value when the user picks Auto", () => {
      const onChange = vi.fn();
      const c = renderPicker({ value: "#ff0000", fallback: "#888888", onChange });
      fireEvent.click(c.getByLabelText("Auto"));
      expect(onChange).toHaveBeenCalledExactlyOnceWith(undefined);
    });

    it("should mark Auto and show the fallback while the value is absent", () => {
      const c = renderPicker({ fallback: "#888888", onChange: vi.fn() });
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
      expect(color.aValue(onChange.mock.calls[0][0])).toBeCloseTo(0.99);
    });

    it("should make a transparent color opaque when the plane moves", () => {
      const onChange = vi.fn();
      const c = renderPicker({ value: color.ZERO, onChange });
      fireEvent.keyDown(c.getByRole("slider", { name: "Saturation and brightness" }), {
        key: "ArrowUp",
      });
      expect(color.aValue(onChange.mock.calls[0][0])).toEqual(1);
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
});
