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

import { List } from "@/list";
import { Store } from "@/store";

interface Entry {
  key: string;
  value: number;
}

const A: Entry = { key: "a", value: 1 };
const B: Entry = { key: "b", value: 2 };
const C: Entry = { key: "c", value: 3 };

interface HarnessProps {
  keys: string[];
}

const useHarness = ({ keys }: HarnessProps) => {
  const data = List.useMapData<string, Entry>({ initialData: [A, B, C] });
  const items = Store.useItems({
    keys,
    subscribe: data.subscribe,
    getItem: data.getItem,
  });
  return { data, items };
};

const renderItems = (keys: string[]) =>
  renderHook(useHarness, { initialProps: { keys } });

describe("useItems", () => {
  it("should re-render with an item's new value when that item changes", () => {
    const { result } = renderItems(["a", "b"]);
    expect(result.current.items).toEqual([A, B]);
    const next: Entry = { key: "a", value: 10 };
    act(() => result.current.data.setItem(next));
    expect(result.current.items).toEqual([next, B]);
  });

  it("should keep the same array when a change leaves every item in place", () => {
    const { result } = renderItems(["a", "b"]);
    const first = result.current.items;
    act(() => result.current.data.setItem(A));
    expect(result.current.items).toBe(first);
  });

  it("should follow the items of new keys", () => {
    const { result, rerender } = renderItems(["a"]);
    rerender({ keys: ["c"] });
    expect(result.current.items).toEqual([C]);
    const next: Entry = { key: "c", value: 30 };
    act(() => result.current.data.setItem(next));
    expect(result.current.items).toEqual([next]);
  });
});
