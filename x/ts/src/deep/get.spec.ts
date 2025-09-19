// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { deep } from "@/deep";
import { type record } from "@/record";

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
    expect(deep.get(a, "")).toStrictEqual(a);
  });

  describe("custom getter function", () => {
    const v = {
      a: {
        value: () => ({
          c: 0,
        }),
      },
    };
    it("should use the custom getter function", () => {
      expect(
        deep.get(v, "a.value().c", {
          optional: false,
          getter: (obj, key) => {
            if (key === "value()")
              return (obj as { value: () => { c: number } }).value();
            return obj[key];
          },
        }),
      ).toEqual(0);
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

  it("should handle null values in get", () => {
    const obj = { a: null };
    expect(deep.get(obj, "a")).toBeNull();
    expect(deep.get(obj, "a.b", { optional: true })).toBeNull();
  });

  it("should handle undefined values in get", () => {
    const obj = { a: undefined };
    expect(deep.get(obj, "a", { optional: true })).toBeNull();
    expect(deep.get(obj, "a.b", { optional: true })).toBeNull();
  });

  it("should handle get with default value", () => {
    const obj = { a: { b: 1 } };
    const result = deep.get(obj, "a.c", { optional: true });
    expect(result).toBeNull();
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

  it("should throw error when path doesn't exist and not optional", () => {
    const obj = { a: { b: 1 } };
    expect(() => deep.get(obj, "a.c.d")).toThrow();
  });

  it("should return null when path doesn't exist and optional", () => {
    const obj = { a: { b: 1 } };
    expect(deep.get(obj, "a.c.d", { optional: true })).toBeNull();
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
});
