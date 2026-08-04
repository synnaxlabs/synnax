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
import { ReferenceText } from "@/crdt/reference";

// mulberry32 is a tiny seedable PRNG so schedules are reproducible per seed.
const mulberry32 = (seed: number): (() => number) => {
  let a = seed;
  return () => {
    a |= 0;
    a = (a + 0x6d2b79f5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t ^= t + Math.imul(t ^ (t >>> 7), 61 | t);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
};

const ALPHABET = "abcdefghijklmnopqrstuvwxyz";

/** Pair drives the production document and the frozen per-character reference model
 * through identical edits, asserting that both generate identical operations. */
class Pair {
  readonly p: crdt.Text;
  readonly r: ReferenceText;

  constructor(replica: number) {
    this.p = new crdt.Text(replica);
    this.r = new ReferenceText(replica);
  }

  insert(index: number, text: string): crdt.Insert[] {
    const ops = this.p.insert(index, text);
    expect(this.r.insert(index, text)).toEqual(ops);
    return ops;
  }

  delete(index: number, length: number): crdt.Delete[] {
    const ops = this.p.delete(index, length);
    expect(this.r.delete(index, length)).toEqual(ops);
    return ops;
  }

  applyInsert(ops: crdt.Insert[]): void {
    this.p.applyInsert(...ops);
    this.r.applyInsert(...ops);
  }

  applyDelete(ops: crdt.Delete[]): void {
    this.p.applyDelete(...ops);
    this.r.applyDelete(...ops);
  }
}

/** expectEquivalent asserts that the production document is observably identical to
 * the reference model: value, length, per-index ids, and snapshots that round-trip
 * across the two implementations in both directions. */
const expectEquivalent = (pr: Pair): void => {
  expect(pr.p.toString()).toEqual(pr.r.toString());
  expect(pr.p.len()).toEqual(pr.r.len());
  for (let i = 0; i < pr.p.len(); i++)
    expect(pr.p.indexToID(i)).toEqual(pr.r.indexToID(i));
  expect(pr.p.indexToID(pr.p.len())).toBeNull();
  const intoRef = new ReferenceText(99);
  intoRef.load(pr.p.snapshot());
  expect(intoRef.toString()).toEqual(pr.r.toString());
  const intoProd = new crdt.Text(99);
  intoProd.load(pr.r.snapshot());
  expect(intoProd.toString()).toEqual(pr.p.toString());
};

interface Recorded {
  owner: number;
  inserts?: crdt.Insert[];
  deletes?: crdt.Delete[];
}

const applyTo = (rec: Recorded, pr: Pair): void => {
  if (rec.inserts != null) pr.applyInsert(rec.inserts);
  else if (rec.deletes != null) pr.applyDelete(rec.deletes);
};

describe("differential against the per-character reference", () => {
  describe("random schedules match the reference model exactly", () => {
    for (const seed of [3, 11, 17, 23, 51, 77, 123, 5150, 90210, 424242])
      it(`seed ${seed}`, () => {
        const rng = mulberry32(seed);
        const randInt = (n: number): number => Math.floor(rng() * n);
        const REPLICAS = 3;
        const ROUNDS = 30;
        const pairs = Array.from({ length: REPLICAS }, (_, i) => new Pair(i + 1));
        const log: Recorded[] = [];
        for (let round = 0; round < ROUNDS; round++) {
          const roundOps: Recorded[] = [];
          for (let i = 0; i < pairs.length; i++) {
            const pr = pairs[i];
            const n = pr.p.len();
            if (n > 0 && randInt(3) === 0) {
              const index = randInt(n);
              const length = 1 + randInt(Math.min(n - index, 5));
              roundOps.push({ owner: i, deletes: pr.delete(index, length) });
              continue;
            }
            const text = Array.from(
              { length: 1 + randInt(4) },
              () => ALPHABET[randInt(ALPHABET.length)],
            ).join("");
            roundOps.push({ owner: i, inserts: pr.insert(randInt(n + 1), text) });
          }
          log.push(...roundOps);
          for (let i = 0; i < pairs.length; i++) {
            const delivery = roundOps.map((_, j) => j);
            for (let j = delivery.length - 1; j > 0; j--) {
              const k = randInt(j + 1);
              [delivery[j], delivery[k]] = [delivery[k], delivery[j]];
            }
            for (const j of delivery)
              if (roundOps[j].owner !== i) applyTo(roundOps[j], pairs[i]);
          }
          if (round % 10 === 9) for (const pr of pairs) expectEquivalent(pr);
        }
        const want = pairs[0].p.toString();
        for (const pr of pairs) {
          expectEquivalent(pr);
          expect(pr.p.toString()).toEqual(want);
        }

        // A fresh pair replays the entire shuffled history, exercising pending-op
        // buffering and tombstone-before-insert arrival in both implementations.
        const shuffled = [...log];
        for (let j = shuffled.length - 1; j > 0; j--) {
          const k = randInt(j + 1);
          [shuffled[j], shuffled[k]] = [shuffled[k], shuffled[j]];
        }
        const fresh = new Pair(999);
        for (const rec of shuffled) applyTo(rec, fresh);
        expectEquivalent(fresh);
        expect(fresh.p.toString()).toEqual(want);
      });
  });

  it("matches on a concurrent insert into the middle of a run", () => {
    const a = new Pair(1);
    const b = new Pair(2);
    const seed = a.insert(0, "abcdef");
    b.applyInsert(seed);
    const mid = b.insert(3, "XY");
    a.applyInsert(mid);
    expectEquivalent(a);
    expectEquivalent(b);
    expect(a.p.toString()).toEqual(b.p.toString());
  });

  it("matches when a run's operations are redelivered after a split", () => {
    const a = new Pair(1);
    const b = new Pair(2);
    const seed = a.insert(0, "abcdef");
    b.applyInsert(seed);
    const mid = b.insert(3, "XY");
    a.applyInsert(mid);
    a.applyInsert(seed);
    b.applyInsert(seed);
    expectEquivalent(a);
    expectEquivalent(b);
    expect(a.p.toString()).toEqual(b.p.toString());
  });

  it("matches when deletes arrive before the run they tombstone", () => {
    const a = new Pair(1);
    const inserts = a.insert(0, "hello world");
    const deletes = a.delete(2, 6);
    const late = new Pair(2);
    late.applyDelete(deletes);
    late.applyInsert(inserts);
    expectEquivalent(late);
    expect(late.p.toString()).toEqual(a.p.toString());
  });

  it("matches on deletes spanning a split boundary", () => {
    const a = new Pair(1);
    const b = new Pair(2);
    const seed = a.insert(0, "abcdef");
    b.applyInsert(seed);
    const mid = b.insert(3, "XY");
    a.applyInsert(mid);
    const wipe = a.delete(1, 6);
    b.applyDelete(wipe);
    expectEquivalent(a);
    expectEquivalent(b);
    expect(a.p.toString()).toEqual(b.p.toString());
  });

  it("matches on multi-code-point text", () => {
    const a = new Pair(1);
    a.insert(0, "héllo→世界");
    a.insert(3, "🚀x");
    a.delete(1, 4);
    expectEquivalent(a);
  });
});
