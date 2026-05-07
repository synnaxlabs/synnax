// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  type RetrieveParams,
  useStaticData,
  type UseStaticDataArgs,
} from "@/list/useStaticData";

interface TestItem {
  key: string;
  name: string;
  category: string;
  value: number;
}

const mockData: TestItem[] = [
  { key: "1", name: "Apple", category: "fruit", value: 10 },
  { key: "2", name: "Banana", category: "fruit", value: 5 },
  { key: "3", name: "Carrot", category: "vegetable", value: 8 },
  { key: "4", name: "Broccoli", category: "vegetable", value: 12 },
  { key: "5", name: "Apricot", category: "fruit", value: 7 },
];

describe("useStaticData", () => {
  describe("basic functionality", () => {
    it("should return all data keys initially", () => {
      const { result } = renderHook(() =>
        useStaticData<string, TestItem>({ data: mockData }),
      );

      expect(result.current.data).toEqual(["1", "2", "3", "4", "5"]);
    });

    it("should return empty array for empty data", () => {
      const { result } = renderHook(() =>
        useStaticData<string, TestItem>({ data: [] }),
      );

      expect(result.current.data).toEqual([]);
    });

    it("should provide getItem function that retrieves items by key", () => {
      const { result } = renderHook(() =>
        useStaticData<string, TestItem>({ data: mockData }),
      );

      expect(result.current.getItem("1")).toEqual({
        key: "1",
        name: "Apple",
        category: "fruit",
        value: 10,
      });
      expect(result.current.getItem("3")).toEqual({
        key: "3",
        name: "Carrot",
        category: "vegetable",
        value: 8,
      });
      expect(result.current.getItem("nonexistent")).toBeUndefined();
    });
  });

  describe("search functionality", () => {
    it("should filter data based on search term", () => {
      const { result } = renderHook(() =>
        useStaticData<string, TestItem>({ data: mockData }),
      );

      act(() => {
        result.current.retrieve({ searchTerm: "Apple" });
      });

      expect(result.current.data).toEqual(["1"]);
    });

    it("should perform fuzzy search", () => {
      const { result } = renderHook(() =>
        useStaticData<string, TestItem>({ data: mockData }),
      );
      act(() => {
        result.current.retrieve({ searchTerm: "Ap" });
      });

      expect(result.current.data).toContain("5");
      expect(result.current.data).toContain("1");
    });

    it("should return all items when search term is empty", () => {
      const { result } = renderHook(() =>
        useStaticData<string, TestItem>({ data: mockData }),
      );

      act(() => {
        result.current.retrieve({ searchTerm: "" });
      });

      expect(result.current.data).toEqual(["1", "2", "3", "4", "5"]);
    });

    it("should return empty results for non-matching search term", () => {
      const { result } = renderHook(() =>
        useStaticData<string, TestItem>({ data: mockData }),
      );

      act(() => {
        result.current.retrieve({ searchTerm: "xyz123" });
      });

      expect(result.current.data).toEqual([]);
    });

    it("should search across multiple fields", () => {
      const { result } = renderHook(() =>
        useStaticData<string, TestItem>({ data: mockData }),
      );
      act(() => {
        result.current.retrieve({ searchTerm: "vegetable" });
      });
      expect(result.current.data).toContain("3");
      expect(result.current.data).toContain("4");
    });
  });

  describe("filter functionality", () => {
    it("should apply custom filter function", () => {
      const filter = (item: TestItem) => item.category === "fruit";

      const { result } = renderHook(() =>
        useStaticData<string, TestItem>({ data: mockData, filter }),
      );

      expect(result.current.data).toEqual(["1", "2", "5"]);
    });

    it("should apply filter with search parameters", () => {
      const filter = (item: TestItem, _params: RetrieveParams) => item.value > 8;

      const { result } = renderHook(() =>
        useStaticData<string, TestItem>({ data: mockData, filter }),
      );

      expect(result.current.data).toEqual(["1", "4"]);
    });

    it("should combine search and filter", () => {
      const filter = (item: TestItem) => item.category === "fruit";

      const { result } = renderHook(() =>
        useStaticData<string, TestItem>({ data: mockData, filter }),
      );

      act(() => {
        result.current.retrieve({ searchTerm: "A" });
      });

      expect(result.current.data).toContain("1");
      expect(result.current.data).toContain("5");
      expect(result.current.data).not.toContain("3");
    });
  });

  describe("retrieve function", () => {
    it("should update search parameters", () => {
      const { result } = renderHook(() =>
        useStaticData<string, TestItem>({ data: mockData }),
      );

      const initialData = result.current.data;

      act(() => {
        result.current.retrieve({ searchTerm: "Apple" });
      });

      expect(result.current.data).not.toEqual(initialData);
      expect(result.current.data).toEqual(["1"]);
    });

    it("should handle partial parameter updates", () => {
      const { result } = renderHook(() =>
        useStaticData<string, TestItem>({ data: mockData }),
      );

      act(() => {
        result.current.retrieve({ searchTerm: "Apple", offset: 0 });
      });

      expect(result.current.data).toEqual(["1"]);

      act(() => {
        result.current.retrieve({ searchTerm: "Banana" });
      });

      expect(result.current.data).toEqual(["2"]);
    });

    it("should clear search when term is undefined", () => {
      const { result } = renderHook(() =>
        useStaticData<string, TestItem>({ data: mockData }),
      );

      act(() => {
        result.current.retrieve({ searchTerm: "Apple" });
      });

      expect(result.current.data).toEqual(["1"]);

      act(() => {
        result.current.retrieve({ searchTerm: undefined });
      });

      expect(result.current.data).toEqual(["1", "2", "3", "4", "5"]);
    });
  });

  describe("sorting functionality", () => {
    it("should sort data using the provided comparator", () => {
      const sortByName = (a: TestItem, b: TestItem) => a.name.localeCompare(b.name);

      const { result } = renderHook(() =>
        useStaticData<string, TestItem>({ data: mockData, sort: sortByName }),
      );

      expect(result.current.data).toEqual(["1", "5", "2", "4", "3"]);
    });

    it("should sort by value in descending order", () => {
      const sortByValueDesc = (a: TestItem, b: TestItem) => b.value - a.value;

      const { result } = renderHook(() =>
        useStaticData<string, TestItem>({ data: mockData, sort: sortByValueDesc }),
      );

      expect(result.current.data).toEqual(["4", "1", "3", "5", "2"]);
    });

    it("should maintain sorting when searching", () => {
      const sortByName = (a: TestItem, b: TestItem) => a.name.localeCompare(b.name);

      const { result } = renderHook(() =>
        useStaticData<string, TestItem>({ data: mockData, sort: sortByName }),
      );

      act(() => {
        result.current.retrieve({ searchTerm: "Ap" });
      });

      expect(result.current.data).toEqual(["1", "5"]);
    });

    it("should combine filter and sort", () => {
      const filter = (item: TestItem) => item.category === "fruit";
      const sortByValue = (a: TestItem, b: TestItem) => a.value - b.value;

      const { result } = renderHook(() =>
        useStaticData<string, TestItem>({ data: mockData, filter, sort: sortByValue }),
      );

      expect(result.current.data).toEqual(["2", "5", "1"]);
    });

    it("should work without sorting function", () => {
      const { result } = renderHook(() =>
        useStaticData<string, TestItem>({ data: mockData }),
      );

      expect(result.current.data).toEqual(["1", "2", "3", "4", "5"]);
    });
  });

  describe("edge cases", () => {
    it("should handle data updates", () => {
      const { result, rerender } = renderHook(
        ({ data }: UseStaticDataArgs<string, TestItem>) =>
          useStaticData<string, TestItem>({ data }),
        { initialProps: { data: mockData.slice(0, 2) } },
      );

      expect(result.current.data).toEqual(["1", "2"]);

      rerender({ data: mockData });

      expect(result.current.data).toEqual(["1", "2", "3", "4", "5"]);
    });

    it("should handle search with single character", () => {
      const { result } = renderHook(() =>
        useStaticData<string, TestItem>({ data: mockData }),
      );

      act(() => {
        result.current.retrieve({ searchTerm: "A" });
      });

      expect(result.current.data.length).toBeGreaterThan(0);
      expect(result.current.data).toContain("1");
    });

    it("should maintain getItem functionality after search", () => {
      const { result } = renderHook(() =>
        useStaticData<string, TestItem>({ data: mockData }),
      );

      act(() => {
        result.current.retrieve({ searchTerm: "Apple" });
      });

      expect(result.current.getItem("1")).toBeDefined();
      expect(result.current.getItem("2")).toBeDefined();
    });
  });
});
