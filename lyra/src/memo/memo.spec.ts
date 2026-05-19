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

import { Memo } from "@/memo";

const setup = <T>(initial: T[]) =>
  renderHook(({ value }) => Memo.useArray(value), {
    initialProps: { value: initial },
  });

describe("Memo.useArray", () => {
  it("returns the initial array on first render", () => {
    const initial = [1, 2, 3];
    const { result } = setup<number>(initial);
    expect(result.current).toBe(initial);
  });

  it("keeps the same reference when contents are element-wise equal", () => {
    const { result, rerender } = setup<number>([1, 2, 3]);
    const first = result.current;
    rerender({ value: [1, 2, 3] });
    expect(result.current).toBe(first);
  });

  it("returns the new reference when an element changes", () => {
    const { result, rerender } = setup<number>([1, 2, 3]);
    const next = [1, 2, 4];
    rerender({ value: next });
    expect(result.current).toBe(next);
  });

  it("returns the new reference when the length changes", () => {
    const { result, rerender } = setup<number>([1, 2, 3]);
    const next = [1, 2, 3, 4];
    rerender({ value: next });
    expect(result.current).toBe(next);
  });

  it("treats empty arrays as a stable reference", () => {
    const { result, rerender } = setup<number>([]);
    const first = result.current;
    rerender({ value: [] });
    expect(result.current).toBe(first);
  });

  it("compares element-wise across primitive types", () => {
    const { result, rerender } = setup<string>(["a", "b"]);
    const first = result.current;
    rerender({ value: ["a", "b"] });
    expect(result.current).toBe(first);
    const next = ["a", "c"];
    rerender({ value: next });
    expect(result.current).toBe(next);
  });

  it("reflects in-place mutation of the same array reference", () => {
    // Sanity: the hook compares contents, not identity. A caller that
    // mutates the previously-returned array still sees the new contents.
    const arr = [1, 2, 3];
    const { result, rerender } = setup<number>(arr);
    arr.push(4);
    rerender({ value: arr });
    expect(result.current).toEqual([1, 2, 3, 4]);
  });

  it("treats NaN as equal to NaN (Object.is semantics)", () => {
    const { result, rerender } = setup<number>([NaN, 1]);
    const first = result.current;
    rerender({ value: [NaN, 1] });
    expect(result.current).toBe(first);
  });

  it("compares object references by identity", () => {
    const a = { id: 1 };
    const b = { id: 2 };
    const { result, rerender } = setup<{ id: number }>([a, b]);
    const first = result.current;
    rerender({ value: [a, b] });
    expect(result.current).toBe(first);
    rerender({ value: [a, { id: 2 }] });
    expect(result.current).not.toBe(first);
  });
});
