// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { type Diff, diff, utf16Offset } from "@/code/text";

// applyDiff replays a Diff against old the way a consumer must: splice out deleteCount
// code points at index and insert the replacement code points in their place. The core
// contract of diff is that this reconstructs the new string exactly.
const applyDiff = (old: string, d: Diff): string => {
  const cp = Array.from(old);
  cp.splice(d.index, d.deleteCount, ...Array.from(d.insert));
  return cp.join("");
};

describe("code/text", () => {
  describe("diff", () => {
    describe("roundtrip contract", () => {
      it.each([
        { name: "a pure insertion", old: "hello world", next: "hello brave world" },
        { name: "a pure deletion", old: "hello brave world", next: "hello world" },
        {
          name: "a replacement",
          old: "the quick brown fox",
          next: "the slow brown fox",
        },
      ])("should reconstruct next from old for $name", ({ old, next }) => {
        expect(applyDiff(old, diff(old, next))).toEqual(next);
      });
    });

    describe("minimal change", () => {
      it.each<{ name: string; old: string; next: string; want: Diff }>([
        {
          name: "a common prefix and suffix, leaving only the middle",
          old: "hello world",
          next: "hello brave world",
          want: { index: 6, deleteCount: 0, insert: "brave " },
        },
        {
          name: "an insertion at the head",
          old: "bc",
          next: "abc",
          want: { index: 0, deleteCount: 0, insert: "a" },
        },
        {
          name: "an insertion at the tail",
          old: "ab",
          next: "abc",
          want: { index: 2, deleteCount: 0, insert: "c" },
        },
        {
          name: "a deletion at the head",
          old: "abc",
          next: "bc",
          want: { index: 0, deleteCount: 1, insert: "" },
        },
        {
          name: "a deletion at the tail",
          old: "abc",
          next: "ab",
          want: { index: 2, deleteCount: 1, insert: "" },
        },
        {
          name: "a single-character replacement in the middle",
          old: "cat",
          next: "cot",
          want: { index: 1, deleteCount: 1, insert: "o" },
        },
      ])("should strip $name", ({ old, next, want }) => {
        expect(diff(old, next)).toEqual(want);
      });
    });

    describe("degenerate inputs", () => {
      it("should produce a no-op diff for identical strings", () => {
        const d = diff("identical", "identical");
        expect(d.deleteCount).toEqual(0);
        expect(d.insert).toEqual("");
        expect(applyDiff("identical", d)).toEqual("identical");
      });

      it("should insert the whole string when old is empty", () => {
        expect(diff("", "abc")).toEqual({ index: 0, deleteCount: 0, insert: "abc" });
      });

      it("should delete the whole string when next is empty", () => {
        expect(diff("abc", "")).toEqual({ index: 0, deleteCount: 3, insert: "" });
      });

      it("should produce a no-op diff for two empty strings", () => {
        expect(applyDiff("", diff("", ""))).toEqual("");
      });
    });

    describe("overlapping prefix and suffix", () => {
      it("should not let the prefix and suffix overlap when shrinking a run", () => {
        const d = diff("aaa", "aa");
        expect(d.deleteCount).toEqual(1);
        expect(applyDiff("aaa", d)).toEqual("aa");
      });

      it("should not let the prefix and suffix overlap when growing a run", () => {
        const d = diff("aa", "aaa");
        expect(d.deleteCount).toEqual(0);
        expect(d.insert).toEqual("a");
        expect(applyDiff("aa", d)).toEqual("aaa");
      });

      it("should reconstruct when removing a char between matching neighbors", () => {
        expect(applyDiff("aba", diff("aba", "aa"))).toEqual("aa");
      });
    });

    describe("multi-code-point characters", () => {
      it.each<{ name: string; old: string; next: string; want: Diff }>([
        {
          name: "measure index in code points, not UTF-16 units",
          old: "a😀b",
          next: "a😀c",
          want: { index: 2, deleteCount: 1, insert: "c" },
        },
        {
          name: "insert an emoji as a single code point",
          old: "ab",
          next: "a😀b",
          want: { index: 1, deleteCount: 0, insert: "😀" },
        },
        {
          name: "delete an emoji as a single code point",
          old: "a😀b",
          next: "ab",
          want: { index: 1, deleteCount: 1, insert: "" },
        },
        {
          name: "replace one emoji with another",
          old: "x😀y",
          next: "x🚀y",
          want: { index: 1, deleteCount: 1, insert: "🚀" },
        },
      ])("should $name", ({ old, next, want }) => {
        const d = diff(old, next);
        expect(d).toEqual(want);
        expect(applyDiff(old, d)).toEqual(next);
      });
    });

    describe("exhaustive roundtrip", () => {
      // Enumerate every string of length 0..3 over an alphabet that includes a
      // surrogate-pair emoji, then assert applyDiff(a, diff(a, b)) === b for every
      // ordered pair. A deterministic stand-in for property-based fuzzing that exercises
      // every prefix/suffix overlap and code-point boundary case at once.
      const alphabet = ["a", "b", "😀"];
      const strings: string[] = [];
      for (let len = 0; len <= 3; len++) {
        const count = alphabet.length ** len;
        for (let i = 0; i < count; i++) {
          let s = "";
          let n = i;
          for (let p = 0; p < len; p++) {
            s += alphabet[n % alphabet.length];
            n = Math.floor(n / alphabet.length);
          }
          strings.push(s);
        }
      }

      it("should reconstruct next for every ordered pair of small strings", () => {
        for (const a of strings)
          for (const b of strings)
            expect(applyDiff(a, diff(a, b)), `diff(${a}, ${b})`).toEqual(b);
      });
    });
  });

  describe("utf16Offset", () => {
    it.each([
      {
        name: "equal the code-point index for ASCII text",
        s: "hello",
        index: 3,
        want: 3,
      },
      { name: "be 0 at the start of the string", s: "a😀b", index: 0, want: 0 },
      {
        name: "count a surrogate pair before the index as two UTF-16 units",
        s: "a😀b",
        index: 2,
        want: 3,
      },
      {
        name: "return the full UTF-16 length at the end of the string",
        s: "a😀b",
        index: 3,
        want: "a😀b".length,
      },
      { name: "account for multiple surrogate pairs", s: "😀😀x", index: 2, want: 4 },
    ])("should $name", ({ s, index, want }) => {
      expect(utf16Offset(s, index)).toEqual(want);
    });

    it("should map the diff boundaries a consumer reads back to UTF-16 offsets", () => {
      const old = "a😀bc";
      const d = diff(old, "a😀Xc");
      expect(utf16Offset(old, d.index)).toEqual(3);
      expect(utf16Offset(old, d.index + d.deleteCount)).toEqual(4);
    });
  });
});
