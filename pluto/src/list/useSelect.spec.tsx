// Copyright 2024 Synnax Labs, Inc.
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

import { useSelect, type UseSelectMultipleProps } from "@/list/useSelect";

interface Entry {
  key: string;
  name: string;
}

const data: Entry[] = [
  {
    key: "1",
    name: "John",
  },
  {
    key: "2",
    name: "James",
  },
  {
    key: "3",
    name: "Javier",
  },
];

interface UseSelectMultipleWrapperReturn {
  value: string[];
  clear: () => void;
  onSelect: (key: string) => void;
}

const useSelectMultipleWrapper = (
  props: Omit<UseSelectMultipleProps<string, Entry>, "data" | "value" | "onChange">,
): UseSelectMultipleWrapperReturn => {
  const [value, setValue] = useState<string[]>([]);
  const { clear, onSelect } = useSelect<string, Entry>({
    ...props,
    data,
    value,
    onChange: setValue,
  });
  return { value, clear, onSelect };
};

interface UseSelectSingleWrapperReturn {
  value: string | null;
  clear: () => void;
  onSelect: (key: string) => void;
}

const useSelectSingleWrapper = (
  props: Omit<UseSelectMultipleProps<string, Entry>, "data" | "value" | "onChange">,
): UseSelectSingleWrapperReturn => {
  const [value, setValue] = useState<string | null>(null);
  // a different wrapper for the allowNone: false case
  const { clear, onSelect } = useSelect<string, Entry>({
    allowNone: true,
    ...props,
    data,
    value,
    onChange: setValue,
    allowMultiple: false,
  });
  return { value, clear, onSelect };
};

describe("useSelect", () => {
  describe("multiple selection", () => {
    it("should select two items", () => {
      const { result } = renderHook(useSelectMultipleWrapper);
      act(() => result.current.onSelect("1"));
      expect(result.current.value).toEqual(["1"]);
      act(() => result.current.onSelect("2"));
      expect(result.current.value).toEqual(["1", "2"]);
    });
    it("should deselect an item when you click it again", () => {
      const { result } = renderHook(useSelectMultipleWrapper);
      act(() => result.current.onSelect("1"));
      act(() => result.current.onSelect("2"));
      act(() => result.current.onSelect("1"));
      expect(result.current.value).toEqual(["2"]);
    });
    it("should clear all selections", () => {
      const { result } = renderHook(useSelectMultipleWrapper);
      act(() => result.current.onSelect("1"));
      act(() => result.current.onSelect("2"));
      act(() => result.current.clear());
      expect(result.current.value).toEqual([]);
    });
    describe("no not allow none", () => {
      it("should not allow removing the last selection", () => {
        const { result } = renderHook(() =>
          useSelectMultipleWrapper({ allowNone: false }),
        );
        act(() => result.current.onSelect("1"));
        act(() => result.current.onSelect("1"));
        expect(result.current.value).toEqual(["1"]);
      });
      it("should automatically populate the first item", () => {
        const { result } = renderHook(() =>
          useSelectMultipleWrapper({ allowNone: false, autoSelectOnNone: true }),
        );
        expect(result.current.value).toEqual(["1"]);
      });
    });
    describe("replaceOnSingle", () => {
      it("should replace the selection when you click a new item", () => {
        const { result } = renderHook(() =>
          useSelectMultipleWrapper({ replaceOnSingle: true }),
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
      it("should automatically populate the first item", () => {
        const { result } = renderHook(() =>
          useSelectSingleWrapper({ allowNone: false, autoSelectOnNone: true }),
        );
        expect(result.current.value).toEqual("1");
      });
    });
  });
});
