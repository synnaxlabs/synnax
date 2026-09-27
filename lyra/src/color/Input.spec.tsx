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
import { describe, expect, it, type Mock, vi } from "vitest";

import { Color } from "@/color";
import { CSS } from "@/css";
import { Theming } from "@/theming";
import { Triggers } from "@/triggers";

const RED = "#ff0000";
const GRAY = "#888888";

const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
  <Triggers.Provider>
    <Theming.Provider>{children}</Theming.Provider>
  </Triggers.Provider>
);

const renderInput = (props: Partial<Color.InputProps> = {}): RenderResult =>
  render(
    <Color.Input aria-label="Color" fallback={GRAY} onChange={vi.fn()} {...props} />,
    { wrapper: Wrapper },
  );

const textOf = (c: RenderResult): HTMLInputElement =>
  c.getByLabelText("Color") as HTMLInputElement;

const lastValue = (onChange: Mock): color.Color | undefined =>
  onChange.mock.calls.at(-1)?.[0];

describe("Input", () => {
  it("should show the hex digits of a set value", () => {
    const c = renderInput({ value: RED });
    expect(textOf(c).value).toEqual("ff0000");
  });

  it("should show Auto and mark the swatch while the value is absent", () => {
    const c = renderInput();
    expect(textOf(c).value).toEqual("");
    expect(textOf(c).placeholder).toEqual("Auto");
    const swatch = c.container.querySelector(`.${CSS.B("color-swatch")}`);
    expect(swatch?.className).toContain(CSS.M("auto"));
  });

  it("should apply a complete hex as the user types", () => {
    const onChange = vi.fn();
    const c = renderInput({ value: RED, onChange });
    fireEvent.change(textOf(c), { target: { value: "0000ff" } });
    expect(color.hex(lastValue(onChange))).toEqual("#0000ff");
  });

  it("should apply a short hex on Enter", () => {
    const onChange = vi.fn();
    const c = renderInput({ value: RED, onChange });
    fireEvent.change(textOf(c), { target: { value: "0f0" } });
    expect(onChange).not.toHaveBeenCalled();
    fireEvent.keyDown(textOf(c), { key: "Enter" });
    expect(color.hex(lastValue(onChange))).toEqual("#00ff00");
  });

  it("should keep the alpha when the typed hex has none", () => {
    const onChange = vi.fn();
    const c = renderInput({ value: "#ff000080", onChange });
    fireEvent.change(textOf(c), { target: { value: "0000ff" } });
    expect(color.aValue(lastValue(onChange) ?? color.ZERO)).toBeCloseTo(0.5, 2);
  });

  it("should apply a pasted CSS color at once", () => {
    const onChange = vi.fn();
    const c = renderInput({ value: RED, onChange });
    fireEvent.paste(textOf(c), {
      clipboardData: { getData: () => "rgb(0, 0, 255)" },
    });
    expect(color.hex(lastValue(onChange))).toEqual("#0000ff");
  });

  it("should clear the value when the user empties the box", () => {
    const onChange = vi.fn();
    const c = renderInput({ value: RED, onChange });
    fireEvent.change(textOf(c), { target: { value: "" } });
    fireEvent.blur(textOf(c));
    expect(onChange).toHaveBeenCalledWith(undefined);
  });

  it("should revert text that is not a color", () => {
    const onChange = vi.fn();
    const c = renderInput({ value: RED, onChange });
    fireEvent.change(textOf(c), { target: { value: "zzz" } });
    fireEvent.blur(textOf(c));
    expect(onChange).not.toHaveBeenCalled();
    expect(textOf(c).value).toEqual("ff0000");
  });

  describe("onlyChangeOnBlur", () => {
    it("should hold a typed hex until blur", () => {
      const onChange = vi.fn();
      const c = renderInput({ value: RED, onChange, onlyChangeOnBlur: true });
      fireEvent.change(textOf(c), { target: { value: "0000ff" } });
      expect(onChange).not.toHaveBeenCalled();
      fireEvent.blur(textOf(c));
      expect(onChange).toHaveBeenCalledOnce();
      expect(color.hex(lastValue(onChange))).toEqual("#0000ff");
    });

    it("should hold a paste until Enter", () => {
      const onChange = vi.fn();
      const c = renderInput({ value: RED, onChange, onlyChangeOnBlur: true });
      fireEvent.paste(textOf(c), {
        clipboardData: { getData: () => "rgb(0, 0, 255)" },
      });
      expect(onChange).not.toHaveBeenCalled();
      expect(textOf(c).value).toEqual("0000ff");
      fireEvent.keyDown(textOf(c), { key: "Enter" });
      expect(color.hex(lastValue(onChange))).toEqual("#0000ff");
    });
  });
});
