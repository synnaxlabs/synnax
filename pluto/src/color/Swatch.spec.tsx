// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { fireEvent, render, type RenderResult } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Color } from "@/color";
import { CSS } from "@/css";
import { mockBoundingClientRect } from "@/testutil/dom";
import { Theming } from "@/theming";
import { Triggers } from "@/triggers";

const RED = "#ff0000";
const GREEN = "00ff00";
const BLUE = "0000ff";

const SWATCH_VAR = CSS.variable("swatch", "color");

const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
  <Triggers.Provider>
    <Theming.Provider>{children}</Theming.Provider>
  </Triggers.Provider>
);

const swatchOf = (c: RenderResult): HTMLElement => {
  const el = c.container.querySelector<HTMLElement>(`.${CSS.B("color-swatch")}`);
  if (el == null) throw new Error("no swatch rendered");
  return el;
};

const hexInputOf = (c: RenderResult): HTMLInputElement =>
  c.getByLabelText("hex") as HTMLInputElement;

const pick = (c: RenderResult, hex: string): void => {
  fireEvent.change(hexInputOf(c), { target: { value: hex } });
};

const closePicker = (c: RenderResult): void => {
  fireEvent.keyDown(c.container, { code: "Escape" });
};

describe("Swatch", () => {
  describe("onChange", () => {
    it("should call onChange on every picker change by default", () => {
      const onChange = vi.fn();
      const c = render(<Color.Swatch value={RED} onChange={onChange} />, {
        wrapper: Wrapper,
      });
      fireEvent.click(swatchOf(c));
      pick(c, GREEN);
      pick(c, BLUE);
      expect(onChange).toHaveBeenCalledTimes(2);
      expect(color.hex(onChange.mock.calls[1][0])).toEqual(`#${BLUE}`);
    });
  });

  describe("onClear", () => {
    it("should render an unset swatch", () => {
      const c = render(<Color.Swatch value={undefined} onChange={vi.fn()} />, {
        wrapper: Wrapper,
      });
      expect(swatchOf(c).classList.contains(CSS.M("unset"))).toBe(true);
    });

    it("should not show the clear button while the swatch is unset", () => {
      const c = render(
        <Color.Swatch value={undefined} onChange={vi.fn()} onClear={vi.fn()} />,
        { wrapper: Wrapper },
      );
      fireEvent.click(swatchOf(c));
      expect(c.queryByText("Clear")).toBeNull();
    });

    it("should call onClear and close the picker", () => {
      const onClear = vi.fn();
      const c = render(
        <Color.Swatch value={RED} onChange={vi.fn()} onClear={onClear} />,
        { wrapper: Wrapper },
      );
      fireEvent.click(swatchOf(c));
      fireEvent.click(c.getByText("Clear"));
      expect(onClear).toHaveBeenCalledOnce();
      expect(c.queryByLabelText("hex")).toBeNull();
    });
  });

  describe("placeholder", () => {
    it("should show the placeholder while the swatch is unset", () => {
      const c = render(
        <Color.Swatch value={undefined} placeholder={RED} onChange={vi.fn()} />,
        { wrapper: Wrapper },
      );
      expect(swatchOf(c).style.getPropertyValue(SWATCH_VAR)).toEqual(
        color.cssString(RED),
      );
    });

    it("should open the picker on the placeholder while the swatch is unset", () => {
      const c = render(
        <Color.Swatch value={undefined} placeholder={RED} onChange={vi.fn()} />,
        { wrapper: Wrapper },
      );
      fireEvent.click(swatchOf(c));
      expect(hexInputOf(c).value).toEqual("FF0000");
    });

    it("should open the picker on the value when the swatch is set", () => {
      const c = render(
        <Color.Swatch value={`#${BLUE}`} placeholder={RED} onChange={vi.fn()} />,
        { wrapper: Wrapper },
      );
      fireEvent.click(swatchOf(c));
      expect(hexInputOf(c).value).toEqual("0000FF");
    });

    it("should open the picker opaque while the swatch is unset", () => {
      const c = render(
        <Color.Swatch value={undefined} placeholder={RED} onChange={vi.fn()} />,
        { wrapper: Wrapper },
      );
      fireEvent.click(swatchOf(c));
      expect((c.getByLabelText("a") as HTMLInputElement).value).toEqual("100");
    });

    it("should default to the theme's emphatic gray", () => {
      const c = render(
        <Triggers.Provider>
          <Theming.Provider theme={{ key: "synnaxLight" }}>
            <Color.Swatch value={undefined} onChange={vi.fn()} />
          </Theming.Provider>
        </Triggers.Provider>,
      );
      const gray = Theming.themeZ.parse(Theming.SYNNAX_LIGHT).colors.gray.l11;
      expect(swatchOf(c).style.getPropertyValue(SWATCH_VAR)).toEqual(
        color.cssString(gray),
      );
    });
  });

  describe("onlyChangeOnBlur", () => {
    it("should not call onChange while the picker is open", () => {
      const onChange = vi.fn();
      const c = render(
        <Color.Swatch value={RED} onChange={onChange} onlyChangeOnBlur />,
        { wrapper: Wrapper },
      );
      fireEvent.click(swatchOf(c));
      pick(c, GREEN);
      pick(c, BLUE);
      expect(onChange).not.toHaveBeenCalled();
    });

    it("should show the pending color on the swatch while the picker is open", () => {
      const c = render(
        <Color.Swatch value={RED} onChange={vi.fn()} onlyChangeOnBlur />,
        { wrapper: Wrapper },
      );
      fireEvent.click(swatchOf(c));
      pick(c, GREEN);
      expect(swatchOf(c).style.getPropertyValue(SWATCH_VAR)).toEqual(
        color.cssString(`#${GREEN}`),
      );
    });

    it("should call onChange once with the last color when the picker closes", () => {
      const onChange = vi.fn();
      const c = render(
        <Color.Swatch value={RED} onChange={onChange} onlyChangeOnBlur />,
        { wrapper: Wrapper },
      );
      fireEvent.click(swatchOf(c));
      pick(c, GREEN);
      pick(c, BLUE);
      closePicker(c);
      expect(onChange).toHaveBeenCalledOnce();
      expect(color.hex(onChange.mock.calls[0][0])).toEqual(`#${BLUE}`);
    });

    it("should not call onChange when the picker closes without a change", () => {
      const onChange = vi.fn();
      const c = render(
        <Color.Swatch value={RED} onChange={onChange} onlyChangeOnBlur />,
        { wrapper: Wrapper },
      );
      fireEvent.click(swatchOf(c));
      closePicker(c);
      expect(onChange).not.toHaveBeenCalled();
    });

    it("should drop a pending change when the swatch unmounts", () => {
      const onChange = vi.fn();
      const c = render(
        <Color.Swatch value={RED} onChange={onChange} onlyChangeOnBlur />,
        { wrapper: Wrapper },
      );
      fireEvent.click(swatchOf(c));
      pick(c, GREEN);
      c.unmount();
      expect(onChange).not.toHaveBeenCalled();
    });

    // Dismissal reads the pointer against getBoundingClientRect, which jsdom reports
    // as all zeros: the origin lands inside the picker and every other point lands
    // outside the window. Give the document a size so the dismissal runs.
    describe("dismissed by a click outside", () => {
      beforeEach(() => {
        vi.spyOn(document.documentElement, "getBoundingClientRect").mockImplementation(
          mockBoundingClientRect(0, 0, 1000, 1000),
        );
      });
      afterEach(() => {
        vi.restoreAllMocks();
      });

      it("should call onChange once with the last color", () => {
        const onChange = vi.fn();
        const c = render(
          <Color.Swatch value={RED} onChange={onChange} onlyChangeOnBlur />,
          { wrapper: Wrapper },
        );
        fireEvent.click(swatchOf(c));
        pick(c, GREEN);
        pick(c, BLUE);
        fireEvent.pointerDown(document.body, { clientX: 500, clientY: 500 });
        expect(onChange).toHaveBeenCalledOnce();
        expect(color.hex(onChange.mock.calls[0][0])).toEqual(`#${BLUE}`);
      });
    });
  });
});
