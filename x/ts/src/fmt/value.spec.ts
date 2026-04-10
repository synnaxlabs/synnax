// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { fmt } from "@/fmt";

describe("fmt.value", () => {
  describe("primitives", () => {
    it("should pass strings through unchanged", () => {
      expect(fmt.value("hello")).toBe("hello");
    });

    it("should pass an empty string through unchanged", () => {
      expect(fmt.value("")).toBe("");
    });

    it("should pass integers through unchanged", () => {
      expect(fmt.value(42)).toBe(42);
    });

    it("should pass floats through unchanged", () => {
      expect(fmt.value(3.14)).toBe(3.14);
    });

    it("should pass zero through unchanged", () => {
      expect(fmt.value(0)).toBe(0);
    });

    it("should pass NaN through unchanged", () => {
      expect(fmt.value(NaN)).toBeNaN();
    });

    it("should pass Infinity through unchanged", () => {
      expect(fmt.value(Infinity)).toBe(Infinity);
    });

    it("should pass true through unchanged", () => {
      expect(fmt.value(true)).toBe(true);
    });

    it("should pass false through unchanged", () => {
      expect(fmt.value(false)).toBe(false);
    });

    it("should pass null through unchanged", () => {
      expect(fmt.value(null)).toBeNull();
    });

    it("should render undefined as [undefined]", () => {
      expect(fmt.value(undefined)).toBe("[undefined]");
    });

    it("should render bigint with an n suffix", () => {
      expect(fmt.value(BigInt(42))).toBe("42n");
    });

    it("should render 0n bigint as 0n", () => {
      expect(fmt.value(BigInt(0))).toBe("0n");
    });

    it("should render functions as [function]", () => {
      expect(fmt.value(() => 1)).toBe("[function]");
    });

    it("should render symbols as [symbol]", () => {
      expect(fmt.value(Symbol("x"))).toBe("[symbol]");
    });
  });

  describe("well-known objects", () => {
    it("should render Date as its ISO string", () => {
      expect(fmt.value(new Date("2026-01-01T00:00:00Z"))).toBe(
        "2026-01-01T00:00:00.000Z",
      );
    });

    it("should render Error as [Error: message]", () => {
      expect(fmt.value(new Error("boom"))).toBe("[Error: boom]");
    });

    it("should render non-plain class instances as [object Object]", () => {
      class Custom {}
      expect(fmt.value(new Custom())).toBe("[[object Object]]");
    });
  });

  describe("arrays", () => {
    it("should render an empty array as an empty array", () => {
      expect(fmt.value([])).toEqual([]);
    });

    it("should recurse into array elements preserving order and type", () => {
      expect(fmt.value([1, "two", true, null])).toEqual([1, "two", true, null]);
    });

    it("should truncate arrays longer than maxArrayLength", () => {
      expect(
        fmt.value([0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14], {
          maxArrayLength: 3,
        }),
      ).toEqual([0, 1, 2, "…(+12 more)"]);
    });

    it("should not truncate arrays at or under maxArrayLength", () => {
      expect(fmt.value([1, 2, 3], { maxArrayLength: 3 })).toEqual([1, 2, 3]);
    });

    it("should mark arrays beyond maxDepth without enumerating", () => {
      expect(fmt.value([[[[[1, 2]]]]], { maxDepth: 2 })).toEqual([["[Array(1)]"]]);
    });
  });

  describe("plain objects", () => {
    it("should render an empty object as an empty object", () => {
      expect(fmt.value({})).toEqual({});
    });

    it("should recurse into nested object values", () => {
      expect(fmt.value({ a: 1, b: { c: "deep" } })).toEqual({
        a: 1,
        b: { c: "deep" },
      });
    });

    it("should mark objects beyond maxDepth as [Object]", () => {
      expect(
        fmt.value({ a: { b: { c: { d: { e: "x" } } } } }, { maxDepth: 2 }),
      ).toEqual({ a: { b: "[Object]" } });
    });
  });

  describe("string truncation", () => {
    it("should truncate strings longer than maxStringLength", () => {
      expect(fmt.value("abcdefghij", { maxStringLength: 4 })).toBe("abcd…(+6 chars)");
    });

    it("should leave strings exactly at the limit alone", () => {
      expect(fmt.value("abcd", { maxStringLength: 4 })).toBe("abcd");
    });

    it("should leave strings shorter than the limit alone", () => {
      expect(fmt.value("hi", { maxStringLength: 10 })).toBe("hi");
    });
  });
});

describe("fmt.stringify", () => {
  it("should render a flat object as pretty-printed JSON", () => {
    expect(fmt.stringify({ a: 1, b: "two" })).toBe(
      `{
  "a": 1,
  "b": "two"
}`,
    );
  });

  it("should render a nested object with indented children", () => {
    expect(fmt.stringify({ a: { b: [1, 2] } })).toBe(
      `{
  "a": {
    "b": [
      1,
      2
    ]
  }
}`,
    );
  });

  it("should render a truncated string inline", () => {
    expect(fmt.stringify("abcdefghij", { maxStringLength: 4 })).toBe(
      `"abcd…(+6 chars)"`,
    );
  });

  it("should render a truncated array inline", () => {
    expect(fmt.stringify([1, 2, 3, 4, 5], { maxArrayLength: 2 })).toBe(
      `[
  1,
  2,
  "…(+3 more)"
]`,
    );
  });

  it("should render undefined as its bracketed tag", () => {
    expect(fmt.stringify(undefined)).toBe(`"[undefined]"`);
  });

  it("should render null as the JSON null literal", () => {
    expect(fmt.stringify(null)).toBe("null");
  });

  it("should tolerate circular references without throwing", () => {
    const circular: Record<string, unknown> = {};
    circular.self = circular;
    expect(() => fmt.stringify(circular)).not.toThrow();
  });
});
