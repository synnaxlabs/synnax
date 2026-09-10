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

describe("freeze", () => {
  it("should freeze a simple object", () => {
    const obj = deep.freeze({ a: 1 });
    expect(Object.isFrozen(obj)).toBe(true);
  });

  it("should freeze nested objects", () => {
    const obj = deep.freeze({ a: { b: { c: 1 } } });
    expect(Object.isFrozen(obj.a)).toBe(true);
    expect(Object.isFrozen(obj.a.b)).toBe(true);
  });

  it("should freeze objects inside arrays", () => {
    const obj = deep.freeze({ a: [{ b: 1 }] });
    expect(Object.isFrozen(obj.a)).toBe(true);
    expect(Object.isFrozen(obj.a[0])).toBe(true);
  });

  it("should return the same reference", () => {
    const obj = { a: 1 };
    expect(deep.freeze(obj)).toBe(obj);
  });

  it("should return primitives and null unchanged", () => {
    expect(deep.freeze(1)).toBe(1);
    expect(deep.freeze("a")).toBe("a");
    expect(deep.freeze(null)).toBeNull();
    expect(deep.freeze(undefined)).toBeUndefined();
  });

  it("should handle a cyclic reference", () => {
    const obj: Record<string, unknown> = { a: 1 };
    obj.self = obj;
    expect(Object.isFrozen(deep.freeze(obj))).toBe(true);
  });
});
