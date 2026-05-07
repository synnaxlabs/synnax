// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { useKeysData } from "@/list/useKeysData";

describe("useKeysData", () => {
  describe("basic functionality", () => {
    it("should return the provided data array", () => {
      const data = ["1", "2", "3"];
      const { result } = renderHook(() => useKeysData(data));

      expect(result.current.data).toEqual(data);
    });

    it("should return empty array for empty data", () => {
      const data: string[] = [];
      const { result } = renderHook(() => useKeysData(data));

      expect(result.current.data).toEqual([]);
    });

    it("should handle single item array", () => {
      const data = ["single"];
      const { result } = renderHook(() => useKeysData(data));

      expect(result.current.data).toEqual(["single"]);
    });
  });

  describe("getItem functionality", () => {
    it("should return keyed object for existing key", () => {
      const data = ["apple", "banana", "cherry"];
      const { result } = renderHook(() => useKeysData(data));

      expect(result.current.getItem("apple")).toEqual({ key: "apple" });
      expect(result.current.getItem("banana")).toEqual({ key: "banana" });
      expect(result.current.getItem("cherry")).toEqual({ key: "cherry" });
    });

    it("should return undefined for non-existing key", () => {
      const data = ["apple", "banana", "cherry"];
      const { result } = renderHook(() => useKeysData(data));

      expect(result.current.getItem("nonexistent")).toBeUndefined();
      expect(result.current.getItem("")).toBeUndefined();
    });

    it("should handle empty array data", () => {
      const data: string[] = [];
      const { result } = renderHook(() => useKeysData(data));

      expect(result.current.getItem("anything")).toBeUndefined();
    });

    it("should work with different key types", () => {
      const numberData = [1, 2, 3];
      const { result: numberResult } = renderHook(() => useKeysData(numberData));

      expect(numberResult.current.getItem(1)).toEqual({ key: 1 });
      expect(numberResult.current.getItem(4)).toBeUndefined();

      const stringData = ["a", "b", "c"];
      const { result: stringResult } = renderHook(() => useKeysData(stringData));

      expect(stringResult.current.getItem("a")).toEqual({ key: "a" });
      expect(stringResult.current.getItem("d")).toBeUndefined();
    });
  });

  describe("readonly array handling", () => {
    it("should handle readonly arrays", () => {
      const data = ["1", "2", "3"] as const;
      const { result } = renderHook(() => useKeysData(data));

      expect(result.current.data).toEqual(["1", "2", "3"]);
      expect(result.current.getItem("1")).toEqual({ key: "1" });
    });
  });

  describe("data changes", () => {
    it("should update when data changes", () => {
      const initialData = ["1", "2"];
      const { result, rerender } = renderHook(({ data }) => useKeysData(data), {
        initialProps: { data: initialData },
      });

      expect(result.current.data).toEqual(["1", "2"]);
      expect(result.current.getItem("1")).toEqual({ key: "1" });
      expect(result.current.getItem("3")).toBeUndefined();

      const newData = ["1", "2", "3"];
      rerender({ data: newData });

      expect(result.current.data).toEqual(["1", "2", "3"]);
      expect(result.current.getItem("3")).toEqual({ key: "3" });
    });

    it("should handle data removal", () => {
      const initialData = ["1", "2", "3"];
      const { result, rerender } = renderHook(({ data }) => useKeysData(data), {
        initialProps: { data: initialData },
      });

      expect(result.current.getItem("2")).toEqual({ key: "2" });

      const newData = ["1", "3"];
      rerender({ data: newData });

      expect(result.current.data).toEqual(["1", "3"]);
      expect(result.current.getItem("2")).toBeUndefined();
      expect(result.current.getItem("1")).toEqual({ key: "1" });
      expect(result.current.getItem("3")).toEqual({ key: "3" });
    });
  });

  describe("duplicate keys", () => {
    it("should handle duplicate keys in array", () => {
      const data = ["apple", "banana", "apple"];
      const { result } = renderHook(() => useKeysData(data));

      expect(result.current.data).toEqual(["apple", "banana", "apple"]);
      expect(result.current.getItem("apple")).toEqual({ key: "apple" });
    });
  });

  describe("edge cases", () => {
    it("should maintain referential stability of getItem", () => {
      const data = ["1", "2", "3"];
      const { result, rerender } = renderHook(() => useKeysData(data));

      const firstGetItem = result.current.getItem;

      rerender();

      expect(result.current.getItem).toBe(firstGetItem);
    });

    it("should update getItem when data changes", () => {
      const initialData = ["1", "2"];
      const { result, rerender } = renderHook(({ data }) => useKeysData(data), {
        initialProps: { data: initialData },
      });

      const firstGetItem = result.current.getItem;

      const newData = ["3", "4"];
      rerender({ data: newData });

      expect(result.current.getItem).not.toBe(firstGetItem);
      expect(result.current.getItem("3")).toEqual({ key: "3" });
      expect(result.current.getItem("1")).toBeUndefined();
    });
  });
});
