// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render, type RenderResult } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { CSS } from "@/css";
import { Icon } from "@/icon";
import { Input } from "@/input";
import { Theming } from "@/theming";
import { Triggers } from "@/triggers";

const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
  <Triggers.Provider>
    <Theming.Provider>{children}</Theming.Provider>
  </Triggers.Provider>
);

const renderPicker = (props: Partial<Input.IconPickerProps> = {}): RenderResult =>
  render(<Input.IconPicker onChange={vi.fn()} {...props} />, { wrapper: Wrapper });

const search = (c: RenderResult): HTMLInputElement =>
  c.getByLabelText("Search icons") as HTMLInputElement;

const options = (c: RenderResult): HTMLElement[] => c.queryAllByRole("option");

const names = (c: RenderResult): (string | null)[] =>
  options(c).map((o) => o.getAttribute("aria-label"));

describe("IconPicker", () => {
  describe("grid", () => {
    it("should show every pickable icon in order", () => {
      const c = renderPicker();
      expect(names(c)).toEqual(Icon.PICKABLE.map(({ name }) => name));
    });

    it("should call onChange with the key of the clicked icon", () => {
      const onChange = vi.fn();
      const c = renderPicker({ onChange });
      fireEvent.click(c.getByLabelText("Valve"));
      expect(onChange).toHaveBeenCalledExactlyOnceWith("valve");
    });

    it("should mark the picked icon as selected", () => {
      const c = renderPicker({ value: "pump" });
      expect(c.getByLabelText("Pump").getAttribute("aria-selected")).toEqual("true");
      expect(c.getByLabelText("Valve").getAttribute("aria-selected")).toEqual("false");
    });

    it("should draw every icon in the given color", () => {
      const c = renderPicker({ color: "#dc136c" });
      options(c).forEach((o) =>
        expect(o.querySelector("svg")?.getAttribute("class")).toContain(
          CSS.BM("symbol-colored", "icon"),
        ),
      );
    });

    it("should draw the icons in the inherited color without a color", () => {
      const c = renderPicker();
      expect(c.container.querySelector(`.${CSS.B("symbol-colored")}`)).toBeNull();
    });
  });

  describe("search", () => {
    it("should focus the search field on mount", () => {
      const c = renderPicker();
      expect(document.activeElement).toBe(search(c));
    });

    it("should find an icon by name", () => {
      const c = renderPicker();
      fireEvent.change(search(c), { target: { value: "valve" } });
      expect(names(c)[0]).toEqual("Valve");
    });

    it("should find an icon by a tag", () => {
      const c = renderPicker();
      fireEvent.change(search(c), { target: { value: "psi" } });
      expect(names(c)).toEqual(["Pressure"]);
    });

    it("should say so when no icon matches", () => {
      const c = renderPicker();
      fireEvent.change(search(c), { target: { value: "zzzzzz" } });
      expect(options(c)).toHaveLength(0);
      expect(c.getByText("No icons match")).toBeTruthy();
    });

    it("should pick the best match when the user presses Enter", () => {
      const onChange = vi.fn();
      const c = renderPicker({ onChange });
      fireEvent.change(search(c), { target: { value: "psi" } });
      fireEvent.keyDown(search(c), { key: "Enter" });
      expect(onChange).toHaveBeenCalledExactlyOnceWith("pressure");
    });
  });

  describe("keyboard", () => {
    const focusedName = (): string | null =>
      document.activeElement?.getAttribute("aria-label") ?? null;

    it("should move from the search field to the first icon with ArrowDown", () => {
      const c = renderPicker();
      fireEvent.keyDown(search(c), { key: "ArrowDown" });
      expect(focusedName()).toEqual(Icon.PICKABLE[0].name);
    });

    it("should move from the search field to the picked icon with ArrowDown", () => {
      const c = renderPicker({ value: "pump" });
      fireEvent.keyDown(search(c), { key: "ArrowDown" });
      expect(focusedName()).toEqual("Pump");
    });

    it("should give only the active icon a tab stop", () => {
      const c = renderPicker({ value: "pump" });
      const stops = options(c).filter((o) => o.tabIndex === 0);
      expect(stops.map((o) => o.getAttribute("aria-label"))).toEqual(["Pump"]);
    });

    it("should move back past the neighbor of the picked icon", () => {
      const c = renderPicker({ value: "pump" });
      fireEvent.keyDown(search(c), { key: "ArrowDown" });
      fireEvent.keyDown(document.activeElement as Element, { key: "ArrowLeft" });
      fireEvent.keyDown(document.activeElement as Element, { key: "ArrowLeft" });
      expect(focusedName()).toEqual("Database");
    });

    it("should move the tab stop to the focused icon", () => {
      const c = renderPicker({ value: "pump" });
      fireEvent.keyDown(search(c), { key: "ArrowDown" });
      fireEvent.keyDown(document.activeElement as Element, { key: "ArrowLeft" });
      const stops = options(c).filter((o) => o.tabIndex === 0);
      expect(stops.map((o) => o.getAttribute("aria-label"))).toEqual(["Valve"]);
    });

    it("should move along a line and wrap onto the next", () => {
      const c = renderPicker();
      fireEvent.keyDown(search(c), { key: "ArrowDown" });
      fireEvent.keyDown(document.activeElement as Element, { key: "ArrowRight" });
      expect(focusedName()).toEqual(Icon.PICKABLE[1].name);
      fireEvent.keyDown(document.activeElement as Element, { key: "ArrowLeft" });
      fireEvent.keyDown(document.activeElement as Element, { key: "End" });
      expect(focusedName()).toEqual(Icon.PICKABLE.at(-1)?.name);
      fireEvent.keyDown(document.activeElement as Element, { key: "Home" });
      expect(focusedName()).toEqual(Icon.PICKABLE[0].name);
    });

    it("should move down and up by a line of ten", () => {
      const c = renderPicker();
      fireEvent.keyDown(search(c), { key: "ArrowDown" });
      fireEvent.keyDown(document.activeElement as Element, { key: "ArrowDown" });
      expect(focusedName()).toEqual(Icon.PICKABLE[10].name);
      fireEvent.keyDown(document.activeElement as Element, { key: "ArrowUp" });
      expect(focusedName()).toEqual(Icon.PICKABLE[0].name);
    });

    it("should return to the search field with ArrowUp from the first line", () => {
      const c = renderPicker();
      fireEvent.keyDown(search(c), { key: "ArrowDown" });
      fireEvent.keyDown(document.activeElement as Element, { key: "ArrowUp" });
      expect(document.activeElement).toBe(search(c));
    });

    it("should stop at the last icon when the line below is short", () => {
      const c = renderPicker();
      fireEvent.change(search(c), { target: { value: "e" } });
      const shown = names(c);
      expect(shown.length % 10).not.toEqual(0);
      fireEvent.keyDown(search(c), { key: "ArrowDown" });
      fireEvent.keyDown(document.activeElement as Element, { key: "End" });
      fireEvent.keyDown(document.activeElement as Element, { key: "ArrowRight" });
      expect(focusedName()).toEqual(shown.at(-1));
    });
  });
});
