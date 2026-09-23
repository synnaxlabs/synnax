// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { useMapData } from "@/list/useMapData";

interface TestItem {
  key: string;
  name: string;
  value: number;
}

describe("useMapData", () => {
  describe("setItem functionality", () => {
    it("should set a single item", () => {
      const { result } = renderHook(() => useMapData<string, TestItem>());

      const item: TestItem = { key: "1", name: "Apple", value: 10 };

      act(() => {
        result.current.setItem(item);
      });

      expect(result.current.getItem("1")).toEqual(item);
    });

    it("should set multiple items", () => {
      const { result } = renderHook(() => useMapData<string, TestItem>());

      const items: TestItem[] = [
        { key: "1", name: "Apple", value: 10 },
        { key: "2", name: "Banana", value: 5 },
      ];

      act(() => {
        result.current.setItem(items);
      });

      expect(result.current.getItem("1")).toEqual(items[0]);
      expect(result.current.getItem("2")).toEqual(items[1]);
    });

    it("should update existing item", () => {
      const { result } = renderHook(() => useMapData<string, TestItem>());

      const originalItem: TestItem = { key: "1", name: "Apple", value: 10 };
      const updatedItem: TestItem = { key: "1", name: "Green Apple", value: 15 };

      act(() => {
        result.current.setItem(originalItem);
      });

      expect(result.current.getItem("1")).toEqual(originalItem);

      act(() => {
        result.current.setItem(updatedItem);
      });

      expect(result.current.getItem("1")).toEqual(updatedItem);
    });
  });

  describe("deleteItem functionality", () => {
    it("should delete a single item", () => {
      const { result } = renderHook(() => useMapData<string, TestItem>());

      const item: TestItem = { key: "1", name: "Apple", value: 10 };

      act(() => {
        result.current.setItem(item);
      });

      expect(result.current.getItem("1")).toEqual(item);

      act(() => {
        result.current.deleteItem("1");
      });

      expect(result.current.getItem("1")).toBeUndefined();
    });

    it("should delete multiple items", () => {
      const { result } = renderHook(() => useMapData<string, TestItem>());

      const items: TestItem[] = [
        { key: "1", name: "Apple", value: 10 },
        { key: "2", name: "Banana", value: 5 },
        { key: "3", name: "Cherry", value: 7 },
      ];

      act(() => {
        result.current.setItem(items);
      });

      act(() => {
        result.current.deleteItem(["1", "3"]);
      });

      expect(result.current.getItem("1")).toBeUndefined();
      expect(result.current.getItem("2")).toEqual(items[1]);
      expect(result.current.getItem("3")).toBeUndefined();
    });

    it("should handle deleting non-existent item", () => {
      const { result } = renderHook(() => useMapData<string, TestItem>());

      act(() => {
        result.current.deleteItem("nonexistent");
      });

      expect(result.current.getItem("nonexistent")).toBeUndefined();
    });
  });

  describe("getItem functionality", () => {
    it("should return undefined for non-existent item", () => {
      const { result } = renderHook(() => useMapData<string, TestItem>());

      expect(result.current.getItem("nonexistent")).toBeUndefined();
    });

    it("should handle undefined key", () => {
      const { result } = renderHook(() => useMapData<string, TestItem>());

      expect(result.current.getItem(undefined)).toBeUndefined();
    });

    it("should get multiple items by array of keys", () => {
      const { result } = renderHook(() => useMapData<string, TestItem>());

      const items: TestItem[] = [
        { key: "1", name: "Apple", value: 10 },
        { key: "2", name: "Banana", value: 5 },
        { key: "3", name: "Cherry", value: 7 },
      ];

      act(() => {
        result.current.setItem(items);
      });

      const retrieved = result.current.getItem(["1", "3"]);
      expect(retrieved).toEqual([items[0], items[2]]);
    });

    it("should filter out non-existent items when getting multiple", () => {
      const { result } = renderHook(() => useMapData<string, TestItem>());

      const item: TestItem = { key: "1", name: "Apple", value: 10 };

      act(() => {
        result.current.setItem(item);
      });

      const retrieved = result.current.getItem(["1", "nonexistent", "2"]);
      expect(retrieved).toEqual([item]);
    });

    it("should return empty array for empty key array", () => {
      const { result } = renderHook(() => useMapData<string, TestItem>());

      const retrieved = result.current.getItem([]);
      expect(retrieved).toEqual([]);
    });
  });

  describe("hasItem functionality", () => {
    it("should return true for existing item", () => {
      const { result } = renderHook(() => useMapData<string, TestItem>());

      const item: TestItem = { key: "1", name: "Apple", value: 10 };

      act(() => {
        result.current.setItem(item);
      });

      expect(result.current.hasItem("1")).toBe(true);
    });

    it("should return false for non-existing item", () => {
      const { result } = renderHook(() => useMapData<string, TestItem>());

      expect(result.current.hasItem("nonexistent")).toBe(false);
    });

    it("should return false after item is deleted", () => {
      const { result } = renderHook(() => useMapData<string, TestItem>());

      const item: TestItem = { key: "1", name: "Apple", value: 10 };

      act(() => {
        result.current.setItem(item);
      });

      expect(result.current.hasItem("1")).toBe(true);

      act(() => {
        result.current.deleteItem("1");
      });

      expect(result.current.hasItem("1")).toBe(false);
    });
  });

  describe("subscribe functionality", () => {
    it("should notify listener when subscribed item changes", async () => {
      const { result } = renderHook(() => useMapData<string, TestItem>());

      const listener = vi.fn();

      const unsubscribe = await act(async () =>
        result.current.subscribe(listener, "1"),
      );

      const item: TestItem = { key: "1", name: "Apple", value: 10 };

      act(() => {
        result.current.setItem(item);
      });

      expect(listener).toHaveBeenCalledTimes(1);

      unsubscribe();
    });

    it("should notify listener when subscribed item is deleted", async () => {
      const { result } = renderHook(() => useMapData<string, TestItem>());

      const item: TestItem = { key: "1", name: "Apple", value: 10 };

      act(() => {
        result.current.setItem(item);
      });

      const listener = vi.fn();

      const unsubscribe = await act(async () =>
        result.current.subscribe(listener, "1"),
      );

      act(() => {
        result.current.deleteItem("1");
      });

      expect(listener).toHaveBeenCalledTimes(1);

      unsubscribe();
    });

    it("should not notify listener for different key", async () => {
      const { result } = renderHook(() => useMapData<string, TestItem>());

      const listener = vi.fn();

      const unsubscribe = await act(async () =>
        result.current.subscribe(listener, "1"),
      );

      const item: TestItem = { key: "2", name: "Banana", value: 5 };

      act(() => {
        result.current.setItem(item);
      });

      expect(listener).not.toHaveBeenCalled();

      unsubscribe();
    });

    it("should handle unsubscription", async () => {
      const { result } = renderHook(() => useMapData<string, TestItem>());

      const listener = vi.fn();

      const unsubscribe = await act(async () =>
        result.current.subscribe(listener, "1"),
      );

      const item: TestItem = { key: "1", name: "Apple", value: 10 };

      act(() => {
        result.current.setItem(item);
      });

      expect(listener).toHaveBeenCalledTimes(1);

      act(() => {
        unsubscribe();
      });

      act(() => {
        result.current.setItem({ ...item, value: 15 });
      });

      expect(listener).toHaveBeenCalledTimes(1); // Still only called once
    });

    it("should notify multiple listeners for same key", async () => {
      const { result } = renderHook(() => useMapData<string, TestItem>());

      const listener1 = vi.fn();
      const listener2 = vi.fn();

      const unsubscribe1 = await act(async () =>
        result.current.subscribe(listener1, "1"),
      );
      const unsubscribe2 = await act(async () =>
        result.current.subscribe(listener2, "1"),
      );

      const item: TestItem = { key: "1", name: "Apple", value: 10 };

      act(() => {
        result.current.setItem(item);
      });

      expect(listener1).toHaveBeenCalledTimes(1);
      expect(listener2).toHaveBeenCalledTimes(1);

      unsubscribe1();
      unsubscribe2();
    });
  });

  describe("edge cases", () => {
    it("should handle different key types", () => {
      const { result: numberResult } = renderHook(() =>
        useMapData<number, { key: number; value: string }>(),
      );

      const numberItem = { key: 1, value: "first" };

      act(() => {
        numberResult.current.setItem(numberItem);
      });

      expect(numberResult.current.getItem(1)).toEqual(numberItem);
      expect(numberResult.current.hasItem(1)).toBe(true);
    });

    it("should handle bulk operations", () => {
      const { result } = renderHook(() => useMapData<string, TestItem>());

      const items: TestItem[] = Array.from({ length: 100 }, (_, i) => ({
        key: `item-${i}`,
        name: `Item ${i}`,
        value: i,
      }));

      act(() => {
        result.current.setItem(items);
      });

      expect(result.current.hasItem("item-0")).toBe(true);
      expect(result.current.hasItem("item-99")).toBe(true);

      const keysToDelete = items.slice(0, 50).map((item) => item.key);

      act(() => {
        result.current.deleteItem(keysToDelete);
      });

      expect(result.current.hasItem("item-0")).toBe(false);
      expect(result.current.hasItem("item-49")).toBe(false);
      expect(result.current.hasItem("item-50")).toBe(true);
    });

    it("should maintain referential stability", () => {
      const { result, rerender } = renderHook(() => useMapData<string, TestItem>());

      const initialSetItem = result.current.setItem;
      const initialDeleteItem = result.current.deleteItem;
      const initialGetItem = result.current.getItem;
      const initialHasItem = result.current.hasItem;
      const initialSubscribe = result.current.subscribe;

      rerender();

      expect(result.current.setItem).toBe(initialSetItem);
      expect(result.current.deleteItem).toBe(initialDeleteItem);
      expect(result.current.getItem).toBe(initialGetItem);
      expect(result.current.hasItem).toBe(initialHasItem);
      expect(result.current.subscribe).toBe(initialSubscribe);
    });
  });
});
