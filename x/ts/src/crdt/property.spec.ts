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

const mulberry32 = (seed: number): (() => number) => {
  let s = seed;
  return () => {
    s |= 0;
    s = (s + 0x6d2b79f5) | 0;
    let t = Math.imul(s ^ (s >>> 15), 1 | s);
    t ^= t + Math.imul(t ^ (t >>> 7), 61 | t);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
};

const ALPHABET = "abcdefghijklmnopqrstuvwxyz";

type Edit = (t: crdt.Text) => void;

const randomEdit = (rng: () => number, d: crdt.Text): Edit => {
  const intn = (n: number): number => Math.floor(rng() * n);
  const n = d.len();
  if (n > 0 && intn(3) === 0) {
    const index = intn(n);
    const ops = d.delete(index, 1 + intn(n - index));
    return (t) => t.applyDelete(...ops);
  }
  const index = intn(n + 1);
  const length = 1 + intn(4);
  let text = "";
  for (let i = 0; i < length; i++) text += ALPHABET[intn(ALPHABET.length)];
  const ops = d.insert(index, text);
  return (t) => t.applyInsert(...ops);
};

describe("convergence property", () => {
  it.each([1, 2, 7, 42, 99, 1000, 31337, 808080])(
    "random interleavings converge across replicas and delivery orders (seed %i)",
    (seed) => {
      const rng = mulberry32(seed);
      const replicas = 4;
      const rounds = 60;
      const docs = Array.from({ length: replicas }, (_, i) => new crdt.Text(i + 1));
      const allEdits: Edit[] = [];
      for (let r = 0; r < rounds; r++) {
        const roundEdits = docs.map((d) => randomEdit(rng, d));
        allEdits.push(...roundEdits);
        roundEdits.forEach((e, origin) =>
          docs.forEach((d, j) => {
            if (j !== origin) e(d);
          }),
        );
      }
      const want = docs[0].toString();
      for (const d of docs.slice(1)) expect(d.toString()).toEqual(want);

      const boot = new crdt.Text(1234);
      boot.load(docs[0].snapshot());
      expect(boot.toString()).toEqual(want);

      const shuffled = [...allEdits];
      for (let i = shuffled.length - 1; i > 0; i--) {
        const j = Math.floor(rng() * (i + 1));
        [shuffled[i], shuffled[j]] = [shuffled[j], shuffled[i]];
      }
      const fresh = new crdt.Text(999);
      for (const e of shuffled) e(fresh);
      expect(fresh.toString()).toEqual(want);
    },
  );
});

// keepAB strips s to the run markers A and B so the two concurrent runs can be checked
// for interleaving independent of the surrounding seed text.
const keepAB = (s: string): string =>
  [...s].filter((c) => c === "A" || c === "B").join("");

// contiguousAB matches a string of As and Bs in which all As precede all Bs or all Bs
// precede all As, i.e. the two runs did not interleave.
const contiguousAB = /^A*B*$|^B*A*$/;

describe("non-interleaving property", () => {
  it.each([1, 2, 7, 42, 99, 1000, 31337, 808080])(
    "concurrent same-position runs never interleave (seed %i)",
    (seed) => {
      const rng = mulberry32(seed);
      const intn = (n: number): number => Math.floor(rng() * n);
      const base = new crdt.Text(1);
      let seedText = "";
      for (let i = 0, n = 1 + intn(8); i < n; i++)
        seedText += ALPHABET[intn(ALPHABET.length)];
      base.insert(0, seedText);
      const snap = base.snapshot();
      const a = new crdt.Text(2);
      a.load(snap);
      const b = new crdt.Text(3);
      b.load(snap);

      const pos = intn(base.len() + 1);
      const runLen = 2 + intn(4);
      const opsA = a.insert(pos, "A".repeat(runLen));
      const opsB = b.insert(pos, "B".repeat(runLen));
      b.applyInsert(...opsA);
      a.applyInsert(...opsB);

      expect(a.toString()).toEqual(b.toString());
      const runs = keepAB(a.toString());
      expect(runs).toHaveLength(2 * runLen);
      expect(contiguousAB.test(runs)).toBe(true);
    },
  );
});
