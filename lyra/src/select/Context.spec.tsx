// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement, useState } from "react";
import { describe, expect, it, vi } from "vitest";

import {
  Context,
  type ContextProps,
  useClear,
  useItemState,
  useSelected,
  useSelectedAmong,
} from "@/select/Context";

const staticWrapper = (props: Omit<ContextProps, "children">) => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Context {...props}>{children}</Context>
  );
  Wrapper.displayName = "Wrapper";
  return Wrapper;
};

describe("Selection", () => {
  describe("useItemState", () => {
    it("should mark the item matching a single selected value", () => {
      const selected = renderHook(() => useItemState("a"), {
        wrapper: staticWrapper({ value: "a" }),
      });
      const unselected = renderHook(() => useItemState("b"), {
        wrapper: staticWrapper({ value: "a" }),
      });
      expect(selected.result.current.selected).toBe(true);
      expect(unselected.result.current.selected).toBe(false);
    });

    it("should mark every item included in an array value", () => {
      const { result } = renderHook(() => useItemState("c"), {
        wrapper: staticWrapper({ value: ["a", "c"] }),
      });
      expect(result.current.selected).toBe(true);
    });

    it("should mark the hovered item", () => {
      const { result } = renderHook(() => useItemState("b"), {
        wrapper: staticWrapper({ value: "a", hover: "b" }),
      });
      expect(result.current.hovered).toBe(true);
      expect(result.current.selected).toBe(false);
    });

    it("should focus only the head of an ordered multi-selection", () => {
      const head = renderHook(() => useItemState("a"), {
        wrapper: staticWrapper({ value: ["a", "b"] }),
      });
      const tail = renderHook(() => useItemState("b"), {
        wrapper: staticWrapper({ value: ["a", "b"] }),
      });
      expect(head.result.current.focused).toBe(true);
      expect(tail.result.current.focused).toBe(false);
    });

    it("should not focus any item in a scalar selection", () => {
      const { result } = renderHook(() => useItemState("a"), {
        wrapper: staticWrapper({ value: "a" }),
      });
      expect(result.current.selected).toBe(true);
      expect(result.current.focused).toBe(false);
    });

    it("should mark only the head of an ordered multi-selection", () => {
      const head = renderHook(() => useItemState("a"), {
        wrapper: staticWrapper({ value: ["a", "b"] }),
      });
      const tail = renderHook(() => useItemState("b"), {
        wrapper: staticWrapper({ value: ["a", "b"] }),
      });
      expect(head.result.current.head).toBe(true);
      expect(tail.result.current.head).toBe(false);
    });

    it("should mark a scalar selection's value as the head", () => {
      const { result } = renderHook(() => useItemState("a"), {
        wrapper: staticWrapper({ value: "a" }),
      });
      expect(result.current.head).toBe(true);
      expect(result.current.focused).toBe(false);
    });

    it("should move focus to the new head when the selection is reordered", () => {
      let setValue: (value: string[]) => void = () => {};
      const wrapper = ({ children }: PropsWithChildren): ReactElement => {
        const [value, setState] = useState<string[]>(["a", "b"]);
        setValue = setState;
        return <Context value={value}>{children}</Context>;
      };
      const { result } = renderHook(
        () => ({ a: useItemState("a"), b: useItemState("b") }),
        { wrapper },
      );
      expect(result.current.a.focused).toBe(true);
      expect(result.current.b.focused).toBe(false);
      act(() => setValue(["b", "a"]));
      expect(result.current.a.focused).toBe(false);
      expect(result.current.b.focused).toBe(true);
    });

    it("should call the provider's onSelect with the item's key", () => {
      const onSelect = vi.fn();
      const { result } = renderHook(() => useItemState("b"), {
        wrapper: staticWrapper({ value: "a", onSelect }),
      });
      act(() => result.current.onSelect());
      expect(onSelect).toHaveBeenCalledTimes(1);
      expect(onSelect).toHaveBeenCalledWith("b");
    });

    it("should report no selection when used outside a provider", () => {
      const { result } = renderHook(() => useItemState("a"));
      expect(result.current.selected).toBe(false);
      expect(result.current.focused).toBe(false);
      expect(result.current.head).toBe(false);
      expect(result.current.hovered).toBe(false);
    });

    it("should re-render an item only when its own selection flips", () => {
      let renders = 0;
      let setValue: (value: string) => void = () => {};
      const wrapper = ({ children }: PropsWithChildren): ReactElement => {
        const [value, setState] = useState("a");
        setValue = setState;
        return <Context value={value}>{children}</Context>;
      };
      renderHook(
        () => {
          renders++;
          return useItemState("c");
        },
        { wrapper },
      );
      const baseline = renders;
      act(() => setValue("b"));
      expect(renders).toBe(baseline);
      act(() => setValue("c"));
      expect(renders).toBeGreaterThan(baseline);
    });

    it("should re-render an item only when its own hover flips", () => {
      let renders = 0;
      let setHover: (key: string) => void = () => {};
      const wrapper = ({ children }: PropsWithChildren): ReactElement => {
        const [hover, setState] = useState("a");
        setHover = setState;
        return (
          <Context value={undefined} hover={hover}>
            {children}
          </Context>
        );
      };
      const { result } = renderHook(
        () => {
          renders++;
          return useItemState("b");
        },
        { wrapper },
      );
      expect(result.current.hovered).toBe(false);
      const baseline = renders;
      act(() => setHover("c"));
      expect(renders).toBe(baseline);
      act(() => setHover("b"));
      expect(result.current.hovered).toBe(true);
      expect(renders).toBeGreaterThan(baseline);
    });
  });

  describe("useSelectedAmong", () => {
    it("should return the selected key among the given keys", () => {
      const { result } = renderHook(() => useSelectedAmong(["b", "c"]), {
        wrapper: staticWrapper({ value: ["a", "c"] }),
      });
      expect(result.current).toEqual("c");
    });

    it("should return undefined when none of the keys is selected", () => {
      const { result } = renderHook(() => useSelectedAmong(["b"]), {
        wrapper: staticWrapper({ value: ["a"] }),
      });
      expect(result.current).toBeUndefined();
    });

    it("should prefer the earliest key in the selection's order", () => {
      const { result } = renderHook(() => useSelectedAmong(["a", "b"]), {
        wrapper: staticWrapper({ value: ["b", "a"] }),
      });
      expect(result.current).toEqual("b");
    });

    it("should track changes to the given keys' selection", () => {
      let setValue: (value: string[]) => void = () => {};
      const wrapper = ({ children }: PropsWithChildren): ReactElement => {
        const [value, setState] = useState(["a"]);
        setValue = setState;
        return <Context value={value}>{children}</Context>;
      };
      const { result } = renderHook(() => useSelectedAmong(["a", "b"]), {
        wrapper,
      });
      expect(result.current).toEqual("a");
      act(() => setValue(["b"]));
      expect(result.current).toEqual("b");
      act(() => setValue([]));
      expect(result.current).toBeUndefined();
    });
  });

  describe("useSelected", () => {
    it("should return a single value as a one-element array", () => {
      const { result } = renderHook(() => useSelected(), {
        wrapper: staticWrapper({ value: "a" }),
      });
      expect(result.current).toEqual(["a"]);
    });

    it("should track selection changes", () => {
      let setValue: (value: string[]) => void = () => {};
      const wrapper = ({ children }: PropsWithChildren): ReactElement => {
        const [value, setState] = useState<string[]>(["a"]);
        setValue = setState;
        return <Context value={value}>{children}</Context>;
      };
      const { result } = renderHook(() => useSelected(), { wrapper });
      expect(result.current).toEqual(["a"]);
      act(() => setValue(["a", "b"]));
      expect(result.current).toEqual(["a", "b"]);
    });

    it("should return an empty array when nothing is selected", () => {
      const { result } = renderHook(() => useSelected(), {
        wrapper: staticWrapper({ value: undefined }),
      });
      expect(result.current).toEqual([]);
    });
  });

  describe("useClear", () => {
    it("should invoke the provider's clear callback", () => {
      const clear = vi.fn();
      const { result } = renderHook(() => useClear(), {
        wrapper: staticWrapper({ value: "a", clear }),
      });
      result.current();
      expect(clear).toHaveBeenCalledTimes(1);
    });
  });
});
