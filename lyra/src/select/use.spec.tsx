// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactNode, useState } from "react";
import { describe, expect, it } from "vitest";

import { Select } from "@/select";
import { Triggers } from "@/triggers";

const DATA = ["1", "2", "3"];

interface Harness<V> {
  value: V;
  onSelect: (key: string) => void;
  clear: () => void;
}

const renderFrame = <V,>(
  initial: V,
  frame: (value: V, onChange: (next: V) => void, children: ReactNode) => ReactNode,
) => {
  const state = { value: initial };
  const Wrapper = ({ children }: PropsWithChildren): ReactNode => {
    const [value, onChange] = useState<V>(initial);
    state.value = value;
    return <Triggers.Provider>{frame(value, onChange, children)}</Triggers.Provider>;
  };
  return renderHook(
    (): Harness<V> => {
      const { onSelect, clear } = Select.useContext<string>();
      return {
        get value() {
          return state.value;
        },
        onSelect,
        clear,
      };
    },
    { wrapper: Wrapper },
  );
};

const renderMultiple = (
  props: Partial<Select.UseMultipleProps<string>> = {},
  initial: string[] = [],
  getData: () => string[] = () => DATA,
) =>
  renderFrame(initial, (value, onChange, children) => (
    <Select.Frame
      multiple
      data={getData()}
      value={value}
      onChange={onChange}
      {...props}
    >
      {children}
    </Select.Frame>
  ));

interface SingleOptions {
  allowNone?: boolean;
  autoSelectOnNone?: boolean;
}

const renderSingle = (
  { allowNone = true, autoSelectOnNone }: SingleOptions = {},
  initial: string | null | undefined = undefined,
  getData: () => string[] = () => DATA,
) =>
  renderFrame(initial, (value, onChange, children) =>
    allowNone ? (
      <Select.Frame
        data={getData()}
        value={value ?? undefined}
        onChange={onChange}
        allowNone
        autoSelectOnNone={autoSelectOnNone}
      >
        {children}
      </Select.Frame>
    ) : (
      <Select.Frame
        data={getData()}
        value={value as string}
        onChange={onChange}
        allowNone={false}
        autoSelectOnNone={autoSelectOnNone}
      >
        {children}
      </Select.Frame>
    ),
  );

