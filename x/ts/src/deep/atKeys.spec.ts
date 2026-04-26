// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { deep } from "@/deep";

describe("deep.atKeys", () => {
  it("should return the value at a single-segment path", () => {
    expect(deep.atKeys({ a: 1 }, ["a"])).toEqual({ present: true, value: 1 });
  });

  it("should return the value at a deeply nested path", () => {
    expect(deep.atKeys({ a: { b: { c: "deep" } } }, ["a", "b", "c"])).toEqual({
      present: true,
      value: "deep",
    });
  });

  it("should walk into arrays by numeric index", () => {
    expect(deep.atKeys({ items: [10, 20, 30] }, ["items", 1])).toEqual({
      present: true,
      value: 20,
    });
  });

  it("should return the root itself for an empty path", () => {
    const root = { a: 1 };
    expect(deep.atKeys(root, [])).toEqual({ present: true, value: root });
  });

  it("should mark a missing top-level key as not present", () => {
    expect(deep.atKeys({ a: 1 }, ["b"])).toEqual({
      present: false,
      value: undefined,
    });
  });

  it("should mark a missing nested key as not present", () => {
    expect(deep.atKeys({ a: { b: 1 } }, ["a", "c"])).toEqual({
      present: false,
      value: undefined,
    });
  });

  it("should return the value when a key is present with explicit undefined", () => {
    expect(deep.atKeys({ a: undefined }, ["a"])).toEqual({
      present: true,
      value: undefined,
    });
  });

  it("should return the value when a key is present with explicit null", () => {
    expect(deep.atKeys({ a: null }, ["a"])).toEqual({
      present: true,
      value: null,
    });
  });

  it("should distinguish present-but-null from missing for nested paths", () => {
    expect(deep.atKeys({ a: { b: null } }, ["a", "b"])).toEqual({
      present: true,
      value: null,
    });
  });

  it("should treat walking into a null value as not present", () => {
    expect(deep.atKeys({ a: null }, ["a", "b"])).toEqual({
      present: false,
      value: undefined,
    });
  });

  it("should treat walking into a primitive as not present", () => {
    expect(deep.atKeys({ a: 5 }, ["a", "b"])).toEqual({
      present: false,
      value: undefined,
    });
  });

  it("should not apply the deep.get 'find by .key' heuristic to arrays", () => {
    const obj = { items: [{ key: "foo", value: 1 }] };
    // deep.get would resolve ["items", "foo"] to the keyed item; atKeys does not.
    expect(deep.atKeys(obj, ["items", "foo"])).toEqual({
      present: false,
      value: undefined,
    });
  });

  it("should handle keys that contain literal dots unambiguously", () => {
    const obj = { "foo.bar": 42 };
    expect(deep.atKeys(obj, ["foo.bar"])).toEqual({ present: true, value: 42 });
  });

  it("should return not-present when root itself is null", () => {
    expect(deep.atKeys(null, ["a"])).toEqual({
      present: false,
      value: undefined,
    });
  });
});
