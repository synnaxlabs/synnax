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
import { type ReactElement, type ReactNode, useState } from "react";
import { describe, expect, it, vi } from "vitest";

import { Color } from "@/color";
import { CSS } from "@/css";
import { theme } from "@/theme";
import { Theming } from "@/theming";
import { Triggers } from "@/triggers";

const DARK = theme.themeZ.parse(theme.SYNNAX_DARK);

interface HarnessProps {
  initial: theme.Pair;
  onChange: (value: theme.Pair) => void;
}

const Harness = ({ initial, onChange }: HarnessProps): ReactElement => {
  const [value, setValue] = useState(initial);
  return (
    <Color.PairEditor
      value={value}
      onChange={(next) => {
        setValue(next);
        onChange(next);
      }}
    />
  );
};

const renderEditor = (
  initial: theme.Pair,
  themeKey: string = "synnaxLight",
): RenderResult & { onChange: ReturnType<typeof vi.fn> } => {
  const onChange = vi.fn();
  const wrapper = ({ children }: { children: ReactNode }): ReactElement => (
    <Triggers.Provider>
      <Theming.Provider theme={{ key: themeKey }} applyCSSVars={false}>
        {children}
      </Theming.Provider>
    </Triggers.Provider>
  );
  const c = render(<Harness initial={initial} onChange={onChange} />, { wrapper });
  return { ...c, onChange };
};

const hex = (c: RenderResult): string =>
  c.getByLabelText<HTMLInputElement>("Hex").value.toLowerCase();

const setHex = (c: RenderResult, value: string): void => {
  fireEvent.change(c.getByLabelText("Hex"), { target: { value } });
};

const autoSelected = (c: RenderResult): boolean =>
  c.getByLabelText("Auto").classList.contains(CSS.M("selected"));

const swatchOf = (c: RenderResult, mode: "Light" | "Dark"): string => {
  const dot = c
    .getByText(mode)
    .closest("button")
    ?.querySelector<HTMLElement>(`.${CSS.BE("color-pair-editor", "dot")}`);
  if (dot == null) throw new Error(`no swatch on ${mode}`);
  return dot.style.getPropertyValue("--pluto-swatch-color");
};

describe("Color.PairEditor", () => {
  it("should edit the light value on Light and the dark value on Dark", () => {
    const c = renderEditor({ light: "#ff0000" });
    expect(hex(c)).toBe("ff0000");
    setHex(c, "00ff00");
    expect(color.hex(c.onChange.mock.lastCall?.[0].light)).toBe("#00ff00");
    expect(c.onChange.mock.lastCall?.[0].dark).toBeUndefined();
    fireEvent.click(c.getByText("Dark"));
    setHex(c, "0000ff");
    const last = c.onChange.mock.lastCall?.[0];
    expect(color.hex(last.light)).toBe("#00ff00");
    expect(color.hex(last.dark)).toBe("#0000ff");
  });

  it("should start on the mode of the current theme", () => {
    const c = renderEditor({ light: "#ff0000", dark: "#0000ff" }, "synnaxDark");
    expect(hex(c)).toBe("0000ff");
  });

  it("should select Auto and show the derived dark value while the dark is absent", () => {
    const c = renderEditor({ light: "#ff0000" });
    expect(autoSelected(c)).toBe(true);
    fireEvent.click(c.getByText("Dark"));
    expect(hex(c)).toBe(color.hex(theme.legible("#ff0000", DARK)).slice(1));
  });

  it("should clear the dark value on Auto", () => {
    const c = renderEditor({ light: "#ff0000", dark: "#0000ff" });
    expect(autoSelected(c)).toBe(false);
    fireEvent.click(c.getByLabelText("Auto"));
    expect(c.onChange.mock.lastCall?.[0]).toEqual({ light: "#ff0000" });
    expect(autoSelected(c)).toBe(true);
  });

  it("should show each value in the swatch of its mode", () => {
    const c = renderEditor({ light: "#ff0000" });
    expect(swatchOf(c, "Light")).toBe(color.cssString("#ff0000"));
    expect(swatchOf(c, "Dark")).toBe(color.cssString(theme.legible("#ff0000", DARK)));
    fireEvent.click(c.getByText("Dark"));
    setHex(c, "0000ff");
    expect(swatchOf(c, "Dark")).toBe(color.cssString("#0000ff"));
  });
});
