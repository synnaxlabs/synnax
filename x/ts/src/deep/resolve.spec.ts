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

describe("resolve", () => {
  describe("objects", () => {
    it("should resolve a key on the root", () => {
      const obj = { a: 1 };
      expect(deep.resolve(obj, "a")).toEqual({ parent: obj, key: "a", value: 1 });
    });

    it("should resolve a nested key to the object that holds it", () => {
      const obj = { a: { b: { c: 1 } } };
      const found = deep.resolve(obj, "a.b.c");
      expect(found?.parent).toBe(obj.a.b);
      expect(found?.key).toBe("c");
      expect(found?.value).toBe(1);
    });

    it("should resolve to the object itself, not a copy", () => {
      const obj = { a: { b: { c: 1 } } };
      expect(deep.resolve(obj, "a.b")?.value).toBe(obj.a.b);
    });

    it("should resolve a key that holds null", () => {
      const obj = { a: { b: null } };
      expect(deep.resolve(obj, "a.b")).toEqual({
        parent: obj.a,
        key: "b",
        value: null,
      });
    });

    it("should resolve a key that holds undefined", () => {
      const obj: { a?: number } = { a: undefined };
      expect(deep.resolve(obj, "a")).toEqual({
        parent: obj,
        key: "a",
        value: undefined,
      });
    });

    it("should resolve a key on a function", () => {
      const fn = Object.assign(() => {}, { nested: { value: 1 } });
      expect(deep.resolve(fn, "nested.value")?.value).toBe(1);
    });

    it("should resolve empty parts as empty keys", () => {
      const obj = { a: { "": { b: 1 } } };
      expect(deep.resolve(obj, "a..b")?.value).toBe(1);
    });
  });

  describe("misses", () => {
    it("should return null for a missing key", () => {
      expect(deep.resolve({ a: 1 }, "b")).toBeNull();
    });

    it("should return null for a missing middle key", () => {
      expect(deep.resolve({ a: { b: 1 } }, "a.c.d")).toBeNull();
    });

    it("should return null when null sits in the middle of the path", () => {
      expect(deep.resolve({ a: null }, "a.b")).toBeNull();
    });

    it("should return null when a primitive sits in the middle of the path", () => {
      expect(deep.resolve({ a: "text" }, "a.length")).toBeNull();
    });

    it("should return null for a root that holds nothing", () => {
      expect(deep.resolve(null, "a")).toBeNull();
      expect(deep.resolve(undefined, "a")).toBeNull();
      expect(deep.resolve(5, "a")).toBeNull();
    });

    it("should return null for a path deeper than the object", () => {
      expect(deep.resolve({ a: { b: 1 } }, "a.b.c.d.e")).toBeNull();
    });
  });

  describe("empty path", () => {
    it("should return null when the object has no empty key", () => {
      expect(deep.resolve({ a: 1 }, "")).toBeNull();
    });

    it("should resolve the empty key", () => {
      const obj = { "": 1 };
      expect(deep.resolve(obj, "")).toEqual({ parent: obj, key: "", value: 1 });
    });
  });

  describe("arrays", () => {
    it("should resolve an index to a numeric key", () => {
      const obj = { list: [10, 20, 30] };
      expect(deep.resolve(obj, "list.1")).toEqual({
        parent: obj.list,
        key: 1,
        value: 20,
      });
    });

    it("should return null for an index past the end", () => {
      expect(deep.resolve({ list: [1, 2] }, "list.2")).toBeNull();
    });

    it("should return null for a hole in a sparse array", () => {
      // eslint-disable-next-line no-sparse-arrays
      expect(deep.resolve({ list: [1, , 3] }, "list.1")).toBeNull();
    });

    it("should resolve an index that holds null", () => {
      const obj = { list: [null] };
      expect(deep.resolve(obj, "list.0")).toEqual({
        parent: obj.list,
        key: 0,
        value: null,
      });
    });

    it("should not read negative indexes", () => {
      expect(deep.resolve({ list: [1, 2] }, "list.-1")).toBeNull();
    });

    it("should not read array properties such as length", () => {
      expect(deep.resolve({ list: [1, 2] }, "list.length")).toBeNull();
    });
  });

  describe("keyed lists", () => {
    const createList = () => ({
      list: [
        { key: "a", value: 1 },
        { key: "b", value: 2 },
      ],
    });

    it("should resolve an entry by its key to its index", () => {
      const obj = createList();
      expect(deep.resolve(obj, "list.b")).toEqual({
        parent: obj.list,
        key: 1,
        value: obj.list[1],
      });
    });

    it("should resolve a field of an entry", () => {
      const obj = createList();
      expect(deep.resolve(obj, "list.b.value")).toEqual({
        parent: obj.list[1],
        key: "value",
        value: 2,
      });
    });

    it("should return null for a missing entry", () => {
      expect(deep.resolve(createList(), "list.c")).toBeNull();
    });

    it("should return null for a field of a missing entry", () => {
      expect(deep.resolve(createList(), "list.c.value")).toBeNull();
    });

    it("should return null for a key on an empty list", () => {
      expect(deep.resolve({ list: [] }, "list.a")).toBeNull();
    });

    it("should resolve the first of entries that share a key", () => {
      const obj = {
        list: [
          { key: "a", value: 1 },
          { key: "a", value: 2 },
        ],
      };
      expect(deep.resolve(obj, "list.a.value")?.value).toBe(1);
    });

    it("should resolve a key that starts with a digit", () => {
      const obj = { list: [{ key: "1b", value: 1 }] };
      expect(deep.resolve(obj, "list.1b.value")?.value).toBe(1);
    });

    it("should read an index before a key", () => {
      const obj = {
        list: [
          { key: "1", value: "first" },
          { key: "0", value: "second" },
        ],
      };
      expect(deep.resolve(obj, "list.1.value")?.value).toBe("second");
    });

    it("should read a key that looks like an index past the end", () => {
      const obj = { list: [{ key: "5", value: "five" }] };
      expect(deep.resolve(obj, "list.5.value")?.value).toBe("five");
    });

    it("should find an entry among values that are not entries", () => {
      const obj = { list: [1, null, { key: "a", value: 3 }] };
      expect(deep.resolve(obj, "list.a.value")?.value).toBe(3);
    });

    it("should not match an entry whose key is a number", () => {
      expect(deep.resolve({ list: [{ key: 7 }] }, "list.7")).toBeNull();
    });
  });

  describe("keys that hold the separator", () => {
    it("should resolve a key with a dot", () => {
      const obj = { "a.b": { c: 1 } };
      expect(deep.resolve(obj, "a.b.c")).toEqual({
        parent: obj["a.b"],
        key: "c",
        value: 1,
      });
    });

    it("should prefer the longest key at the leaf", () => {
      const obj = { a: { b: 1 }, "a.b": 2 };
      expect(deep.resolve(obj, "a.b")).toEqual({ parent: obj, key: "a.b", value: 2 });
    });

    it("should prefer the longest key in the middle", () => {
      const obj = { a: { b: { c: 1 } }, "a.b": { c: 2 } };
      expect(deep.resolve(obj, "a.b.c")?.value).toBe(2);
    });

    it("should fall back to a shorter key when the longer one is missing", () => {
      const obj = { a: { b: { c: 1 } } };
      expect(deep.resolve(obj, "a.b.c")?.value).toBe(1);
    });

    it("should not go back to a shorter key after a longer one leads nowhere", () => {
      const obj = { a: { b: { c: 1 } }, "a.b": null };
      expect(deep.resolve(obj, "a.b.c")).toBeNull();
    });

    it("should resolve an entry key with dots", () => {
      const obj = { list: [{ key: "ns=1;s=a.b", port: 1 }] };
      expect(deep.resolve(obj, "list.ns=1;s=a.b.port")).toEqual({
        parent: obj.list[0],
        key: "port",
        value: 1,
      });
    });

    it("should resolve entry keys with dots in nested lists", () => {
      const obj = {
        items: [
          {
            key: "item.one",
            value: [{ key: "sub.one", value: { "leaf.one": 3 } }],
          },
        ],
      };
      expect(
        deep.resolve(obj, "items.item.one.value.sub.one.value.leaf.one")?.value,
      ).toBe(3);
    });

    it("should resolve keys over 100 parts deep", () => {
      const obj: Record<string, unknown> = {};
      let current = obj;
      for (let i = 0; i < 99; i++) {
        const next = {};
        current.a = next;
        current = next;
      }
      current.a = "deep";
      expect(deep.resolve(obj, new Array(100).fill("a").join("."))?.value).toBe("deep");
    });
  });
});

