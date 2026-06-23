// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { crdt } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { changesToDiffs, CollabText, diff } from "@/arc/text/collab";

const snapshotOf = (
  text: string,
): { inserts: crdt.Insert[]; deletes: crdt.Delete[] } => {
  const seed = new crdt.Text(1);
  seed.insert(0, text);
  return seed.snapshot();
};

describe("collab", () => {
  describe("diff", () => {
    it("should find a pure insertion", () => {
      expect(diff("ac", "abc")).toEqual({ index: 1, deleteCount: 0, insert: "b" });
    });
    it("should find a pure deletion", () => {
      expect(diff("abc", "ac")).toEqual({ index: 1, deleteCount: 1, insert: "" });
    });
    it("should find a replacement", () => {
      expect(diff("abc", "axc")).toEqual({ index: 1, deleteCount: 1, insert: "x" });
    });
    it("should find an append", () => {
      expect(diff("ab", "abcd")).toEqual({ index: 2, deleteCount: 0, insert: "cd" });
    });
    it("should be empty for equal strings", () => {
      expect(diff("abc", "abc")).toEqual({ index: 3, deleteCount: 0, insert: "" });
    });
    it("should diff in code points", () => {
      expect(diff("a世c", "a界c")).toEqual({ index: 1, deleteCount: 1, insert: "界" });
    });
  });

  describe("CollabText", () => {
    it("should bootstrap from a snapshot", () => {
      const t = CollabText.bootstrap(snapshotOf("héllo→世界"));
      expect(t.value()).toEqual("héllo→世界");
    });

    it("should return no operations when the value is unchanged", () => {
      const t = CollabText.bootstrap(snapshotOf("abc"));
      expect(t.edit("abc")).toEqual([]);
    });

    it("should apply a local insertion and reflect it in the value", () => {
      const t = CollabText.bootstrap(snapshotOf("ac"));
      const ops = t.edit("abc");
      expect(ops).toHaveLength(1);
      expect(t.value()).toEqual("abc");
    });

    it("should round-trip a local edit to a remote replica", () => {
      const snap = snapshotOf("base");
      const a = CollabText.bootstrap(snap);
      const b = CollabText.bootstrap(snap);
      b.applyRemote(a.edit("base!"));
      expect(b.value()).toEqual("base!");
      expect(b.value()).toEqual(a.value());
    });

    it("should converge two replicas editing concurrently", () => {
      const snap = snapshotOf("shared");
      const a = CollabText.bootstrap(snap);
      const b = CollabText.bootstrap(snap);
      const opsA = a.edit("Ashared");
      const opsB = b.edit("sharedB");
      b.applyRemote(opsA);
      a.applyRemote(opsB);
      expect(a.value()).toEqual(b.value());
      expect(a.value()).toEqual("AsharedB");
    });

    it("should converge a replacement edit", () => {
      const snap = snapshotOf("hello world");
      const a = CollabText.bootstrap(snap);
      const b = CollabText.bootstrap(snap);
      b.applyRemote(a.edit("hello there"));
      expect(b.value()).toEqual("hello there");
    });

    it("should apply a precise replacement change", () => {
      const a = CollabText.bootstrap(snapshotOf("abc"));
      a.applyChanges([{ index: 1, deleteCount: 1, insert: "x" }]);
      expect(a.value()).toEqual("axc");
    });

    it("should converge a multi-region change without interleaving the regions", () => {
      const snap = snapshotOf("a_a");
      const a = CollabText.bootstrap(snap);
      const b = CollabText.bootstrap(snap);
      // Two regions edited at once (e.g. find-replace-all): a@0 -> b, a@2 -> b.
      const diffs = changesToDiffs("a_a", [
        { rangeOffset: 0, rangeLength: 1, text: "b" },
        { rangeOffset: 2, rangeLength: 1, text: "b" },
      ]);
      b.applyRemote(a.applyChanges(diffs));
      expect(a.value()).toEqual("b_b");
      expect(b.value()).toEqual(a.value());
    });
  });

  describe("changesToDiffs", () => {
    it("should convert a single insertion", () => {
      expect(
        changesToDiffs("ac", [{ rangeOffset: 1, rangeLength: 0, text: "b" }]),
      ).toEqual([{ index: 1, deleteCount: 0, insert: "b" }]);
    });

    it("should convert a deletion", () => {
      expect(
        changesToDiffs("abc", [{ rangeOffset: 1, rangeLength: 1, text: "" }]),
      ).toEqual([{ index: 1, deleteCount: 1, insert: "" }]);
    });

    it("should order multiple changes highest-offset-first", () => {
      expect(
        changesToDiffs("a_a", [
          { rangeOffset: 0, rangeLength: 1, text: "b" },
          { rangeOffset: 2, rangeLength: 1, text: "b" },
        ]),
      ).toEqual([
        { index: 2, deleteCount: 1, insert: "b" },
        { index: 0, deleteCount: 1, insert: "b" },
      ]);
    });

    it("should convert UTF-16 offsets after a surrogate pair to code points", () => {
      // "😀" is two UTF-16 code units but one code point: an insert at UTF-16 offset 2
      // lands at code-point index 1.
      expect(
        changesToDiffs("😀ab", [{ rangeOffset: 2, rangeLength: 0, text: "x" }]),
      ).toEqual([{ index: 1, deleteCount: 0, insert: "x" }]);
    });

    it("should count a deleted surrogate pair as one code point", () => {
      expect(
        changesToDiffs("a😀b", [{ rangeOffset: 1, rangeLength: 2, text: "" }]),
      ).toEqual([{ index: 1, deleteCount: 1, insert: "" }]);
    });
  });
});
