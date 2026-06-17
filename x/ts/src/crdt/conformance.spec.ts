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

import { crdt } from "@/crdt";

// The conformance vectors are authored once and shared with the Go implementation so
// that both runtimes materialize identically for the same edit schedule. The canonical
// copy lives alongside the Go package; the operations' wire encoding is owned and
// tested by oracle codegen.
const VECTORS_URL = new URL("../../../go/crdt/testdata/vectors.json", import.meta.url);

interface Step {
  type: "insert" | "delete" | "sync";
  replica: number;
  index: number;
  text?: string;
  length?: number;
}

interface BehaviorVector {
  name: string;
  replicas: number;
  steps: Step[];
  expected: string;
}

const vectors = JSON.parse(readFileSync(VECTORS_URL, "utf-8")) as {
  behavior: BehaviorVector[];
};

interface GenOp {
  owner: number;
  apply: (t: crdt.Text) => void;
}

const runBehavior = (v: BehaviorVector): crdt.Text[] => {
  const docs = Array.from({ length: v.replicas }, (_, i) => new crdt.Text(i + 1));
  const generated: GenOp[] = [];
  const sync = (): void =>
    docs.forEach((d, di) =>
      generated.forEach((g) => {
        if (g.owner !== di) g.apply(d);
      }),
    );
  for (const s of v.steps)
    if (s.type === "insert") {
      const ops = docs[s.replica - 1].insert(s.index, s.text ?? "");
      generated.push({ owner: s.replica - 1, apply: (t) => t.applyInsert(...ops) });
    } else if (s.type === "delete") {
      const ops = docs[s.replica - 1].delete(s.index, s.length ?? 0);
      generated.push({ owner: s.replica - 1, apply: (t) => t.applyDelete(...ops) });
    } else if (s.type === "sync") sync();
  sync();
  return docs;
};

describe("conformance vectors", () => {
  vectors.behavior.forEach((v) => {
    it(`converges: ${v.name}`, () => {
      for (const d of runBehavior(v)) expect(d.toString()).toEqual(v.expected);
    });
  });
});
