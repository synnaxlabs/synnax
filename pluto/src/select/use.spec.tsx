// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Optional } from "@synnaxlabs/x";
import { act, renderHook } from "@testing-library/react";
import { type PropsWithChildren, useState } from "react";
import { describe, expect, it } from "vitest";

import { List } from "@/list";
import { Select } from "@/select";
import { Triggers } from "@/triggers";

interface UseSelectMultipleWrapperReturn
  extends Pick<Select.UseReturn<string>, "onSelect" | "clear"> {
  value: string[];
}

const useMultipleWrapper = (
  props?: Optional<Omit<Select.UseMultipleProps<string>, "data" | "onChange">, "value">,
): UseSelectMultipleWrapperReturn => {
  const [value, onChange] = useState<string[]>(props?.value ?? []);
  const { clear, onSelect } = Select.useMultiple<string>({
    ...props,
    value,
    onChange,
  });
  return { value, clear, onSelect };
};

interface UseSelectSingleWrapperReturn
  extends Pick<Select.UseReturn<string>, "onSelect" | "clear"> {
  value: string | undefined;
}

const useSelectSingleWrapper = (
  props?: Omit<Select.UseSingleProps<string>, "data" | "value" | "onChange">,
): UseSelectSingleWrapperReturn => {
  const [value, onChange] = useState<string | undefined>(undefined);
  const { clear, onSelect } = Select.useSingle<string>({
    allowNone: true,
    ...props,
    value,
    onChange,
  });
  return { value, clear, onSelect };
};

const data = ["1", "2", "3", "4"];

const Wrapper = (props: PropsWithChildren) => (
  <Triggers.Provider>
    <List.Frame data={data} {...props} />;
  </Triggers.Provider>
);

