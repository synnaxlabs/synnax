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

interface TestRecord {
  a: number;
  b: {
    c?: number;
    d?: number;
  };
  c: number[];
}

describe("remove", () => {
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
});
