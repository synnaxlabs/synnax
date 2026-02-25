// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Provider as ColorProvider } from "@/color/Provider";
import { Swatch } from "@/color/Swatch";
import { Triggers } from "@/triggers";

const wrapper = ({ children }: PropsWithChildren): ReactElement => (
  <Triggers.Provider>
    <ColorProvider>{children}</ColorProvider>
  </Triggers.Provider>
);

const findSwatch = (container: HTMLElement): Element =>
  container.querySelector(".pluto-color-swatch")!;

const findHexInput = (): HTMLInputElement | null => {
  const dialog = document.querySelector('[role="dialog"]');
  if (dialog == null) return null;
  const inputs = dialog.querySelectorAll("input");
  return inputs[0] ?? null;
};

describe("Swatch", () => {
  describe("rendering", () => {
    it("should render a button element", () => {
      const c = render(<Swatch value="#FF0000" />, { wrapper });
      const swatch = findSwatch(c.container);
      expect(swatch).toBeTruthy();
      expect(swatch.tagName).toBe("BUTTON");
    });

    it("should render as disabled when onChange is not provided", () => {
      const c = render(<Swatch value="#FF0000" />, { wrapper });
      const swatch = findSwatch(c.container);
      expect(swatch.className).toContain("pluto--disabled");
    });

    it("should render as disabled when allowChange is false", () => {
      const c = render(
        <Swatch value="#FF0000" onChange={vi.fn()} allowChange={false} />,
        { wrapper },
      );
      const swatch = findSwatch(c.container);
      expect(swatch.className).toContain("pluto--disabled");
    });

    it("should call onClick when allowChange is false and swatch is clicked", () => {
      const onClick = vi.fn();
      const c = render(
        <Swatch
          value="#FF0000"
          onChange={vi.fn()}
          allowChange={false}
          onClick={onClick}
        />,
        { wrapper },
      );
      fireEvent.click(findSwatch(c.container));
      expect(onClick).toHaveBeenCalled();
    });
  });

  describe("picker visibility", () => {
    it("should not show the picker dialog by default", () => {
      const c = render(<Swatch value="#FF0000" onChange={vi.fn()} />, { wrapper });
      expect(c.queryByRole("dialog")).toBeNull();
    });

    it("should show the picker dialog when the swatch is clicked", () => {
      const c = render(<Swatch value="#FF0000" onChange={vi.fn()} />, { wrapper });
      fireEvent.click(findSwatch(c.container));
      expect(c.queryByRole("dialog")).toBeTruthy();
    });

    it("should hide the picker dialog when Escape is pressed", () => {
      const c = render(<Swatch value="#FF0000" onChange={vi.fn()} />, { wrapper });
      fireEvent.click(findSwatch(c.container));
      expect(c.queryByRole("dialog")).toBeTruthy();
      fireEvent.keyDown(c.container, { code: "Escape" });
      expect(c.queryByRole("dialog")).toBeNull();
    });

    it("should respect initialVisible", () => {
      const c = render(<Swatch value="#FF0000" onChange={vi.fn()} initialVisible />, {
        wrapper,
      });
      expect(c.queryByRole("dialog")).toBeTruthy();
    });
  });

  describe("onChange", () => {
    it("should call onChange when the picker fires a color change", () => {
      const onChange = vi.fn();
      const c = render(<Swatch value="#FF0000" onChange={onChange} />, { wrapper });
      fireEvent.click(findSwatch(c.container));
      const hexInput = findHexInput();
      expect(hexInput).toBeTruthy();
      fireEvent.change(hexInput!, { target: { value: "00FF00" } });
      expect(onChange).toHaveBeenCalled();
    });
  });

  describe("onlyChangeOnBlur", () => {
    it("should not call onChange on picker interaction when onlyChangeOnBlur is true", () => {
      const onChange = vi.fn();
      const c = render(
        <Swatch value="#FF0000" onChange={onChange} onlyChangeOnBlur />,
        { wrapper },
      );
      fireEvent.click(findSwatch(c.container));
      const hexInput = findHexInput();
      expect(hexInput).toBeTruthy();
      fireEvent.change(hexInput!, { target: { value: "00FF00" } });
      expect(onChange).not.toHaveBeenCalled();
    });

    it("should call onChange with the pending color when the dialog closes", () => {
      const onChange = vi.fn();
      const c = render(
        <Swatch value="#FF0000" onChange={onChange} onlyChangeOnBlur />,
        { wrapper },
      );
      fireEvent.click(findSwatch(c.container));
      const hexInput = findHexInput();
      expect(hexInput).toBeTruthy();
      fireEvent.change(hexInput!, { target: { value: "00FF00" } });
      fireEvent.keyDown(c.container, { code: "Escape" });
      expect(onChange).toHaveBeenCalledTimes(1);
    });

    it("should not call onChange on close if no color was picked", () => {
      const onChange = vi.fn();
      const c = render(
        <Swatch value="#FF0000" onChange={onChange} onlyChangeOnBlur />,
        { wrapper },
      );
      fireEvent.click(findSwatch(c.container));
      fireEvent.keyDown(c.container, { code: "Escape" });
      expect(onChange).not.toHaveBeenCalled();
    });
  });

  describe("onVisibleChange", () => {
    it("should call onVisibleChange when the dialog opens", () => {
      const onVisibleChange = vi.fn();
      const c = render(
        <Swatch
          value="#FF0000"
          onChange={vi.fn()}
          visible={false}
          onVisibleChange={onVisibleChange}
        />,
        { wrapper },
      );
      fireEvent.click(findSwatch(c.container));
      expect(onVisibleChange).toHaveBeenCalledWith(true);
    });

    it("should call onVisibleChange when the dialog closes", () => {
      const onVisibleChange = vi.fn();
      const c = render(
        <Swatch
          value="#FF0000"
          onChange={vi.fn()}
          visible
          onVisibleChange={onVisibleChange}
        />,
        { wrapper },
      );
      fireEvent.keyDown(c.container, { code: "Escape" });
      expect(onVisibleChange).toHaveBeenCalled();
    });
  });
});
