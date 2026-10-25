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
import { describe, expect, it, vi } from "vitest";

import { Color } from "@/color";
import { CSS } from "@/css";
import { Theming } from "@/theming";
import { Triggers } from "@/triggers";

const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
  <Triggers.Provider>
    <Theming.Provider>{children}</Theming.Provider>
  </Triggers.Provider>
);

const FREQUENT: Color.ContextState = {
  frequent: { "#123456": { lastUsed: 0, count: 1, relevance: 1 } },
};

const renderSwatches = (props: Color.SwatchesProps): RenderResult =>
  render(<Color.Swatches {...props} />, { wrapper: Wrapper });

const renderWithFrequent = (props: Color.SwatchesProps): RenderResult =>
  render(
    <Color.Provider value={FREQUENT} onChange={vi.fn()}>
      <Color.Swatches {...props} />
    </Color.Provider>,
    { wrapper: Wrapper },
  );

describe("Swatches", () => {
  describe("presets", () => {
    it("should offer the theme colors and the visualization palette", () => {
      const c = renderSwatches({ value: undefined, onChange: vi.fn() });
      const swatches = c.container.querySelectorAll(`.${CSS.B("color-swatch")}`);
      expect(swatches).toHaveLength(18);
    });

    it("should pick a preset with one click", () => {
      const onChange = vi.fn();
      const c = renderSwatches({ value: "#ff0000", onChange });
      fireEvent.click(c.getByLabelText("#dc136c"));
      expect(onChange).toHaveBeenCalledExactlyOnceWith(color.construct("#dc136c"));
    });

    it("should mark the swatch that matches the value", () => {
      const c = renderSwatches({ value: "#DC136C", onChange: vi.fn() });
      expect(c.getByLabelText("#dc136c").className).toContain(CSS.M("selected"));
      expect(c.getByLabelText("#20a4f3").className).not.toContain(CSS.M("selected"));
    });
  });

  describe("auto", () => {
    it("should not offer Auto without a fallback", () => {
      const c = renderSwatches({ value: "#ff0000", onChange: vi.fn() });
      expect(c.queryByLabelText("Auto")).toBeNull();
    });

    it("should clear the value when the user picks Auto", () => {
      const onChange = vi.fn();
      const c = renderSwatches({ value: "#ff0000", fallback: "#888888", onChange });
      fireEvent.click(c.getByLabelText("Auto"));
      expect(onChange).toHaveBeenCalledExactlyOnceWith(undefined);
    });

    it("should mark Auto while the value is absent", () => {
      const c = renderSwatches({
        value: undefined,
        fallback: "#888888",
        onChange: vi.fn(),
      });
      expect(c.getByLabelText("Auto").className).toContain(CSS.M("selected"));
    });

    it("should not mark Auto while a value is set", () => {
      const c = renderSwatches({
        value: "#ff0000",
        fallback: "#888888",
        onChange: vi.fn(),
      });
      expect(c.getByLabelText("Auto").className).not.toContain(CSS.M("selected"));
    });
  });

  describe("frequent colors", () => {
    it("should not show frequent colors when the user has none", () => {
      const c = renderSwatches({ value: undefined, onChange: vi.fn() });
      expect(
        c.container.querySelectorAll(`.${CSS.BE("color-swatches", "grid")}`),
      ).toHaveLength(1);
    });

    it("should show the frequent colors after the presets", () => {
      const c = renderWithFrequent({ value: undefined, onChange: vi.fn() });
      const grids = c.container.querySelectorAll(
        `.${CSS.BE("color-swatches", "grid")}`,
      );
      expect(grids).toHaveLength(2);
      expect(grids[1].querySelector(`[aria-label="#123456"]`)).not.toBeNull();
    });

    it("should pick a frequent color with one click", () => {
      const onChange = vi.fn();
      const c = renderWithFrequent({ value: undefined, onChange });
      fireEvent.click(c.getByLabelText("#123456"));
      expect(onChange).toHaveBeenCalledExactlyOnceWith(color.construct("#123456"));
    });
  });

  describe("children", () => {
    it("should render the children after the presets", () => {
      const c = renderSwatches({
        value: undefined,
        onChange: vi.fn(),
        children: <button aria-label="Extra" />,
      });
      const grid = c.container.querySelector(`.${CSS.BE("color-swatches", "grid")}`);
      expect(grid?.lastElementChild).toBe(c.getByLabelText("Extra"));
    });
  });
});
