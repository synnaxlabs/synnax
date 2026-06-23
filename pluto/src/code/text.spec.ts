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
      it("should reconstruct next from old for a pure insertion", () => {
        const old = "hello world";
        const next = "hello brave world";
        expect(applyDiff(old, diff(old, next))).toEqual(next);
      });

      it("should reconstruct next from old for a pure deletion", () => {
        const old = "hello brave world";
        const next = "hello world";
        expect(applyDiff(old, diff(old, next))).toEqual(next);
      });

      it("should reconstruct next from old for a replacement", () => {
        const old = "the quick brown fox";
        const next = "the slow brown fox";
        expect(applyDiff(old, diff(old, next))).toEqual(next);
      });
    });

    describe("minimal change", () => {
      it("should strip a common prefix and suffix, leaving only the middle", () => {
        expect(diff("hello world", "hello brave world")).toEqual({
          index: 6,
          deleteCount: 0,
          insert: "brave ",
        });
      });

      it("should report an insertion at the head", () => {
        expect(diff("bc", "abc")).toEqual({ index: 0, deleteCount: 0, insert: "a" });
      });

      it("should report an insertion at the tail", () => {
        expect(diff("ab", "abc")).toEqual({ index: 2, deleteCount: 0, insert: "c" });
      });

      it("should report a deletion at the head", () => {
        expect(diff("abc", "bc")).toEqual({ index: 0, deleteCount: 1, insert: "" });
      });

      it("should report a deletion at the tail", () => {
        expect(diff("abc", "ab")).toEqual({ index: 2, deleteCount: 1, insert: "" });
      });

      it("should report a single-character replacement in the middle", () => {
        expect(diff("cat", "cot")).toEqual({ index: 1, deleteCount: 1, insert: "o" });
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
      it("should measure index in code points, not UTF-16 units", () => {
        expect(diff("a😀b", "a😀c")).toEqual({
          index: 2,
          deleteCount: 1,
          insert: "c",
        });
      });

      it("should insert an emoji as a single code point", () => {
        expect(diff("ab", "a😀b")).toEqual({
          index: 1,
          deleteCount: 0,
          insert: "😀",
        });
      });

      it("should delete an emoji as a single code point", () => {
        expect(diff("a😀b", "ab")).toEqual({ index: 1, deleteCount: 1, insert: "" });
      });

      it("should replace one emoji with another", () => {
        const d = diff("x😀y", "x🚀y");
        expect(d).toEqual({ index: 1, deleteCount: 1, insert: "🚀" });
        expect(applyDiff("x😀y", d)).toEqual("x🚀y");
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
    it("should equal the code-point index for ASCII text", () => {
      expect(utf16Offset("hello", 3)).toEqual(3);
    });

    it("should be 0 at the start of the string", () => {
      expect(utf16Offset("a😀b", 0)).toEqual(0);
    });

    it("should count a surrogate pair before the index as two UTF-16 units", () => {
      expect(utf16Offset("a😀b", 2)).toEqual(3);
    });

    it("should return the full UTF-16 length at the end of the string", () => {
      expect(utf16Offset("a😀b", 3)).toEqual("a😀b".length);
    });

    it("should account for multiple surrogate pairs", () => {
      expect(utf16Offset("😀😀x", 2)).toEqual(4);
    });

    it("should map the diff boundaries a consumer reads back to UTF-16 offsets", () => {
      const old = "a😀bc";
      const d = diff(old, "a😀Xc");
      expect(utf16Offset(old, d.index)).toEqual(3);
      expect(utf16Offset(old, d.index + d.deleteCount)).toEqual(4);
    });
  });
});
