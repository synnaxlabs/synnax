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
    for (const [i, b] of vectors.ordered.entries()) {
      const a = vectors.ordered[i - 1];
      if (a == null) continue;
      expect(compare.text(a, b), `${a} before ${b}`).toBe(-1);
      expect(compare.text(b, a), `${b} after ${a}`).toBe(1);
    }
  });

  it("should order every member of an equal group the same", () => {
    for (const group of vectors.equal)
      for (const s of group) for (const t of group) expect(compare.text(s, t)).toBe(0);
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

describe("natural", () => {
  interface Spec {
    name: string;
    input: string[];
    expected: string[];
  }
  const SPECS: Spec[] = [
    { name: "empty input", input: [], expected: [] },
    {
      name: "pure numbers",
      input: ["10", "9", "2", "1", "100"],
      expected: ["1", "2", "9", "10", "100"],
    },
    {
      name: "digit runs by value",
      input: ["file10", "file2", "file1"],
      expected: ["file1", "file2", "file10"],
    },
    {
      name: "different prefixes",
      input: ["b1", "a2", "a10", "b10", "a1"],
      expected: ["a1", "a2", "a10", "b1", "b10"],
    },
    {
      name: "several digit runs",
      input: ["v1.10.0", "v1.2.10", "v1.2.9", "v10.0.0"],
      expected: ["v1.2.9", "v1.2.10", "v1.10.0", "v10.0.0"],
    },
    {
      name: "text runs by Core text order",
      input: ["Beta 1", "alpha 2", "ALPHA 1"],
      expected: ["ALPHA 1", "alpha 2", "Beta 1"],
    },
    {
      name: "separators as text",
      input: ["a_4", "a-3", "a2", "a.1"],
      expected: ["a2", "a-3", "a.1", "a_4"],
    },
    {
      name: "duplicates",
      input: ["x2", "x1", "x2", "x1"],
      expected: ["x1", "x1", "x2", "x2"],
    },
    {
      name: "a prefix first",
      input: ["ab1", "a", "ab", ""],
      expected: ["", "a", "ab", "ab1"],
    },
  ];
  SPECS.forEach(({ name, input, expected }) => {
    it(`should sort ${name}`, () => {
      expect(input.sort(compare.natural)).toEqual(expected);
    });
  });

  it("should not equate strings that differ only in a separator", () => {
    expect(compare.natural("a-1", "a1")).not.toBe(0);
  });

  it("should not equate strings with no letters or digits", () => {
    expect(compare.natural("-", "_")).toBe(-1);
    expect(compare.natural("", "-")).toBe(-1);
    expect(compare.natural("-", "-")).toBe(0);
  });

  it("should order a digit run before a letter run", () => {
    expect(compare.natural("1a", "a1")).toBe(-1);
  });

  it("should compare digit runs longer than a safe integer by value", () => {
    expect(compare.natural("n90071992547409930", "n90071992547409931")).toBe(-1);
    expect(compare.natural("n100000000000000000000", "n99999999999999999999")).toBe(1);
  });

  it("should order digit runs that differ only in leading zeros the same", () => {
    expect(compare.natural("a007", "a7")).toBe(0);
    expect(compare.natural("a007", "a8")).toBe(-1);
  });
});
