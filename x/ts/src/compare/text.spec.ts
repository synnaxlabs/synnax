// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { readFileSync } from "node:fs";

import { describe, expect, it } from "vitest";

import { compare } from "@/compare";

const VECTORS_URL = new URL("../../../testdata/compare/text.json", import.meta.url);

const vectors = JSON.parse(readFileSync(VECTORS_URL, "utf-8")) as {
  ordered: string[];
  equal: string[][];
};

describe("text", () => {
  it("should order each vector before the next", () => {
    for (let i = 1; i < vectors.ordered.length; i++) {
      const a = vectors.ordered[i - 1];
      const b = vectors.ordered[i];
      expect(compare.text(a, b), `${a} before ${b}`).toBe(-1);
      expect(compare.text(b, a), `${b} after ${a}`).toBe(1);
    }
  });

  it("should order every member of an equal group the same", () => {
    for (const group of vectors.equal)
      for (const s of group.slice(1)) expect(compare.text(s, group[0])).toBe(0);
  });
});

describe("codePoints", () => {
  it("should order a code point above U+FFFF after U+FFFF", () => {
    expect(compare.codePoints("\u{1d4d0}", "\uffff")).toBe(1);
    expect(compare.codePoints("\uffff", "\u{1d4d0}")).toBe(-1);
  });

  it("should not fold case", () => {
    expect(compare.codePoints("B", "a")).toBe(-1);
    expect(compare.codePoints("ab", "ab")).toBe(0);
  });

  it("should order a prefix first", () => {
    expect(compare.codePoints("ab", "abc")).toBe(-1);
  });
});
