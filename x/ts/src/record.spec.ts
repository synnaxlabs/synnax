// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { mapValues } from "@/record";

describe("mapValues", () => {
  it("should map values of a record using the provided function", () => {
    const input = { a: 1, b: 2, c: 3 };
    const result = mapValues(input, (x) => x * 2);
    expect(result).toEqual({ a: 2, b: 4, c: 6 });
  });

  it("should handle empty objects", () => {
    const input = {};
    const result = mapValues(input, (x) => x * 2);
    expect(result).toEqual({});
  });

  it("should handle different value types", () => {
    const input = { name: "John", age: 30 };
    const result = mapValues(input, (value) => String(value));
    expect(result).toEqual({ name: "John", age: "30" });
  });

  it("should preserve keys while transforming values", () => {
    const input = { x: "hello", y: "world" };
    const result = mapValues(input, (str) => str.toUpperCase());
    expect(result).toEqual({ x: "HELLO", y: "WORLD" });
  });
});
