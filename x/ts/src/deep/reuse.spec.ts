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
import { TimeStamp } from "@/telem";

describe("reuse", () => {
  it("should return prev when next equals it", () => {
    const prev = { a: { b: [1, 2] }, c: "x" };
    expect(deep.reuse(prev, { a: { b: [1, 2] }, c: "x" })).toBe(prev);
  });

  it("should keep the identity of the unchanged fields of a changed object", () => {
    const prev = { a: { b: 1 }, c: { d: 2 } };
    const res = deep.reuse(prev, { a: { b: 1 }, c: { d: 3 } });
    expect(res).toEqual({ a: { b: 1 }, c: { d: 3 } });
    expect(res).not.toBe(prev);
    expect(res.a).toBe(prev.a);
    expect(res.c).not.toBe(prev.c);
  });

  it("should match the entries of keyed lists by key", () => {
    const a = { key: "a", v: 1 };
    const b = { key: "b", v: 2 };
    const res = deep.reuse(
      [a, b],
      [
        { key: "c", v: 3 },
        { key: "b", v: 2 },
        { key: "a", v: 1 },
      ],
    );
    expect(res[1]).toBe(b);
    expect(res[2]).toBe(a);
  });

  it("should match the entries of other lists by position", () => {
    const first = { v: 1 };
    const res = deep.reuse([first, { v: 2 }], [{ v: 1 }, { v: 3 }]);
    expect(res[0]).toBe(first);
    expect(res[1]).toEqual({ v: 3 });
  });

  it("should return a new list when an entry is removed", () => {
    const prev = [{ key: "a" }, { key: "b" }];
    const res = deep.reuse(prev, [{ key: "a" }]);
    expect(res).not.toBe(prev);
    expect(res[0]).toBe(prev[0]);
  });

  it("should return a new object when a field is removed", () => {
    const prev = { a: 1, b: 2 };
    expect(deep.reuse(prev, { a: 1 })).toEqual({ a: 1 });
  });

  it("should compare a class instance by value", () => {
    const prev = { at: new TimeStamp(5) };
    expect(deep.reuse(prev, { at: new TimeStamp(5) })).toBe(prev);
    expect(deep.reuse(prev, { at: new TimeStamp(6) }).at).toEqual(new TimeStamp(6));
  });

  it("should return next when it differs in kind", () => {
    expect(deep.reuse({ a: 1 }, [1])).toEqual([1]);
    expect(deep.reuse(undefined, { a: 1 })).toEqual({ a: 1 });
  });
});
