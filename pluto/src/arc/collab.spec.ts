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

import { CollabText, diff } from "@/arc/collab";

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
  });
});