describe("useSelect", () => {
  describe("multiple selection", () => {
    it("should select two items", () => {
      const { result } = renderMultiple();
      act(() => result.current.onSelect("1"));
      expect(result.current.value).toEqual(["1"]);
      act(() => result.current.onSelect("2"));
      expect(result.current.value).toEqual(["1", "2"]);
    });

    it("should deselect an item when you click it again", () => {
      const { result } = renderMultiple();
      act(() => result.current.onSelect("1"));
      act(() => result.current.onSelect("2"));
      act(() => result.current.onSelect("1"));
      expect(result.current.value).toEqual(["2"]);
    });

    it("should clear all selections", () => {
      const { result } = renderMultiple();
      act(() => result.current.onSelect("1"));
      act(() => result.current.onSelect("2"));
      act(() => result.current.clear());
      expect(result.current.value).toEqual([]);
    });
    describe("no not allow none", () => {
      it("should not allow removing the last selection", () => {
        const { result } = renderMultiple({ allowNone: false });
        act(() => result.current.onSelect("1"));
        act(() => result.current.onSelect("1"));
        expect(result.current.value).toEqual(["1"]);
      });
    });
    describe("replaceOnSingle", () => {
      it("should replace the selection when you click a new item", () => {
        const { result } = renderMultiple({ replaceOnSingle: true });
        act(() => result.current.onSelect("1"));
        act(() => result.current.onSelect("2"));
        expect(result.current.value).toEqual(["2"]);
      });
    });

    it("should clear the selection when clear() is called", () => {
      const { result } = renderMultiple();
      act(() => result.current.onSelect("1"));
      act(() => result.current.clear());
      expect(result.current.value).toEqual([]);
    });

    describe("autoSelectOnNone", () => {
      it("should auto-select the first item when value is empty", () => {
        const { result } = renderMultiple({ autoSelectOnNone: true });
        expect(result.current.value).toEqual(["1"]);
      });

      it("should not auto-select when value is not empty", () => {
        const { result } = renderMultiple({ autoSelectOnNone: true }, ["2"]);
        expect(result.current.value).toEqual(["2"]);
      });

      it("should auto-select after clearing when autoSelectOnNone is true", () => {
        const { result } = renderMultiple({ autoSelectOnNone: true });
        expect(result.current.value).toEqual(["1"]);
        act(() => result.current.onSelect("2"));
        expect(result.current.value).toEqual(["1", "2"]);
        act(() => result.current.clear());
        expect(result.current.value).toEqual(["1"]);
      });

      it("should not auto-select when autoSelectOnNone is false", () => {
        const { result } = renderMultiple({ autoSelectOnNone: false });
        expect(result.current.value).toEqual([]);
      });

      it("should auto-select first item when allowNone is false and autoSelectOnNone is true", () => {
        const { result } = renderMultiple({ allowNone: false, autoSelectOnNone: true });
        expect(result.current.value).toEqual(["1"]);
      });

      it("should auto-select when the selected list item is removed", () => {
        let data = ["1", "2", "3"];
        const { result, rerender } = renderMultiple(
          { allowNone: false, autoSelectOnNone: true },
          [],
          () => data,
        );
        expect(result.current.value).toEqual(["1"]);
        act(() => {
          data = ["2", "3"];
          rerender();
        });
        expect(result.current.value).toEqual(["2"]);
      });
    });
  });
  describe("single selection", () => {
    it("should select one item", () => {
      const { result } = renderSingle();
      act(() => result.current.onSelect("1"));
      expect(result.current.value).toEqual("1");
      act(() => result.current.onSelect("2"));
      expect(result.current.value).toEqual("2");
    });

    it("should deselect an item when you click it again", () => {
      const { result } = renderSingle();
      act(() => result.current.onSelect("1"));
      act(() => result.current.onSelect("1"));
      expect(result.current.value).toEqual(null);
    });

    describe("not allow none", () => {
      it("should not allow clearing all selections", () => {
        const { result } = renderSingle({ allowNone: false });
        act(() => result.current.onSelect("1"));
        act(() => result.current.onSelect("1"));
        expect(result.current.value).toEqual("1");
      });
    });

    it("should clear the selection when clear() is called", () => {
      const { result } = renderSingle();
      act(() => result.current.onSelect("1"));
      act(() => result.current.clear());
      expect(result.current.value).toEqual(null);
    });

    describe("autoSelectOnNone", () => {
      it("should auto-select the first item when value is null", () => {
        const { result } = renderSingle({ autoSelectOnNone: true });
        expect(result.current.value).toEqual("1");
      });

      it("should not auto-select when value is not null", () => {
        const { result } = renderSingle({ autoSelectOnNone: true }, "2");
        expect(result.current.value).toEqual("2");
      });

      it("should auto-select after clearing when autoSelectOnNone is true and allowNone is true", () => {
        const { result } = renderSingle({ autoSelectOnNone: true, allowNone: true });
        expect(result.current.value).toEqual("1");
        act(() => result.current.onSelect("2"));
        expect(result.current.value).toEqual("2");
        act(() => result.current.clear());
        expect(result.current.value).toEqual("1");
      });

      it("should not auto-select when autoSelectOnNone is false", () => {
        const { result } = renderSingle({ autoSelectOnNone: false });
        expect(result.current.value).toEqual(undefined);
      });

      it("should auto-select first item when allowNone is false and autoSelectOnNone is true", () => {
        const { result } = renderSingle({ allowNone: false, autoSelectOnNone: true });
        expect(result.current.value).toEqual("1");
      });

      it("should handle empty data gracefully", () => {
        const { result } = renderSingle(
          { autoSelectOnNone: true },
          undefined,
          () => [],
        );
        expect(result.current.value).toEqual(undefined);
      });

      it("should auto-select when the selected list item is removed", () => {
        let data = ["1", "2", "3"];
        const { result, rerender } = renderSingle(
          { allowNone: false, autoSelectOnNone: true },
          undefined,
          () => data,
        );
        expect(result.current.value).toEqual("1");
        act(() => {
          data = ["2", "3"];
          rerender();
        });
        expect(result.current.value).toEqual("2");
      });
    });
  });
});

describe("hasModifier", () => {
  it.each([
    ["shift", { shiftKey: true, ctrlKey: false, metaKey: false }],
    ["control", { shiftKey: false, ctrlKey: true, metaKey: false }],
    ["command", { shiftKey: false, ctrlKey: false, metaKey: true }],
  ])("should report a %s click as a selection gesture", (_, e) => {
    expect(Select.hasModifier(e)).toBe(true);
  });

  it("should report an unmodified click as an activation", () => {
    expect(
      Select.hasModifier({ shiftKey: false, ctrlKey: false, metaKey: false }),
    ).toBe(false);
  });
});
