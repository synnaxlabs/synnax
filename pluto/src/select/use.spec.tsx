// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, renderHook } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it } from "vitest";

import { Select } from "@/select";

interface UseSelectMultipleWrapperReturn {
  value: string[];
  clear: () => void;
  onSelect: (key: string) => void;
}

const useMultipleWrapper = (
  props: Omit<Select.UseMultipleProps<string>, "data" | "value" | "onChange">,
): UseSelectMultipleWrapperReturn => {
  const [value, onChange] = useState<string[]>([]);
  const { clear, onSelect } = Select.useMultiple<string>({
    ...props,
    value,
    onChange,
  });
  return { value, clear, onSelect };
};

interface UseSelectSingleWrapperReturn {
  value: string | undefined;
  clear: () => void;
  onSelect: (key: string) => void;
}

const useSelectSingleWrapper = (
  props: Omit<Select.UseSingleProps<string>, "data" | "value" | "onChange">,
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

describe("useSelect", () => {
  describe("multiple selection", () => {
    it("should select two items", () => {
      const { result } = renderHook(useMultipleWrapper);
      act(() => result.current.onSelect("1"));
      expect(result.current.value).toEqual(["1"]);
      act(() => result.current.onSelect("2"));
      expect(result.current.value).toEqual(["1", "2"]);
    });
    it("should deselect an item when you click it again", () => {
      const { result } = renderHook(useMultipleWrapper);
      act(() => result.current.onSelect("1"));
      act(() => result.current.onSelect("2"));
      act(() => result.current.onSelect("1"));
      expect(result.current.value).toEqual(["2"]);
    });
    it("should clear all selections", () => {
      const { result } = renderHook(useMultipleWrapper);
      act(() => result.current.onSelect("1"));
      act(() => result.current.onSelect("2"));
      act(() => result.current.clear());
      expect(result.current.value).toEqual([]);
    });
    describe("no not allow none", () => {
      it("should not allow removing the last selection", () => {
        const { result } = renderHook(() => useMultipleWrapper({}));
        act(() => result.current.onSelect("1"));
        act(() => result.current.onSelect("1"));
        expect(result.current.value).toEqual(["1"]);
      });
    });
    describe("replaceOnSingle", () => {
      it("should replace the selection when you click a new item", () => {
        const { result } = renderHook(() =>
          useMultipleWrapper({ replaceOnSingle: true }),
        );
        act(() => result.current.onSelect("1"));
        act(() => result.current.onSelect("2"));
        expect(result.current.value).toEqual(["2"]);
      });
    });
  });
  describe("single selection", () => {
    it("should select one item", () => {
      const { result } = renderHook(useSelectSingleWrapper);
      act(() => result.current.onSelect("1"));
      expect(result.current.value).toEqual("1");
      act(() => result.current.onSelect("2"));
      expect(result.current.value).toEqual("2");
    });
    it("should deselect an item when you click it again", () => {
      const { result } = renderHook(useSelectSingleWrapper);
      act(() => result.current.onSelect("1"));
      act(() => result.current.onSelect("1"));
      expect(result.current.value).toEqual(null);
    });
    describe("no not allow none", () => {
      it("should not allow clearing all selections", () => {
        const { result } = renderHook(() =>
          useSelectSingleWrapper({ allowNone: false }),
        );
        act(() => result.current.onSelect("1"));
        act(() => result.current.onSelect("1"));
        expect(result.current.value).toEqual("1");
      });
    });
  });
});
