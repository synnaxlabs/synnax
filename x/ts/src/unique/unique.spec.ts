// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { unique } from "@/unique";

describe("unique", () => {
  it("removes duplicate primitive values", () => {
    const result = unique.unique([1, 2, 2, 3, 4, 4, 5]);
    expect(result).toEqual([1, 2, 3, 4, 5]);
  });

  it("works with strings", () => {
    const result = unique.unique(["a", "b", "a", "c", "b"]);
    expect(result).toEqual(["a", "b", "c"]);
  });

  it("works with mixed types", () => {
    const result = unique.unique([1, "1", 2, "2", 1, "1"]);
    expect(result).toEqual([1, "1", 2, "2"]);
  });

  it("handles an empty array", () => {
    const result = unique.unique([]);
    expect(result).toEqual([]);
  });

  it("works with readonly arrays", () => {
    const values: readonly number[] = [1, 1, 2, 3];
    const result = unique.unique(values);
    expect(result).toEqual([1, 2, 3]);
  });
});

describe("by", () => {
  interface IDTestCase {
    id: number;
  }

  it("removes duplicates based on a key function and keeps the first instance by default", () => {
    const result = unique.by(
      [
        { id: 1, name: "A" },
        { id: 2, name: "B" },
        { id: 1, name: "C" },
      ],
      (value: IDTestCase) => value.id,
    );
    expect(result).toEqual([
      { id: 1, name: "A" },
      { id: 2, name: "B" },
    ]);
  });

  it("removes duplicates based on a key function and keeps the first instance when keepFirst is true", () => {
    const result = unique.by(
      [
        { id: 1, name: "A" },
        { id: 2, name: "B" },
        { id: 1, name: "C" },
      ],
      (value: IDTestCase) => value.id,
      true,
    );
    expect(result).toEqual([
      { id: 1, name: "A" },
      { id: 2, name: "B" },
    ]);
  });

  it("removes duplicates based on a key function and keeps the last instance when keepFirst is false", () => {
    const result = unique.by(
      [
        { id: 1, name: "A" },
        { id: 2, name: "B" },
        { id: 1, name: "C" },
      ],
      (value: IDTestCase) => value.id,
      false,
    );
    expect(result).toEqual([
      { id: 2, name: "B" },
      { id: 1, name: "C" },
    ]);
  });

  interface ValueTestCase {
    value: string;
  }

  it("works with a custom key function", () => {
    const result = unique.by(
      [{ value: "apple" }, { value: "banana" }, { value: "apple" }],
      (v: ValueTestCase) => v.value,
    );
    expect(result).toEqual([{ value: "apple" }, { value: "banana" }]);
  });

  it("handles an empty array", () => {
    const result = unique.by([], (v: unknown) => v);
    expect(result).toEqual([]);
  });

  it("works with readonly arrays and keeps the first instance by default", () => {
    const values: readonly { id: number; name: string }[] = [
      { id: 1, name: "A" },
      { id: 1, name: "B" },
    ];
    const result = unique.by(values, (v: IDTestCase) => v.id);
    expect(result).toEqual([{ id: 1, name: "A" }]);
  });

  it("works with readonly arrays and keeps the last instance when keepFirst is false", () => {
    const values: readonly { id: number; name: string }[] = [
      { id: 1, name: "A" },
      { id: 1, name: "B" },
    ];
    const result = unique.by(values, (v: IDTestCase) => v.id, false);
    expect(result).toEqual([{ id: 1, name: "B" }]);
  });

  interface ComplexTestCase {
    id: number;
    nested: { value: string };
  }

  it("works with complex keys and keeps the first instance by default", () => {
    const result = unique.by(
      [
        { id: 1, nested: { value: "A" } },
        { id: 1, nested: { value: "B" } },
        { id: 1, nested: { value: "A", otherKey: "4" } },
      ],
      (v: ComplexTestCase) => `${v.id}-${v.nested.value}`,
    );
    expect(result).toEqual([
      { id: 1, nested: { value: "A" } },
      { id: 1, nested: { value: "B" } },
    ]);
  });

  it("works with complex keys and keeps the last instance when keepFirst is false", () => {
    const result = unique.by(
      [
        { id: 1, nested: { value: "A" } },
        { id: 1, nested: { value: "B" } },
        { id: 1, nested: { value: "A", otherKey: "4" } },
      ],
      (v: ComplexTestCase) => `${v.id}-${v.nested.value}`,
      false,
    );
    expect(result).toEqual([
      { id: 1, nested: { value: "B" } },
      { id: 1, nested: { value: "A", otherKey: "4" } },
    ]);
  });

  it("handles cases where all keys are unique", () => {
    const result = unique.by(
      [
        { id: 1, name: "A" },
        { id: 2, name: "B" },
        { id: 3, name: "C" },
      ],
      (v: IDTestCase) => v.id,
    );
    expect(result).toEqual([
      { id: 1, name: "A" },
      { id: 2, name: "B" },
      { id: 3, name: "C" },
    ]);
  });

  it("handles cases where all values are identical", () => {
    const result = unique.by(
      [
        { id: 1, name: "A" },
        { id: 1, name: "A" },
        { id: 1, name: "A" },
      ],
      (v: IDTestCase) => v.id,
    );
    expect(result).toEqual([{ id: 1, name: "A" }]);
  });
});
