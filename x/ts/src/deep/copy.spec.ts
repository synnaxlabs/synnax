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

describe("copy", () => {
  it("should deep copy a simple object", () => {
    const obj = { a: 1, b: 2 };
    const copied = deep.copy(obj);
    expect(copied).toEqual(obj);
    expect(copied).not.toBe(obj);
  });

  it("should deep copy nested objects", () => {
    const obj = { a: 1, b: { c: 2, d: { e: 3 } } };
    const copied = deep.copy(obj);
    expect(copied).toEqual(obj);
    expect(copied).not.toBe(obj);
    expect(copied.b).not.toBe(obj.b);
    expect(copied.b.d).not.toBe(obj.b.d);
  });

  it("should deep copy arrays", () => {
    const obj = { a: [1, 2, 3], b: [{ c: 4 }, { d: 5 }] };
    const copied = deep.copy(obj);
    expect(copied).toEqual(obj);
    expect(copied.a).not.toBe(obj.a);
    expect(copied.b).not.toBe(obj.b);
    expect(copied.b[0]).not.toBe(obj.b[0]);
  });

  it("should handle null values", () => {
    const obj = { a: null, b: { c: null } };
    const copied = deep.copy(obj);
    expect(copied).toEqual(obj);
    expect(copied).not.toBe(obj);
  });

  it("should handle undefined values", () => {
    const obj = { a: undefined, b: { c: undefined } };
    const copied = deep.copy(obj);
    expect(copied).toEqual(obj);
    expect(copied).not.toBe(obj);
  });

  it("should handle Date objects", () => {
    const date = new Date("2025-01-01");
    const obj = { a: date, b: { c: date } };
    const copied = deep.copy(obj);
    expect(copied.a).toEqual(date);
    expect(copied.a).not.toBe(date);
    expect(copied.b.c).toEqual(date);
    expect(copied.b.c).not.toBe(date);
  });

  it("should handle circular references with structuredClone", () => {
    const obj: any = { a: 1 };
    obj.circular = obj;

    const copied = deep.copy(obj);

    expect(copied.a).toBe(1);
    expect(copied.circular).toBe(copied);
    expect(copied).not.toBe(obj);
  });

  it("should handle Map objects", () => {
    const map = new Map([
      ["a", 1],
      ["b", 2],
    ]);
    const obj = { map };
    const copied = deep.copy(obj);
    expect(copied.map).toEqual(map);
    expect(copied.map).not.toBe(map);
  });

  it("should handle Set objects", () => {
    const set = new Set([1, 2, 3]);
    const obj = { set };
    const copied = deep.copy(obj);
    expect(copied.set).toEqual(set);
    expect(copied.set).not.toBe(set);
  });

  it("should handle RegExp objects", () => {
    const regex = /test/gi;
    const obj = { regex };
    const copied = deep.copy(obj);
    expect(copied.regex).toEqual(regex);
    expect(copied.regex).not.toBe(regex);
  });

  it("should handle mixed types", () => {
    const obj = {
      num: 42,
      str: "test",
      bool: true,
      arr: [1, 2, { nested: "value" }],
      obj: { a: 1, b: { c: 2 } },
      date: new Date("2025-01-01"),
      nil: null,
      undef: undefined,
    };
    const copied = deep.copy(obj);
    expect(copied).toEqual(obj);
    expect(copied).not.toBe(obj);
    expect(copied.arr).not.toBe(obj.arr);
    expect(copied.obj).not.toBe(obj.obj);
    expect(copied.date).not.toBe(obj.date);
  });

  it("should handle empty objects and arrays", () => {
    const obj = { empty: {}, arr: [] };
    const copied = deep.copy(obj);
    expect(copied).toEqual(obj);
    expect(copied.empty).not.toBe(obj.empty);
    expect(copied.arr).not.toBe(obj.arr);
  });

  it("should preserve array length with sparse arrays", () => {
    // eslint-disable-next-line no-sparse-arrays
    const arr = [1, , , 4];
    const copied = deep.copy(arr);
    expect(copied).toEqual(arr);
    expect(copied.length).toBe(4);
    expect(copied[1]).toBeUndefined();
  });

  it("should handle Uint8Array and other typed arrays", () => {
    const uint8 = new Uint8Array([1, 2, 3]);
    const obj = { uint8 };
    const copied = deep.copy(obj);
    expect(copied.uint8).toBeDefined();
    if (copied.uint8 instanceof Uint8Array) {
      expect(Array.from(copied.uint8)).toEqual([1, 2, 3]);
      expect(copied.uint8).not.toBe(uint8);
    } else expect(Array.from(copied.uint8 as Uint8Array)).toEqual([1, 2, 3]);
  });
});
