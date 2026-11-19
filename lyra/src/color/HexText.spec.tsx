// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, render, type RenderResult } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Color } from "@/color";
import { CSS } from "@/css";

const renderHexText = (props: Partial<Color.HexTextProps> = {}): RenderResult =>
  render(
    <Color.HexText
      aria-label="Color"
      text="ff0000"
      alpha={1}
      onChange={vi.fn()}
      {...props}
    />,
  );

const textOf = (c: RenderResult): HTMLInputElement =>
  c.getByLabelText("Color") as HTMLInputElement;

const boxOf = (c: RenderResult): Element | null =>
  textOf(c).closest(`.${CSS.B("color-hex-text")}`);

describe("HexText", () => {
  it("should mark a name until the user types", () => {
    const c = renderHexText({ text: "Fault", named: true });
    expect(boxOf(c)?.className).toContain(CSS.M("named"));
    fireEvent.change(textOf(c), { target: { value: "0f0" } });
    expect(boxOf(c)?.className).not.toContain(CSS.M("named"));
  });

  it("should not mark hex digits", () => {
    const c = renderHexText();
    expect(boxOf(c)?.className).not.toContain(CSS.M("named"));
  });

  it("should discard a short hex when the user presses Escape", () => {
    const onChange = vi.fn();
    const c = renderHexText({ onChange });
    act(() => textOf(c).focus());
    fireEvent.change(textOf(c), { target: { value: "0f0" } });
    fireEvent.keyDown(textOf(c), { key: "Escape" });
    expect(onChange).not.toHaveBeenCalled();
    expect(textOf(c).value).toEqual("ff0000");
  });

  it("should discard a complete hex on Escape when it applies on blur", () => {
    const onChange = vi.fn();
    const c = renderHexText({ onChange, onlyChangeOnBlur: true });
    act(() => textOf(c).focus());
    fireEvent.change(textOf(c), { target: { value: "00ff00" } });
    fireEvent.keyDown(textOf(c), { key: "Escape" });
    expect(onChange).not.toHaveBeenCalled();
    expect(textOf(c).value).toEqual("ff0000");
  });

  it("should put a name back when the user presses Escape", () => {
    const onChange = vi.fn();
    const c = renderHexText({ text: "Fault", named: true, onChange });
    act(() => textOf(c).focus());
    fireEvent.change(textOf(c), { target: { value: "0f0" } });
    fireEvent.keyDown(textOf(c), { key: "Escape" });
    expect(onChange).not.toHaveBeenCalled();
    expect(textOf(c).value).toEqual("Fault");
    expect(boxOf(c)?.className).toContain(CSS.M("named"));
  });
});