describe("useSelect", () => {
  describe("multiple selection", () => {
    describe("basic selection mechanics", () => {
      it("should select two items", () => {
        const { result } = renderHook(useMultipleWrapper, { wrapper: Wrapper });
        act(() => result.current.onSelect("1"));
        expect(result.current.value).toEqual(["1"]);
        act(() => result.current.onSelect("2"));
        expect(result.current.value).toEqual(["1", "2"]);
      });

      it("should deselect an item when you click it again", () => {
        const { result } = renderHook(useMultipleWrapper, { wrapper: Wrapper });
        act(() => result.current.onSelect("1"));
        act(() => result.current.onSelect("2"));
        act(() => result.current.onSelect("1"));
        expect(result.current.value).toEqual(["2"]);
      });
    });

    describe("clear", () => {
      it("should clear all selections", () => {
        const { result } = renderHook(useMultipleWrapper, { wrapper: Wrapper });
        act(() => result.current.onSelect("1"));
        act(() => result.current.onSelect("2"));
        act(() => result.current.clear());
        expect(result.current.value).toEqual([]);
      });
    });

    describe("allowNone is false", () => {
      it("should not allow removing the last selection", () => {
        const { result } = renderHook(() => useMultipleWrapper({ allowNone: false }), {
          wrapper: Wrapper,
        });
        act(() => result.current.onSelect("1"));
        act(() => result.current.onSelect("1"));
        expect(result.current.value).toEqual(["1"]);
      });
    });

    describe("replaceOnSingle", () => {
      it("should replace the selection when you click a new item", () => {
        const { result } = renderHook(
          () => useMultipleWrapper({ replaceOnSingle: true }),
          {
            wrapper: Wrapper,
          },
        );
        act(() => result.current.onSelect("1"));
        act(() => result.current.onSelect("2"));
        expect(result.current.value).toEqual(["2"]);
      });
    });

    describe("autoSelectOnNone", () => {
      it("should auto-select the first item when value is empty", () => {
        const { result } = renderHook(
          () => useMultipleWrapper({ autoSelectOnNone: true }),
          { wrapper: Wrapper },
        );
        expect(result.current.value).toEqual(["1"]);
      });

      it("should not auto-select when value is not empty", () => {
        const { result } = renderHook(
          () => {
            const [value, onChange] = useState<string[]>(["2"]);
            const { clear, onSelect } = Select.useMultiple<string>({
              autoSelectOnNone: true,
              value,
              onChange,
            });
            return { value, clear, onSelect };
          },
          { wrapper: Wrapper },
        );
        expect(result.current.value).toEqual(["2"]);
      });

      it("should auto-select after clearing when autoSelectOnNone is true", () => {
        const { result } = renderHook(
          () => useMultipleWrapper({ autoSelectOnNone: true }),
          { wrapper: Wrapper },
        );
        expect(result.current.value).toEqual(["1"]);
        act(() => result.current.onSelect("2"));
        expect(result.current.value).toEqual(["1", "2"]);
        act(() => result.current.clear());
        expect(result.current.value).toEqual(["1"]);
      });

      it("should not auto-select when autoSelectOnNone is false", () => {
        const { result } = renderHook(
          () => useMultipleWrapper({ autoSelectOnNone: false }),
          { wrapper: Wrapper },
        );
        expect(result.current.value).toEqual([]);
      });

      it("should auto-select first item when allowNone is false and autoSelectOnNone is true", () => {
        const { result } = renderHook(
          () => useMultipleWrapper({ allowNone: false, autoSelectOnNone: true }),
          { wrapper: Wrapper },
        );
        expect(result.current.value).toEqual(["1"]);
      });
    });

    describe("right click", () => {
      describe("multiple selected before", () => {
        it("should extend the selection when you right click even if replaceOnSingle is true", () => {
          const { result } = renderHook(
            () => useMultipleWrapper({ replaceOnSingle: true, value: ["1", "2"] }),
            {
              wrapper: Wrapper,
            },
          );
          act(() =>
            result.current.onSelect("3", { button: Triggers.MOUSE_RIGHT_NUMBER }),
          );
          expect(result.current.value).toEqual(["1", "2", "3"]);
        });

        it("should replace the previous right click selection when you right click a new item", () => {
          const { result } = renderHook(
            () => useMultipleWrapper({ replaceOnSingle: true, value: ["1", "2"] }),
            {
              wrapper: Wrapper,
            },
          );
          act(() =>
            result.current.onSelect("3", { button: Triggers.MOUSE_RIGHT_NUMBER }),
          );
          expect(result.current.value).toEqual(["1", "2", "3"]);
          act(() =>
            result.current.onSelect("4", { button: Triggers.MOUSE_RIGHT_NUMBER }),
          );
          expect(result.current.value).toEqual(["1", "2", "4"]);
        });

        it("should not replace the previous right click entry if it was already in the selection buffer", () => {
          const { result } = renderHook(
            () => useMultipleWrapper({ replaceOnSingle: true, value: ["1", "2"] }),
            {
              wrapper: Wrapper,
            },
          );

          act(() =>
            result.current.onSelect("1", { button: Triggers.MOUSE_RIGHT_NUMBER }),
          );
          expect(result.current.value).toEqual(["1", "2"]);
          act(() =>
            result.current.onSelect("2", { button: Triggers.MOUSE_RIGHT_NUMBER }),
          );
          expect(result.current.value).toEqual(["1", "2"]);
        });
      });

      describe("single selected before", () => {
        it("should replace the selection when you right click a new item", () => {
          const { result } = renderHook(
            () => useMultipleWrapper({ replaceOnSingle: true, value: ["1"] }),
            {
              wrapper: Wrapper,
            },
          );
          act(() =>
            result.current.onSelect("2", { button: Triggers.MOUSE_RIGHT_NUMBER }),
          );
          expect(result.current.value).toEqual(["2"]);
        });
      });
    });
  });

  describe("single selection", () => {
    it("should select one item", () => {
      const { result } = renderHook(useSelectSingleWrapper, { wrapper: Wrapper });
      act(() => result.current.onSelect("1"));
      expect(result.current.value).toEqual("1");
      act(() => result.current.onSelect("2"));
      expect(result.current.value).toEqual("2");
    });

    it("should deselect an item when you click it again", () => {
      const { result } = renderHook(useSelectSingleWrapper, { wrapper: Wrapper });
      act(() => result.current.onSelect("1"));
      act(() => result.current.onSelect("1"));
      expect(result.current.value).toEqual(null);
    });

    describe("not allow none", () => {
      it("should not allow clearing all selections", () => {
        const { result } = renderHook(
          () => useSelectSingleWrapper({ allowNone: false }),
          {
            wrapper: Wrapper,
          },
        );
        act(() => result.current.onSelect("1"));
        act(() => result.current.onSelect("1"));
        expect(result.current.value).toEqual("1");
      });
    });

    it("should clear the selection when clear() is called", () => {
      const { result } = renderHook(useSelectSingleWrapper, { wrapper: Wrapper });
      act(() => result.current.onSelect("1"));
      act(() => result.current.clear());
      expect(result.current.value).toEqual(null);
    });

    describe("autoSelectOnNone", () => {
      it("should auto-select the first item when value is null", () => {
        const { result } = renderHook(
          () => useSelectSingleWrapper({ autoSelectOnNone: true }),
          { wrapper: Wrapper },
        );
        expect(result.current.value).toEqual("1");
      });

      it("should not auto-select when value is not null", () => {
        const { result } = renderHook(
          () => {
            const [value, onChange] = useState<string | undefined>("2");
            const { clear, onSelect } = Select.useSingle<string>({
              autoSelectOnNone: true,
              allowNone: true,
              value: value as string,
              onChange,
            });
            return { value, clear, onSelect };
          },
          { wrapper: Wrapper },
        );
        expect(result.current.value).toEqual("2");
      });

      it("should auto-select after clearing when autoSelectOnNone is true and allowNone is true", () => {
        const { result } = renderHook(
          () => useSelectSingleWrapper({ autoSelectOnNone: true, allowNone: true }),
          { wrapper: Wrapper },
        );
        expect(result.current.value).toEqual("1");
        act(() => result.current.onSelect("2"));
        expect(result.current.value).toEqual("2");
        act(() => result.current.clear());
        expect(result.current.value).toEqual("1");
      });

      it("should not auto-select when autoSelectOnNone is false", () => {
        const { result } = renderHook(
          () => useSelectSingleWrapper({ autoSelectOnNone: false }),
          { wrapper: Wrapper },
        );
        expect(result.current.value).toEqual(undefined);
      });

      it("should auto-select first item when allowNone is false and autoSelectOnNone is true", () => {
        const { result } = renderHook(
          () => useSelectSingleWrapper({ allowNone: false, autoSelectOnNone: true }),
          { wrapper: Wrapper },
        );
        expect(result.current.value).toEqual("1");
      });

      it("should handle empty data gracefully", () => {
        const EmptyWrapper = (props: PropsWithChildren) => (
          <Triggers.Provider>
            <List.Frame data={[]} {...props} />
          </Triggers.Provider>
        );
        const { result } = renderHook(
          () => useSelectSingleWrapper({ autoSelectOnNone: true }),
          { wrapper: EmptyWrapper },
        );
        expect(result.current.value).toEqual(undefined);
      });
    });
  });
});
