// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bench, describe } from "vitest";

import { crdt } from "@/crdt";

// SIZES are document lengths in characters. They bracket the realistic range for an Arc
// source file (low thousands) and push past it to expose the scaling curve.
const SIZES = [100, 1000, 5000, 25000];

const buildDoc = (n: number): crdt.Text => {
  const t = new crdt.Text(1);
  for (let i = 0; i < n; i++) t.insert(i, "a");
  return t;
};

// Build measures typing a document of n characters from empty, one keystroke at a time.
// Each insert re-traverses the whole document, so the build is O(n^2); dividing the
// per-iteration time by n gives the average per-keystroke cost, which rises with n.
describe("build", () => {
  for (const n of SIZES)
    bench(`n=${n}`, () => {
      const t = new crdt.Text(1);
      for (let i = 0; i < n; i++) t.insert(i, "a");
    });
});

// BuildAndRender measures the real editor loop: every keystroke inserts a character and
// then materializes the whole string (what the binding diffs against). Both halves are
// O(n) per keystroke, so the loop is O(n^2).
describe("buildAndRender", () => {
  for (const n of SIZES)
    bench(`n=${n}`, () => {
      const t = new crdt.Text(1);
      for (let i = 0; i < n; i++) {
        t.insert(i, "a");
        t.toString();
      }
    });
});

// ApplyRemote measures a replica integrating n insert operations produced by another
// replica, delivered in order (the receiver's hot path while a peer types).
describe("applyRemote", () => {
  for (const n of SIZES) {
    const ops = new crdt.Text(2).insert(0, "a".repeat(n));
    bench(`n=${n}`, () => {
      new crdt.Text(1).applyInsert(...ops);
    });
  }
});

// Bootstrap measures a joining client reconstructing a document of n characters from a
// server snapshot.
describe("bootstrap", () => {
  for (const n of SIZES) {
    const snapshot = buildDoc(n).snapshot();
    bench(`n=${n}`, () => {
      new crdt.Text(1).load(snapshot);
    });
  }
});
