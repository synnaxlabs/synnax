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

describe("set", () => {
  it("should set a key", () => {
    const a: TestRecord = {
      a: 1,
      b: {
        c: 2,
      },
      c: [1],
    };
    const b: TestRecord = {
      a: 1,
      b: {
        c: 3,
      },
      c: [1],
    };
    deep.set(a, "b.c", 3);
    expect(a).toEqual(b);
  });

  it("should set an array index", () => {
    const a: TestRecord = {
      a: 1,
      b: {
        c: 2,
      },
      c: [1, 2, 3],
    };
    const b: TestRecord = {
      a: 1,
      b: {
        c: 2,
      },
      c: [1, 4, 3],
    };
    deep.set(a, "c.1", 4);
    expect(a).toEqual(b);
  });

  it("should interpret a leading number also containing letters as a key", () => {
    const data = {
      a: [
        {
          key: "1b",
          value: 1,
        },
      ],
    };
    deep.set(data, "a.1b.value", 2);
    expect(deep.get(data, "a.1b.value")).toEqual(2);
  });

  it("should set a value on a nested object in the array by key", () => {
    const data = {
      config: {
        channels: [{ key: "tMnAnJeQmn6", type: "ai_voltage" }],
      },
    };
    deep.set(data, "config.channels.tMnAnJeQmn6.type", "ai_force_bridge_table");
    expect(data.config.channels[0].type).toEqual("ai_force_bridge_table");
  });

  it("should set an entire item in the array by its key", () => {
    const data = {
      config: {
        channels: [{ key: "tMnAnJeQmn6", type: "ai_voltage" }],
      },
    };
    deep.set(data, "config.channels.tMnAnJeQmn6", {
      key: "tMnAnJeQmn6",
      type: "ai_force_bridge_table",
    });
    expect(data.config.channels[0]).toEqual({
      key: "tMnAnJeQmn6",
      type: "ai_force_bridge_table",
    });
  });

  it("should prefer setting an existing value if a key contains a period instead of creating a new object", () => {
    const data = { "a.b": { c: 1 } };
    deep.set(data, "a.b.d", 2);
    expect(data).toEqual({ "a.b": { c: 1, d: 2 } });
  });

  it("should write the value get reads when a shorter key also matches", () => {
    const data = { a: { b: { c: 1 } }, "a.b": { c: 2 } };
    deep.set(data, "a.b.c", 3);
    expect(deep.get(data, "a.b.c")).toBe(3);
    expect(data.a.b.c).toBe(1);
  });

  it("should not write through the object prototype", () => {
    const obj: Record<string, unknown> = {};
    deep.set(obj, "__proto__.polluted", 1);
    expect(({} as Record<string, unknown>).polluted).toBeUndefined();
  });

  it("should handle setting on null", () => {
    const obj: any = { a: null };
    deep.set(obj, "a.b", 1);
    expect(obj.a.b).toEqual(1);
  });

  it("should create nested path when intermediate values don't exist", () => {
    const obj: any = {};
    deep.set(obj, "a.b.c.d", "value");
    expect(obj.a.b.c.d).toEqual("value");
  });

  it("should throw for a negative index", () => {
    const obj = { arr: [1, 2, 3, 4, 5] };
    expect(() => deep.set(obj, "arr.-1", 99)).toThrow(
      "Cannot set arr.-1: the list holds no entry with the key -1",
    );
    expect(obj.arr).toEqual([1, 2, 3, 4, 5]);
  });

  it("should handle very long paths", () => {
    const longPath = new Array(100).fill("a").join(".");
    const obj: any = {};
    deep.set(obj, longPath, "deep");
    expect(deep.get(obj, longPath)).toEqual("deep");
  });

  it("should create arrays when next part is numeric", () => {
    const obj: any = {};
    deep.set(obj, "a.0", "first");
    expect(Array.isArray(obj.a)).toBe(true);
    expect(obj.a[0]).toEqual("first");
  });

  it("should create objects when next part is non-numeric", () => {
    const obj: any = {};
    deep.set(obj, "a.b", "value");
    expect(Array.isArray(obj.a)).toBe(false);
    expect(obj.a.b).toEqual("value");
  });

  it("should handle empty arrays", () => {
    const obj = { items: [] };
    deep.set(obj, "items.0", "first");
    expect(obj.items).toEqual(["first"]);
  });

  it("should handle setting values on root", () => {
    const obj = { a: 1 };
    deep.set(obj, "b", 2);
    expect(obj).toEqual({ a: 1, b: 2 });
  });

  it("should overwrite existing values", () => {
    const obj = { a: { b: 1 } };
    deep.set(obj, "a.b", 2);
    expect(obj.a.b).toEqual(2);
  });

  it("should handle mixed array and object paths", () => {
    const obj: any = {};
    deep.set(obj, "a.0.b.1.c", "value");
    expect(obj.a[0].b[1].c).toEqual("value");
  });

  it("should handle setting on frozen objects without throwing", () => {
    const frozen = Object.freeze({ a: { b: 1 } });
    expect(() => deep.set(frozen, "a.c", 2)).not.toThrow();
  });

  it("should handle setting through sealed objects", () => {
    const sealed: any = { a: { b: 1 } };
    Object.seal(sealed.a);
    deep.set(sealed, "a.b", 2);
    expect(sealed.a.b).toBe(2);
  });

  it("should handle unicode keys in paths", () => {
    const obj: any = {};
    deep.set(obj, "🔑.中文.العربية", "multilingual");
    expect(obj["🔑"]["中文"]["العربية"]).toBe("multilingual");
  });

  it("should handle keys with special regex characters", () => {
    const obj: any = {};
    deep.set(obj, "a[0].b(1).c{2}", "special");
    expect(obj["a[0]"]["b(1)"]["c{2}"]).toBe("special");
  });

  it("should handle setting through getter/setter properties", () => {
    const obj: any = {
      _value: 0,
      get prop() {
        return { nested: this._value };
      },
      set prop(v) {
        this._value = v.nested;
      },
    };
    deep.set(obj, "prop.nested", 42);
    expect(obj.prop.nested).toBe(0);
  });

  it("should handle setting on sparse arrays", () => {
    // eslint-disable-next-line no-sparse-arrays
    const obj = { arr: [1, , , 4] };
    deep.set(obj, "arr.1", 2);
    expect(obj.arr[1]).toBe(2);
    expect(obj.arr.length).toBe(4);
  });

  it("should handle setting beyond array bounds", () => {
    const obj = { arr: [1, 2] };
    deep.set(obj, "arr.5", 6);
    expect(obj.arr[5]).toBe(6);
    expect(obj.arr.length).toBe(6);
  });

  it("should handle circular reference in path", () => {
    const obj: any = { a: {} };
    obj.a.b = obj.a;
    deep.set(obj, "a.b.c", "value");
    expect(obj.a.c).toBe("value");
  });

  it("should handle setting on null prototype objects", () => {
    const nullProto = Object.create(null);
    deep.set(nullProto, "a.b.c", "value");
    expect(nullProto.a.b.c).toBe("value");
  });

  it("should handle setting values with toString/valueOf overrides", () => {
    const value = {
      toString: () => "string",
      valueOf: () => 42,
      actual: "real",
    };
    const obj: any = {};
    deep.set(obj, "a.b", value);
    expect(obj.a.b.actual).toBe("real");
  });

  it("should handle empty string as key at various levels", () => {
    const obj: any = {};
    deep.set(obj, "a..b...c", "empty");
    expect(obj.a[""].b[""][""].c).toBe("empty");
  });

  it("should handle array-like objects", () => {
    const arrayLike: any = { 0: "a", 1: "b", length: 2 };
    deep.set(arrayLike, "2", "c");
    expect(arrayLike[2]).toBe("c");
    deep.set(arrayLike, "length", 3);
    expect(arrayLike.length).toBe(3);
  });

  it("should handle setting in keyed arrays with duplicate keys", () => {
    const obj = {
      items: [
        { key: "duplicate", value: 1 },
        { key: "duplicate", value: 2 },
      ],
    };
    deep.set(obj, "items.duplicate.value", 99);
    expect(obj.items[0].value).toBe(99);
    expect(obj.items[1].value).toBe(2);
  });

  it("should throw for a key on an empty list instead of writing it onto the array", () => {
    const obj: any = { items: [] };
    expect(() => deep.set(obj, "items.newKey.value", 42)).toThrow(
      "Cannot set items.newKey.value: the list holds no entry with the key newKey",
    );
    expect(obj.items).toEqual([]);
    expect(Object.keys(obj.items)).toEqual([]);
  });

  it("should handle mixed numeric and string keys in objects", () => {
    const obj: any = { "123": { "456": {} } };
    deep.set(obj, "123.456.789", "numeric");
    expect(obj["123"]["456"]["789"]).toBe("numeric");
  });

  it("should handle setting Symbol properties", () => {
    const sym = Symbol("test");
    const obj: any = {};
    deep.set(obj, `a.${String(sym)}`, "symbol");
    expect(obj.a[String(sym)]).toBe("symbol");
  });

  it("should handle very deep recursive structures", () => {
    const obj: any = {};
    let path = "";
    for (let i = 0; i < 100; i++) path = path ? `${path}.level${i}` : `level${i}`;

    deep.set(obj, path, "deep");
    expect(deep.get(obj, path)).toBe("deep");
  });

  it("should handle setting on functions with properties", () => {
    const fn: any = function () {};
    fn.nested = { value: 1 };
    deep.set(fn, "nested.value", 2);
    expect(fn.nested.value).toBe(2);
  });

  it("should handle keys with newlines and tabs", () => {
    const obj: any = {};
    deep.set(obj, "line1\nline2.tab\there", "whitespace");
    expect(obj["line1\nline2"]["tab\there"]).toBe("whitespace");
  });

  it("should handle setting through Proxy objects", () => {
    const target = { a: { b: 1 } };
    const proxy = new Proxy(target, {
      get(target, prop) {
        return Reflect.get(target, prop);
      },
    });
    deep.set(proxy, "a.b", 2);
    expect(target.a.b).toBe(2);
  });

  it("should handle alternating keyed arrays and regular arrays", () => {
    const obj: any = {
      items: [{ key: "first", data: [1, 2, 3] }],
    };
    deep.set(obj, "items.first.data.1", 99);
    expect(obj.items[0].data[1]).toBe(99);
  });

  it("should handle setting values that are undefined", () => {
    const obj: any = { a: { b: 1 } };
    deep.set(obj, "a.b", undefined);
    expect(obj.a.b).toBeUndefined();
    expect("b" in obj.a).toBe(true);
  });

  it("should handle complex keyed array navigation with periods", () => {
    const obj = {
      sections: [{ key: "user.profile", settings: { theme: "light" } }],
    };
    deep.set(obj, "sections.user.profile.settings.theme", "dark");
    expect(obj.sections[0].settings.theme).toBe("dark");
  });

  it("should handle setting on Map-like objects", () => {
    const mapLike: any = {
      _data: {},
      get(key: string) {
        return this._data[key];
      },
      set(key: string, value: any) {
        this._data[key] = value;
      },
    };
    deep.set(mapLike, "_data.key", "value");
    expect(mapLike._data.key).toBe("value");
  });

  it("should handle setting with mixed types in path", () => {
    const obj: any = {
      true: { false: { null: { undefined: {} } } },
    };
    deep.set(obj, "true.false.null.undefined.NaN", "special");
    expect(obj.true.false.null.undefined.NaN).toBe("special");
  });

  describe("keyed lists", () => {
    const createList = () => ({
      items: [
        { key: "a", value: 1 },
        { key: "b", value: 2 },
      ],
    });

    it("should throw for a missing entry at the end of the path", () => {
      const obj = createList();
      expect(() => deep.set(obj, "items.c", { key: "c", value: 3 })).toThrow(
        "Cannot set items.c: the list holds no entry with the key c",
      );
      expect(obj).toEqual(createList());
    });

    it("should throw for a missing entry in the middle of the path", () => {
      const obj = createList();
      expect(() => deep.set(obj, "items.c.value", 3)).toThrow(
        "Cannot set items.c.value: the list holds no entry with the key c",
      );
      expect(obj).toEqual(createList());
      expect(Object.keys(obj.items)).toEqual(["0", "1"]);
    });

    it("should throw for an entry key with dots that the list does not hold", () => {
      const obj = createList();
      expect(() => deep.set(obj, "items.ns=1;s=a.b.value", 3)).toThrow(
        "Cannot set items.ns=1;s=a.b.value: the list holds no entry with the key ns=1;s=a",
      );
      expect(obj).toEqual(createList());
    });

    it("should replace an entry found by its key", () => {
      const obj = createList();
      deep.set(obj, "items.b", { key: "b", value: 5 });
      expect(obj.items).toEqual([
        { key: "a", value: 1 },
        { key: "b", value: 5 },
      ]);
    });

    it("should write into an entry with a dot in its key", () => {
      const obj = { items: [{ key: "ns=1;s=a.b", port: 1 }] };
      deep.set(obj, "items.ns=1;s=a.b.port", 2);
      expect(obj.items[0].port).toBe(2);
    });

    it("should add a field that an entry does not hold yet", () => {
      const obj: any = createList();
      deep.set(obj, "items.a.extra", true);
      expect(obj.items[0]).toEqual({ key: "a", value: 1, extra: true });
    });

    it("should write an index past the end of a list", () => {
      const obj: any = createList();
      deep.set(obj, "items.2", { key: "c", value: 3 });
      expect(obj.items[2]).toEqual({ key: "c", value: 3 });
    });
  });

  describe("missing paths", () => {
    it("should replace null in the middle of the path with an object", () => {
      const obj: any = { a: { b: null } };
      deep.set(obj, "a.b.c", 1);
      expect(obj).toEqual({ a: { b: { c: 1 } } });
    });

    it("should replace a null list entry with an object", () => {
      const obj: any = { a: [null] };
      deep.set(obj, "a.0.b", 1);
      expect(obj.a).toEqual([{ b: 1 }]);
    });

    it("should build an array inside a new object for an index part", () => {
      const obj: any = { a: {} };
      deep.set(obj, "a.b.0", "x");
      expect(obj.a.b).toEqual(["x"]);
    });

    it("should write the empty path as the empty key", () => {
      const obj: any = { a: 1 };
      deep.set(obj, "", 2);
      expect(obj).toEqual({ a: 1, "": 2 });
    });

    it("should throw when a primitive sits in the middle of the path", () => {
      const obj = { a: "text" };
      expect(() => deep.set(obj, "a.b", 1)).toThrow(TypeError);
      expect(obj).toEqual({ a: "text" });
    });
  });

  describe("keys that hold the separator", () => {
    it("should overwrite a key with a dot instead of building nested objects", () => {
      const obj: any = { "a.b": 1 };
      deep.set(obj, "a.b", 2);
      expect(obj).toEqual({ "a.b": 2 });
    });

    it("should add a key below a key with a dot", () => {
      const obj: any = { "a.b": { c: 1 } };
      deep.set(obj, "a.b.d", 2);
      expect(obj).toEqual({ "a.b": { c: 1, d: 2 } });
    });
  });

  describe("canSet", () => {
    const cases: Array<[string, unknown, string, boolean]> = [
      ["a path that resolves", { a: { b: 1 } }, "a.b", true],
      ["missing objects on the way", {}, "a.b.c", true],
      ["a null value on the way", { a: null }, "a.b", true],
      ["the index just past the end of a list", { items: ["a"] }, "items.1", true],
      ["a new field of an entry", { items: [{ key: "x" }] }, "items.x.port", true],
      ["a key that names no entry", { items: [{ key: "x" }] }, "items.y", false],
      [
        "a field of an entry that is gone",
        { items: [{ key: "x" }] },
        "items.y.a",
        false,
      ],
      ["a key on an empty list", { items: [] }, "items.y", false],
    ];
    it.each(cases)("should be %s: %#", (_, obj, path, expected) => {
      expect(deep.canSet(obj, path)).toBe(expected);
    });
    it.each(cases)("should agree with whether set throws for %s", (_, obj, path) => {
      const settable = deep.canSet(obj, path);
      const write = () => deep.set(obj, path, 1);
      if (settable) expect(write).not.toThrow();
      else expect(write).toThrow();
    });
  });
});
