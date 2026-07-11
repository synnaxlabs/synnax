// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, render, renderHook, screen } from "@testing-library/react";
import { memo, type PropsWithChildren, type ReactElement, useState } from "react";
import { describe, expect, it, vi } from "vitest";

import { Select } from "@/select/base";

interface Counters {
  [key: string]: number;
}

const createItem = (renders: Counters) => {
  const Item = memo(({ itemKey }: { itemKey: string }): ReactElement => {
    renders[itemKey] = (renders[itemKey] ?? 0) + 1;
    const { selected, hovered, onSelect } = Select.useItemState(itemKey);
    return (
      <button
        data-testid={`item-${itemKey}`}
        data-selected={selected}
        data-hovered={hovered}
        onClick={onSelect}
      />
    );
  });
  Item.displayName = "Item";
  return Item;
};

const item = (key: string): HTMLElement => screen.getByTestId(`item-${key}`);

const isSelected = (key: string): boolean =>
  item(key).getAttribute("data-selected") === "true";

const isHovered = (key: string): boolean =>
  item(key).getAttribute("data-hovered") === "true";

describe("Selection", () => {
  describe("useItemState", () => {
    it("should mark the item matching a single selected value", () => {
      const Item = createItem({});
      render(
        <Select.Context value="a">
          <Item itemKey="a" />
          <Item itemKey="b" />
        </Select.Context>,
      );
      expect(isSelected("a")).toBe(true);
      expect(isSelected("b")).toBe(false);
    });

    it("should mark every item included in an array value", () => {
      const Item = createItem({});
      render(
        <Select.Context value={["a", "c"]}>
          <Item itemKey="a" />
          <Item itemKey="b" />
          <Item itemKey="c" />
        </Select.Context>,
      );
      expect(isSelected("a")).toBe(true);
      expect(isSelected("b")).toBe(false);
      expect(isSelected("c")).toBe(true);
    });

    it("should call the provider's onSelect with the item's key", () => {
      const onSelect = vi.fn();
      const Item = createItem({});
      render(
        <Select.Context value="a" onSelect={onSelect}>
          <Item itemKey="a" />
          <Item itemKey="b" />
        </Select.Context>,
      );
      fireEvent.click(item("b"));
      expect(onSelect).toHaveBeenCalledTimes(1);
      expect(onSelect).toHaveBeenCalledWith("b");
    });

    it("should re-render only the items whose selection state changed", () => {
      const renders: Counters = {};
      const Item = createItem(renders);
      const Harness = (): ReactElement => {
        const [value, setValue] = useState("a");
        return (
          <Select.Context value={value} onSelect={setValue}>
            <Item itemKey="a" />
            <Item itemKey="b" />
            <Item itemKey="c" />
          </Select.Context>
        );
      };
      render(<Harness />);
      const baseline = { ...renders };
      fireEvent.click(item("b"));
      expect(isSelected("a")).toBe(false);
      expect(isSelected("b")).toBe(true);
      expect(renders.a).toBeGreaterThan(baseline.a);
      expect(renders.b).toBeGreaterThan(baseline.b);
      expect(renders.c).toEqual(baseline.c);
    });

    it("should mark the hovered item", () => {
      const Item = createItem({});
      render(
        <Select.Context value="a" hover="b">
          <Item itemKey="a" />
          <Item itemKey="b" />
        </Select.Context>,
      );
      expect(item("a").getAttribute("data-hovered")).toEqual("false");
      expect(item("b").getAttribute("data-hovered")).toEqual("true");
    });

    it("should move the hover mark when the hovered value changes", () => {
      const renders: Counters = {};
      const Item = createItem(renders);
      let setHover: (key: string) => void = () => {};
      const Harness = (): ReactElement => {
        const [hover, setState] = useState("a");
        setHover = setState;
        return (
          <Select.Context value={undefined} hover={hover}>
            <Item itemKey="a" />
            <Item itemKey="b" />
            <Item itemKey="c" />
          </Select.Context>
        );
      };
      render(<Harness />);
      expect(isHovered("a")).toBe(true);
      expect(isHovered("b")).toBe(false);
      const baseline = { ...renders };
      act(() => setHover("b"));
      expect(isHovered("a")).toBe(false);
      expect(isHovered("b")).toBe(true);
      expect(renders.a).toBeGreaterThan(baseline.a);
      expect(renders.b).toBeGreaterThan(baseline.b);
      expect(renders.c).toEqual(baseline.c);
    });

    it("should report no selection when used outside a provider", () => {
      const Item = createItem({});
      render(<Item itemKey="a" />);
      expect(isSelected("a")).toBe(false);
      fireEvent.click(item("a"));
      expect(isSelected("a")).toBe(false);
    });
  });

  describe("useSelected", () => {
    it("should return a single value as a one-element array", () => {
      const { result } = renderHook(() => Select.useSelected(), {
        wrapper: ({ children }) => (
          <Select.Context value="a">{children}</Select.Context>
        ),
      });
      expect(result.current).toEqual(["a"]);
    });

    it("should track selection changes", () => {
      let setValue: (value: string[]) => void = () => {};
      const Wrapper = ({ children }: PropsWithChildren): ReactElement => {
        const [value, setState] = useState<string[]>(["a"]);
        setValue = setState;
        return <Select.Context value={value}>{children}</Select.Context>;
      };
      const { result } = renderHook(() => Select.useSelected(), {
        wrapper: Wrapper,
      });
      expect(result.current).toEqual(["a"]);
      act(() => setValue(["a", "b"]));
      expect(result.current).toEqual(["a", "b"]);
    });

    it("should return an empty array when nothing is selected", () => {
      const { result } = renderHook(() => Select.useSelected(), {
        wrapper: ({ children }) => (
          <Select.Context value={undefined}>{children}</Select.Context>
        ),
      });
      expect(result.current).toEqual([]);
    });
  });

  describe("useClear", () => {
    it("should invoke the provider's clear callback", () => {
      const clear = vi.fn();
      const { result } = renderHook(() => Select.useClear(), {
        wrapper: ({ children }) => (
          <Select.Context value="a" clear={clear}>
            {children}
          </Select.Context>
        ),
      });
      result.current();
      expect(clear).toHaveBeenCalledTimes(1);
    });
  });
});