describe("resolvePath", () => {
  it("should leave a path through objects as it is", () => {
    expect(deep.resolvePath("a.b.c", { a: { b: { c: 1 } } })).toEqual("a.b.c");
  });

  it("should leave an index into a list without keys as it is", () => {
    expect(deep.resolvePath("a.b.0.c", { a: { b: [{ c: 1 }] } })).toEqual("a.b.0.c");
  });

  it("should rewrite an index into the key of the entry", () => {
    expect(deep.resolvePath("a.b.0.d", { a: { b: [{ key: "c", d: 1 }] } })).toEqual(
      "a.b.c.d",
    );
  });

  it("should leave a path that names an entry by key as it is", () => {
    expect(deep.resolvePath("a.b.c.d", { a: { b: [{ key: "c", d: 1 }] } })).toEqual(
      "a.b.c.d",
    );
  });

  it("should rewrite only the parts inside arrays", () => {
    expect(deep.resolvePath("a.b.c", { a: { b: { key: "", c: 1 } } })).toEqual("a.b.c");
  });

  it("should rewrite an index into an entry key with dots", () => {
    expect(
      deep.resolvePath("list.0.port", { list: [{ key: "a.b", port: 1 }] }),
    ).toEqual("list.a.b.port");
  });

  it("should rewrite indexes in nested lists", () => {
    const obj = { a: [{ key: "x", b: [{ key: "y", c: 1 }] }] };
    expect(deep.resolvePath("a.0.b.0.c", obj)).toEqual("a.x.b.y.c");
  });

  it("should keep the parts past where the path leaves the object", () => {
    const obj = { a: [{ key: "x" }] };
    expect(deep.resolvePath("a.0.missing.deeper", obj)).toEqual("a.x.missing.deeper");
  });
});
