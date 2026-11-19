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

interface TestRecord {
  a: number;
  b: {
    c?: number;
    d?: number;
  };
  c: number[];
}

describe("remove", () => {
  it("should skip null entries after the first in a keyed array", () => {
    const a = { a: [{ key: "x" }, null, { key: "y" }] };
    deep.remove(a, "a.y");
    deep.remove(a, "a.z.w");
    expect(a).toEqual({ a: [{ key: "x" }, null] });
  });

  it("should not throw on a key into an array whose first entry is null", () => {
    const a = { a: [null] };
    deep.remove(a, "a.x");
    deep.remove(a, "a.x.y");
    expect(a).toEqual({ a: [null] });
  });

  it("should delete a key", () => {
    const a: TestRecord = {
      a: 1,
      b: {
        c: 2,
      },
      c: [1, 2, 3],
    };
    const cpy = deep.copy(a);
    deep.remove(a, "b.c");
    expect(a).toEqual({ ...cpy, b: {} });
  });

  it("should delete an array index", () => {
    const a: TestRecord = {
      a: 1,
      b: {
        c: 2,
      },
      c: [1, 2, 3],
    };
    const cpy = deep.copy(a);
    deep.remove(a, "c.1");
    expect(a).toEqual({ ...cpy, c: [1, 3] });
  });

  it("should not throw an error when the index is out of bounds", () => {
    const a: TestRecord = {
      a: 1,
      b: {
        c: 2,
      },
      c: [1, 2, 3],
    };
    const cpy = deep.copy(a);
    deep.remove(a, "c.100");
    expect(a).toEqual(cpy);
  });

  it("should not throw an error when the key doesn't exist", () => {
    const a: TestRecord = {
      a: 1,
      b: {
        c: 2,
      },
      c: [1, 2, 3],
    };
    const cpy = deep.copy(a);
    deep.remove(a, "b.d");
    expect(a).toEqual(cpy);
  });

  it("should not throw an error when recursion depth exceeds the depth of the object", () => {
    const a: TestRecord = {
      a: 1,
      b: {
        c: 2,
      },
      c: [1, 2, 3],
    };
    const cpy = deep.copy(a);
    deep.remove(a, "b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t.u.v.w.x.y.z");
    expect(a).toEqual(cpy);
  });

  it("should handle removing from root level", () => {
    const obj = { a: 1, b: 2, c: 3 };
    deep.remove(obj, "b");
    expect(obj).toEqual({ a: 1, c: 3 });
  });

  it("should handle removing entire nested object", () => {
    const obj = { a: 1, b: { c: 2, d: 3 }, e: 4 };
    deep.remove(obj, "b");
    expect(obj).toEqual({ a: 1, e: 4 });
  });

  it("should handle removing from keyed records in arrays", () => {
    const obj = {
      items: [
        { key: "item1", value: 1 },
        { key: "item2", value: 2 },
      ],
    };
    deep.remove(obj, "items.item1");
    expect(obj.items).toEqual([{ key: "item2", value: 2 }]);
  });

  it("should remove a keyed entry whose key starts with a digit", () => {
    const obj = {
      items: [
        { key: "a", value: 1 },
        { key: "0b", value: 2 },
      ],
    };
    deep.remove(obj, "items.0b");
    expect(obj.items).toEqual([{ key: "a", value: 1 }]);
  });

  it("should handle empty path", () => {
    const obj = { a: 1, b: 2 };
    deep.remove(obj, "");
    expect(obj).toEqual({ a: 1, b: 2 });
  });

  it("should handle removing from nested arrays", () => {
    const obj = {
      a: {
        b: [1, 2, 3],
      },
    };
    deep.remove(obj, "a.b.1");
    expect(obj.a.b).toEqual([1, 3]);
  });

  it("should not affect other properties when removing", () => {
    const obj = {
      a: 1,
      b: {
        c: 2,
        d: 3,
        e: 4,
      },
    };
    deep.remove(obj, "b.d");
    expect(obj).toEqual({
      a: 1,
      b: {
        c: 2,
        e: 4,
      },
    });
  });

  it("should handle removing non-existent intermediate paths", () => {
    const obj = { a: { b: 1 } };
    deep.remove(obj, "c.d.e");
    expect(obj).toEqual({ a: { b: 1 } });
  });

  it("should handle arrays with objects that don't have keys", () => {
    const obj = {
      items: [{ name: "item1" }, { name: "item2" }],
    };
    deep.remove(obj, "items.0");
    expect(obj.items).toEqual([{ name: "item2" }]);
  });

  it("should handle removing with keys containing periods", () => {
    const obj = {
      items: [
        { key: "item.one", value: 1 },
        { key: "item.two", value: 2 },
      ],
    };
    deep.remove(obj, `items.item.one`);
    expect(obj.items).toEqual([{ key: "item.two", value: 2 }]);
  });

  it("should handle removing with keys that contain periods as intermediate keys", () => {
    const obj = {
      items: [
        { key: "item.one", value: { "subitem.one": 3 } },
        { key: "item.two", value: 2 },
      ],
    };
    deep.remove(obj, `items.item.one.value.subitem.one`);
    expect(obj.items).toEqual([
      { key: "item.one", value: {} },
      { key: "item.two", value: 2 },
    ]);
  });

  it("should handle nested array indexes and keys with periods", () => {
    const obj = {
      items: [
        {
          key: "item.one",
          value: [{ key: "subitem.one", value: { "grandchild.one": 3 } }],
        },
        { key: "item.two", value: 2 },
      ],
    };
    deep.remove(obj, `items.item.one.value.subitem.one.value.grandchild.one`);
    expect(obj.items).toEqual([
      { key: "item.one", value: [{ key: "subitem.one", value: {} }] },
      { key: "item.two", value: 2 },
    ]);
  });

  it("should handle nested array indexes and keys with periods", () => {
    const obj = {
      items: [
        {
          key: "item.one",
          value: [{ key: "subitem.one", value: { "grandchild.one": 3 } }],
        },
        { key: "item.two", value: 2 },
      ],
    };
    deep.remove(obj, `items.item.one.value.subitem.one.value.grandchild.one`);
    expect(obj.items).toEqual([
      { key: "item.one", value: [{ key: "subitem.one", value: {} }] },
      { key: "item.two", value: 2 },
    ]);
  });

  it("should remove a key that holds undefined", () => {
    const obj: { a?: number; b: number } = { a: undefined, b: 1 };
    deep.remove(obj, "a");
    expect(Object.keys(obj)).toEqual(["b"]);
  });

  it("should remove a key that holds null", () => {
    const obj: { a: null; b?: number } = { a: null, b: 1 };
    deep.remove(obj, "a");
    expect(obj).toEqual({ b: 1 });
  });

  it("should do nothing for a missing entry of a keyed list", () => {
    const obj = { items: [{ key: "a" }, { key: "b" }] };
    deep.remove(obj, "items.c");
    expect(obj.items).toEqual([{ key: "a" }, { key: "b" }]);
  });

  it("should do nothing for a field of a missing entry", () => {
    const obj = { items: [{ key: "a", value: 1 }] };
    deep.remove(obj, "items.c.value");
    expect(obj.items).toEqual([{ key: "a", value: 1 }]);
  });

  it("should do nothing for a path through null", () => {
    const obj = { a: null };
    deep.remove(obj, "a.b");
    expect(obj).toEqual({ a: null });
  });

  it("should remove the longest key when a shorter key also matches", () => {
    const obj = { a: { b: 1 }, "a.b": 2 };
    deep.remove(obj, "a.b");
    expect(obj).toEqual({ a: { b: 1 } });
  });

  it("should remove by index before key", () => {
    const obj = {
      items: [
        { key: "1", value: "first" },
        { key: "0", value: "second" },
      ],
    };
    deep.remove(obj, "items.1");
    expect(obj.items).toEqual([{ key: "1", value: "first" }]);
  });

  it("should remove the entry with a dot in its key", () => {
    const obj = { items: [{ key: "ns=1;s=a.b" }, { key: "ns=1;s=a" }] };
    deep.remove(obj, "items.ns=1;s=a.b");
    expect(obj.items).toEqual([{ key: "ns=1;s=a" }]);
  });

  it("should not remove array properties such as length", () => {
    const obj = { items: [1, 2] };
    deep.remove(obj, "items.length");
    expect(obj.items).toEqual([1, 2]);
  });
});
