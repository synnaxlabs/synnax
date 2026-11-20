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
import type { record } from "@/record";

interface TestRecord {
  a: number;
  b: {
    c?: number;
    d?: number;
  };
  c: number[];
}

describe("get", () => {
  it("should get a key", () => {
    const a: TestRecord = {
      a: 1,
      b: {
        c: 2,
      },
      c: [1],
    };
    expect(deep.get(a, "b.c")).toEqual(2);
  });

  it("should get an array index", () => {
    const a: TestRecord = {
      a: 1,
      b: {
        c: 2,
      },
      c: [1, 2, 3],
    };
    expect(deep.get(a, "c.1")).toEqual(2);
  });

  it("should return the object itself if the key is empty", () => {
    const a: TestRecord = {
      a: 1,
      b: {
        c: 2,
      },
      c: [1, 2, 3],
    };
    expect(deep.get(a, "")).toBe(a);
  });

  it("should get an array of keyed records", () => {
    interface TestKeyedRecord {
      values: record.KeyedNamed[];
    }
    const a: TestKeyedRecord = {
      values: [
        { key: "a", name: "a" },
        { key: "b", name: "b" },
      ],
    };
    expect(deep.get(a, "values.a.name")).toEqual("a");
  });

  describe("path includes a .", () => {
    it("should get a value in case the path includes a period", () => {
      const data = { "a.b": { c: 1 } };
      expect(deep.get(data, "a.b.c")).toEqual(1);
    });

    it("should work with keys that contain a period", () => {
      const data = {
        channels: [
          { key: "key.period", color: "blue" },
          { key: "noPeriod", color: "red" },
        ],
      };
      expect(deep.get(data, "channels.key.period.color")).toEqual("blue");
      expect(deep.get(data, "channels.noPeriod.color")).toEqual("red");
    });
  });

  it("should return null for a key that holds null", () => {
    expect(deep.get({ a: null }, "a")).toBeNull();
  });

  it("should return null for a path through null", () => {
    expect(deep.get({ a: null }, "a.b")).toBeNull();
  });

  it("should return null for a key that holds undefined", () => {
    const obj = { a: undefined };
    expect(deep.get(obj, "a")).toBeNull();
    expect(deep.get(obj, "a.b")).toBeNull();
  });

  it("should skip null entries after the first in a keyed array", () => {
    const obj = { a: [{ key: "x", v: 1 }, null] };
    expect(deep.get(obj, "a.y")).toBeNull();
  });

  it("should return null for a key into an array whose first entry is null", () => {
    expect(deep.get({ a: [null] }, "a.x")).toBeNull();
  });

  it("should return null for a missing key", () => {
    expect(deep.get({ a: { b: 1 } }, "a.c")).toBeNull();
  });

  it("should return null for a missing middle key", () => {
    expect(deep.get({ a: { b: 1 } }, "a.c.d")).toBeNull();
  });

  it("should return null for a missing entry of a keyed list", () => {
    const obj = { items: [{ key: "a", value: 1 }] };
    expect(deep.get(obj, "items.b")).toBeNull();
    expect(deep.get(obj, "items.b.value")).toBeNull();
  });

  it("should return null for an index past the end of a list", () => {
    expect(deep.get({ items: [1, 2] }, "items.2")).toBeNull();
  });

  it("should return falsy values that are not null", () => {
    const obj = { zero: 0, empty: "", no: false };
    expect(deep.get(obj, "zero")).toBe(0);
    expect(deep.get(obj, "empty")).toBe("");
    expect(deep.get(obj, "no")).toBe(false);
  });

  it("should return the value itself, not a copy", () => {
    const obj = { a: { b: { c: 1 } } };
    expect(deep.get(obj, "a.b")).toBe(obj.a.b);
  });

  it("should prefer the longest key when a shorter key also matches", () => {
    const obj = { a: { b: { c: 1 } }, "a.b": { c: 2 } };
    expect(deep.get(obj, "a.b.c")).toBe(2);
  });

  it("should handle numeric string keys", () => {
    const obj = { "123": "numeric key" };
    expect(deep.get(obj, "123")).toEqual("numeric key");
  });

  it("should handle special characters in keys", () => {
    const obj = { "key-with-dash": 1, key_with_underscore: 2 };
    expect(deep.get(obj, "key-with-dash")).toEqual(1);
    expect(deep.get(obj, "key_with_underscore")).toEqual(2);
  });

  it("should handle boolean values in paths", () => {
    const obj = { a: { true: "yes", false: "no" } };
    expect(deep.get(obj, "a.true")).toEqual("yes");
    expect(deep.get(obj, "a.false")).toEqual("no");
  });

  it("should handle very long paths", () => {
    const longPath = new Array(100).fill("a").join(".");
    const obj: any = {};
    let current = obj;
    for (let i = 0; i < 99; i++) {
      current.a = {};
      current = current.a;
    }
    current.a = "deep";
    expect(deep.get(obj, longPath)).toEqual("deep");
  });

  it("should not throw for a path the object does not hold", () => {
    expect(() => deep.get({ a: { b: 1 } }, "a.c.d")).not.toThrow();
  });
});

describe("has", () => {
  it("should return true if path exists", () => {
    const obj = { a: { b: { c: 1 } } };
    expect(deep.has(obj, "a.b.c")).toBe(true);
  });

  it("should return false if path doesn't exist", () => {
    const obj = { a: { b: 1 } };
    expect(deep.has(obj, "a.c")).toBe(false);
  });

  it("should handle arrays", () => {
    const obj = { a: [1, 2, 3] };
    expect(deep.has(obj, "a.1")).toBe(true);
    expect(deep.has(obj, "a.5")).toBe(false);
  });

  it("should handle keyed records in arrays", () => {
    const obj = {
      items: [
        { key: "item1", value: 1 },
        { key: "item2", value: 2 },
      ],
    };
    expect(deep.has(obj, "items.item1")).toBe(true);
    expect(deep.has(obj, "items.item3")).toBe(false);
  });

  it("should return false for a key that holds null", () => {
    expect(deep.has({ a: null }, "a")).toBe(false);
  });

  it("should return true for falsy values that are not null", () => {
    expect(deep.has({ a: 0 }, "a")).toBe(true);
    expect(deep.has({ a: false }, "a")).toBe(true);
  });

  it("should return true for the empty path", () => {
    expect(deep.has({}, "")).toBe(true);
  });
});
