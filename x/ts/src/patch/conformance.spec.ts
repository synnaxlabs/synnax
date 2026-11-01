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

import { patch } from "@/patch";
import { type record } from "@/record";

// The vectors are shared with the Go implementation so that both runtimes apply an
// action to a document alike. The canonical copy lives in x/testdata/patch.
const VECTORS_URL = new URL("../../../testdata/patch/vectors.json", import.meta.url);

interface VectorAction {
  type: patch.Action["type"];
  path: string[];
  value?: unknown;
  key?: string;
  after?: string;
  guards?: patch.Guard[];
}

interface Vector {
  name: string;
  doc: record.Unknown;
  actions: VectorAction[];
  expected?: record.Unknown;
  error?: string;
}

const vectors = JSON.parse(readFileSync(VECTORS_URL, "utf-8")) as { cases: Vector[] };

const toAction = ({ type, path, value, key, after, guards = [] }: VectorAction) => {
  switch (type) {
    case "set":
      return { type, set: { path, value, guards } } satisfies patch.Action;
    case "remove":
      return { type, remove: { path, key: key ?? "", guards } } satisfies patch.Action;
    case "insert":
      return { type, insert: { path, value, after, guards } } satisfies patch.Action;
    case "move":
      return {
        type,
        move: { path, key: key ?? "", after, guards },
      } satisfies patch.Action;
  }
};

describe("conformance vectors", () => {
  vectors.cases.forEach((v) => {
    it(v.name, () => {
      const doc = structuredClone(v.doc);
      const run = () => v.actions.forEach((a) => patch.apply(doc, toAction(a)));
      if (v.error != null) {
        expect(run).toThrow(v.error);
        return;
      }
      run();
      expect(doc).toEqual(v.expected);
    });

    if (v.error == null)
      it(`undoes: ${v.name}`, () => {
        const doc = structuredClone(v.doc);
        let inverse: patch.Action[] = [];
        v.actions.forEach((a) => {
          inverse = [...patch.apply(doc, toAction(a)), ...inverse];
        });
        inverse.forEach((a) => patch.apply(doc, a));
        expect(doc).toEqual(v.doc);
      });
  });
});
