// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { type FrameProps, type GetItem } from "@/list/Frame";
import { useCombinedData } from "@/list/useCombinedData";

interface TestItem {
  key: string;
  name: string;
  value: number;
}

describe("useCombinedData", () => {
  describe("data combination", () => {
    it("should combine data arrays from both sources", () => {
      const first: Pick<
        FrameProps<string, TestItem>,
        "data" | "getItem" | "subscribe"
      > = {
        data: ["1", "2"],
      };

      const second: Pick<
        FrameProps<string, TestItem>,
        "data" | "getItem" | "subscribe"
      > = {
        data: ["3", "4"],
      };

      const { result } = renderHook(() =>
        useCombinedData<string, TestItem>({ first, second }),
      );

      expect(result.current.data).toEqual(["1", "2", "3", "4"]);
    });

    it("should handle empty data arrays", () => {
      const first: Pick<FrameProps<string>, "data"> = { data: [] };
      const second: Pick<FrameProps<string>, "data"> = { data: [] };

      const { result } = renderHook(() => useCombinedData({ first, second }));

      expect(result.current.data).toEqual([]);
    });

    it("should update when source data changes", () => {
      const { result, rerender } = renderHook(
        ({ first, second }) => useCombinedData({ first, second }),
        {
          initialProps: {
            first: { data: ["1"] },
            second: { data: ["2"] },
          },
        },
      );

      expect(result.current.data).toEqual(["1", "2"]);

      rerender({
        first: { data: ["1", "3"] },
        second: { data: ["2", "4"] },
      });

      expect(result.current.data).toEqual(["1", "3", "2", "4"]);
    });
  });

  describe("getItem", () => {
    it("should retrieve items from first source first", () => {
      const item1: TestItem = { key: "1", name: "First", value: 100 };
      const item2: TestItem = { key: "2", name: "Second", value: 200 };

      const first: Pick<FrameProps<string, TestItem>, "data" | "getItem"> = {
        data: ["1"],
        getItem: vi.fn((key: string | string[]) => {
          if (Array.isArray(key) && key.includes("1")) return [item1];
          if (key === "1") return item1;
          return undefined;
        }) as GetItem<string, TestItem>,
      };

      const second: Pick<FrameProps<string, TestItem>, "data" | "getItem"> = {
        data: ["2"],
        getItem: vi.fn((key: string | string[]) => {
          if (Array.isArray(key) && key.includes("2")) return [item2];
          if (key === "2") return item2;
          return undefined;
        }) as GetItem<string, TestItem>,
      };

      const { result } = renderHook(() =>
        useCombinedData<string, TestItem>({ first, second }),
      );

      expect(result.current.getItem?.("1")).toEqual(item1);
      expect(first.getItem).toHaveBeenCalledWith("1");

      expect(result.current.getItem?.("2")).toEqual(item2);
      expect(first.getItem).toHaveBeenCalledWith("2");
      expect(second.getItem).toHaveBeenCalledWith("2");
    });

    it("should handle duplicate keys by preferring first source", () => {
      const firstItem: TestItem = { key: "1", name: "From First", value: 100 };
      const secondItem: TestItem = { key: "1", name: "From Second", value: 200 };

      const first: Pick<FrameProps<string, TestItem>, "data" | "getItem"> = {
        data: ["1"],
        getItem: vi.fn((key: string | string[]) => {
          if (Array.isArray(key)) return [firstItem];
          return firstItem;
        }) as GetItem<string, TestItem>,
      };

      const second: Pick<FrameProps<string, TestItem>, "data" | "getItem"> = {
        data: ["1"],
        getItem: vi.fn((key: string | string[]) => {
          if (Array.isArray(key)) return [secondItem];
          return secondItem;
        }) as GetItem<string, TestItem>,
      };

      const { result } = renderHook(() =>
        useCombinedData<string, TestItem>({ first, second }),
      );

      const item = result.current.getItem?.("1");
      expect(item).toEqual(firstItem);
      expect(first.getItem).toHaveBeenCalled();
      expect(second.getItem).not.toHaveBeenCalled();
    });

    it("should handle missing getItem functions", () => {
      const first: Pick<FrameProps<string>, "data"> = { data: ["1"] };
      const second: Pick<FrameProps<string>, "data"> = { data: ["2"] };

      const { result } = renderHook(() => useCombinedData({ first, second }));

      expect(result.current.getItem?.("1")).toBeUndefined();
    });
  });

  describe("subscribe", () => {
    it("should subscribe to both sources and return combined unsubscribe", () => {
      const callback = vi.fn();
      const firstUnsub = vi.fn();
      const secondUnsub = vi.fn();

      const first: Pick<FrameProps<string>, "data" | "subscribe"> = {
        data: ["1"],
        subscribe: vi.fn(() => firstUnsub),
      };

      const second: Pick<FrameProps<string>, "data" | "subscribe"> = {
        data: ["2"],
        subscribe: vi.fn(() => secondUnsub),
      };

      const { result } = renderHook(() => useCombinedData({ first, second }));

      const unsubscribe = result.current.subscribe?.(callback, "1");

      expect(first.subscribe).toHaveBeenCalledWith(callback, "1");
      expect(second.subscribe).toHaveBeenCalledWith(callback, "1");

      unsubscribe?.();

      expect(firstUnsub).toHaveBeenCalled();
      expect(secondUnsub).toHaveBeenCalled();
    });

    it("should handle partial subscribe functions", () => {
      const callback = vi.fn();
      const firstUnsub = vi.fn();

      const first: Pick<FrameProps<string>, "data" | "subscribe"> = {
        data: ["1"],
        subscribe: vi.fn(() => firstUnsub),
      };

      const second: Pick<FrameProps<string>, "data"> = {
        data: ["2"],
      };

      const { result } = renderHook(() => useCombinedData({ first, second }));

      const unsubscribe = result.current.subscribe?.(callback, "1");

      expect(first.subscribe).toHaveBeenCalledWith(callback, "1");

      unsubscribe?.();

      expect(firstUnsub).toHaveBeenCalled();
    });

    it("should handle missing subscribe functions", () => {
      const first: Pick<FrameProps<string>, "data"> = { data: ["1"] };
      const second: Pick<FrameProps<string>, "data"> = { data: ["2"] };

      const { result } = renderHook(() => useCombinedData({ first, second }));

      const callback = vi.fn();
      const unsubscribe = result.current.subscribe?.(callback, "1");

      expect(unsubscribe).toBeDefined();
      expect(() => unsubscribe?.()).not.toThrow();
    });
  });

  describe("memoization", () => {
    it("should memoize getItem callback", () => {
      const first: Pick<FrameProps<string>, "data" | "getItem"> = {
        data: ["1"],
        getItem: vi.fn(),
      };

      const second: Pick<FrameProps<string>, "data" | "getItem"> = {
        data: ["2"],
        getItem: vi.fn(),
      };

      const { result, rerender } = renderHook(
        ({ first, second }) => useCombinedData({ first, second }),
        { initialProps: { first, second } },
      );

      const initialGetItem = result.current.getItem;

      rerender({ first, second });

      expect(result.current.getItem).toBe(initialGetItem);
    });

    it("should memoize subscribe callback", () => {
      const first: Pick<FrameProps<string>, "data" | "subscribe"> = {
        data: ["1"],
        subscribe: vi.fn(),
      };

      const second: Pick<FrameProps<string>, "data" | "subscribe"> = {
        data: ["2"],
        subscribe: vi.fn(),
      };

      const { result, rerender } = renderHook(
        ({ first, second }) => useCombinedData({ first, second }),
        { initialProps: { first, second } },
      );

      const initialSubscribe = result.current.subscribe;

      rerender({ first, second });

      expect(result.current.subscribe).toBe(initialSubscribe);
    });
  });

  describe("integration scenarios", () => {
    it("should work as a complete FrameProps implementation", () => {
      const items = new Map<string, TestItem>([
        ["1", { key: "1", name: "Item 1", value: 100 }],
        ["2", { key: "2", name: "Item 2", value: 200 }],
        ["3", { key: "3", name: "Item 3", value: 300 }],
      ]);

      const subscribers = new Map<string, Set<() => void>>();

      const createFrameProps = (
        keys: string[],
      ): Pick<FrameProps<string, TestItem>, "data" | "getItem" | "subscribe"> => ({
        data: keys,
        getItem: ((key: string | string[]) => {
          if (Array.isArray(key))
            return key.map((k) => items.get(k)).filter((i) => i != null);
          return items.get(key);
        }) as GetItem<string, TestItem>,
        subscribe: (callback, key) => {
          if (!subscribers.has(key)) subscribers.set(key, new Set());

          subscribers.get(key)!.add(callback);
          return () => subscribers.get(key)?.delete(callback);
        },
      });

      const first = createFrameProps(["1", "2"]);
      const second = createFrameProps(["3"]);

      const { result } = renderHook(() =>
        useCombinedData<string, TestItem>({ first, second }),
      );

      expect(result.current.data).toEqual(["1", "2", "3"]);
      expect(result.current.getItem?.("1")).toEqual({
        key: "1",
        name: "Item 1",
        value: 100,
      });
      expect(result.current.getItem?.("3")).toEqual({
        key: "3",
        name: "Item 3",
        value: 300,
      });

      const callback = vi.fn();
      const unsub = result.current.subscribe?.(callback, "2");
      expect(subscribers.get("2")?.size).toBe(1);

      unsub?.();
      expect(subscribers.get("2")?.size).toBe(0);
    });
  });
});
