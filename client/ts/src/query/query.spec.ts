// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { NotFoundError } from "@/index";
import { query } from "@/query";

interface Rec extends record.Keyed<string> {
  key: string;
  value: number;
}
const rec = (key: string, value: number): Rec => ({ key, value });

describe("hash", () => {
  it("collapses key orderings to the same hash", () => {
    expect(query.hash({ a: 1, b: 2 })).toEqual(query.hash({ b: 2, a: 1 }));
  });

  it("preserves array order", () => {
    expect(query.hash([1, 2, 3])).not.toEqual(query.hash([3, 2, 1]));
  });

  it("hashes nested objects recursively", () => {
    expect(query.hash({ a: { x: 1, y: 2 } })).toEqual(
      query.hash({ a: { y: 2, x: 1 } }),
    );
  });

  it("hashes null and primitives", () => {
    expect(query.hash(null)).toEqual("null");
    expect(query.hash(undefined)).toEqual("undefined");
    expect(query.hash(42)).toEqual("42");
    expect(query.hash("x")).toEqual('"x"');
  });

  it("disambiguates null from undefined at the top level", () => {
    expect(query.hash(null)).not.toEqual(query.hash(undefined));
  });

  it("keeps null fields distinct from absent while folding undefined", () => {
    const nestedNull = query.hash({ a: null });
    const nestedUndef = query.hash({ a: undefined });
    const absent = query.hash({});
    expect(nestedNull).not.toEqual(absent);
    expect(nestedUndef).toEqual(absent);
  });

  it("hashes bigints without throwing and disambiguates from same-valued numbers", () => {
    expect(query.hash(42n)).toEqual("42n");
    expect(query.hash(42n)).not.toEqual(query.hash(42));
    expect(query.hash({ k: 42n })).toEqual('{"k":42n}');
    expect(() => query.hash({ k: 9007199254740993n })).not.toThrow();
  });

  it("hashes a plain object once per identity", () => {
    const inner = { hash: vi.fn(() => "inner") };
    const q = { id: inner };
    const first = query.hash(q);
    expect(query.hash(q)).toEqual(first);
    expect(inner.hash).toHaveBeenCalledTimes(1);
  });

  it("delegates to primitive.Hashable.hash() for class instances", () => {
    class TaggedID {
      constructor(private readonly v: string) {}
      hash(): string {
        return `tag:${this.v}`;
      }
    }
    expect(query.hash({ id: new TaggedID("abc") })).toEqual('{"id":tag:abc}');
    expect(query.hash(new TaggedID("xyz"))).toEqual("tag:xyz");
  });

  it("produces stable hashes across instances representing the same value", () => {
    class Wrapper {
      constructor(private readonly v: number) {}
      hash(): string {
        return this.v.toString();
      }
    }
    expect(query.hash({ k: new Wrapper(7) })).toEqual(
      query.hash({ k: new Wrapper(7) }),
    );
  });
});

describe("isLive", () => {
  it("accepts a cached answer", () => {
    expect(query.isLive(rec("a", 7))).toBe(true);
  });

  it("rejects an uncached answer", () => {
    expect(query.isLive(undefined)).toBe(false);
  });

  it("rejects a deletion", () => {
    expect(query.isLive(new query.Deleted(rec("a", 7), TimeStamp.now()))).toBe(false);
  });

  it("accepts falsy answers", () => {
    expect(query.isLive(0)).toBe(true);
    expect(query.isLive("")).toBe(true);
    expect(query.isLive(false)).toBe(true);
    expect(query.isLive([])).toBe(true);
  });
});

describe("requireCorpse", () => {
  it("returns the corpse of a deletion", () => {
    const corpse = rec("a", 7);
    expect(query.requireCorpse(new query.Deleted(corpse, TimeStamp.now()))).toEqual(
      corpse,
    );
  });

  it("throws when the answer is live", () => {
    expect(() => query.requireCorpse(rec("a", 7))).toThrow(NotFoundError);
  });

  it("throws when nothing is cached", () => {
    expect(() => query.requireCorpse(undefined)).toThrow(NotFoundError);
  });
});
