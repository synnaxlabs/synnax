// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { crdt } from "@/crdt";

describe("crdt", () => {
  describe("local editing", () => {
    it("should materialize forward insertions", () => {
      const t = new crdt.Text(1);
      t.insert(0, "hello");
      expect(t.toString()).toEqual("hello");
      expect(t.len()).toEqual(5);
    });

    it("should materialize backward insertions", () => {
      const t = new crdt.Text(1);
      for (const ch of ["o", "l", "l", "e", "h"]) t.insert(0, ch);
      expect(t.toString()).toEqual("hello");
    });

    it("should insert into the middle", () => {
      const t = new crdt.Text(1);
      t.insert(0, "ad");
      t.insert(1, "bc");
      expect(t.toString()).toEqual("abcd");
    });

    it("should append at the end", () => {
      const t = new crdt.Text(1);
      t.insert(0, "ab");
      t.insert(t.len(), "cd");
      expect(t.toString()).toEqual("abcd");
    });

    it("should delete a range", () => {
      const t = new crdt.Text(1);
      t.insert(0, "hello world");
      t.delete(5, 6);
      expect(t.toString()).toEqual("hello");
    });

    it("should append after deleting the tail", () => {
      const t = new crdt.Text(1);
      t.insert(0, "ab");
      t.delete(1, 1);
      t.insert(t.len(), "c");
      expect(t.toString()).toEqual("ac");
    });

    it("should support multi-byte code points", () => {
      const t = new crdt.Text(1);
      t.insert(0, "héllo→世界");
      expect(t.toString()).toEqual("héllo→世界");
      expect(t.len()).toEqual(8);
    });

    it("should report the id at a live index", () => {
      const t = new crdt.Text(1);
      const ops = t.insert(0, "ab");
      expect(t.indexToID(1)).toEqual(ops[1].id);
      expect(t.indexToID(2)).toBeNull();
    });

    it("should materialize a document deeper than the recursion limit", () => {
      const t = new crdt.Text(1);
      const n = 50_000;
      t.insert(0, "a".repeat(n));
      expect(t.len()).toEqual(n);
      expect(t.toString()).toEqual("a".repeat(n));
    });
  });

  describe("replication", () => {
    it("should converge two replicas editing the same gap", () => {
      const a = new crdt.Text(1);
      const b = new crdt.Text(2);
      const opsA = a.insert(0, "abc");
      const opsB = b.insert(0, "xyz");
      b.applyInsert(...opsA);
      a.applyInsert(...opsB);
      expect(a.toString()).toEqual(b.toString());
    });

    it("should not interleave concurrent same-position runs", () => {
      const a = new crdt.Text(1);
      const b = new crdt.Text(2);
      const opsA = a.insert(0, "abc");
      const opsB = b.insert(0, "xyz");
      b.applyInsert(...opsA);
      a.applyInsert(...opsB);
      expect(["abcxyz", "xyzabc"]).toContain(a.toString());
    });

    it("should not interleave concurrent backward typing", () => {
      const a = new crdt.Text(1);
      const b = new crdt.Text(2);
      const opsA: crdt.Insert[] = [];
      const opsB: crdt.Insert[] = [];
      for (const ch of ["c", "b", "a"]) opsA.push(...a.insert(0, ch));
      for (const ch of ["z", "y", "x"]) opsB.push(...b.insert(0, ch));
      b.applyInsert(...opsA);
      a.applyInsert(...opsB);
      expect(a.toString()).toEqual(b.toString());
      expect(["abcxyz", "xyzabc"]).toContain(a.toString());
    });

    it("should not interleave concurrent inserts before a shared character", () => {
      const a = new crdt.Text(1);
      const b = new crdt.Text(2);
      const seed = a.insert(0, "o");
      b.applyInsert(...seed);
      const opsA = a.insert(0, "ab");
      const opsB = b.insert(0, "xy");
      b.applyInsert(...opsA);
      a.applyInsert(...opsB);
      expect(a.toString()).toEqual(b.toString());
      expect(["abxyo", "xyabo"]).toContain(a.toString());
    });

    it("should integrate inserts delivered before their origin", () => {
      const a = new crdt.Text(1);
      const b = new crdt.Text(2);
      const ops = a.insert(0, "abc");
      b.applyInsert(ops[2], ops[0], ops[1]);
      expect(b.toString()).toEqual("abc");
    });

    it("should delete a character inserted concurrently", () => {
      const a = new crdt.Text(1);
      const b = new crdt.Text(2);
      const opsA = a.insert(0, "abc");
      b.applyInsert(...opsA);
      const delOps = b.delete(1, 1);
      a.applyDelete(...delOps);
      expect(a.toString()).toEqual("ac");
      expect(b.toString()).toEqual("ac");
    });

    it("should apply a delete that arrives before the insert", () => {
      const a = new crdt.Text(1);
      const b = new crdt.Text(2);
      const insertOps = a.insert(0, "abc");
      const delOps = a.delete(1, 1);
      b.applyDelete(...delOps);
      b.applyInsert(...insertOps);
      expect(b.toString()).toEqual("ac");
    });

    it("should ignore duplicate operations", () => {
      const a = new crdt.Text(1);
      const b = new crdt.Text(2);
      const ops = a.insert(0, "abc");
      b.applyInsert(...ops);
      b.applyInsert(...ops);
      expect(b.toString()).toEqual("abc");
    });
  });
});
