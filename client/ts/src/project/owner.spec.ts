// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { project } from "@/project";
import { query } from "@/query";

const P1 = "11111111-1111-4111-8111-111111111111";
const P2 = "22222222-2222-4222-8222-222222222222";
const G1 = "33333333-3333-4333-8333-333333333333";

interface Doc {
  key: string;
  project?: project.Key;
  group?: string;
}

const createTable = (...docs: Doc[]) => {
  const table = new query.Table<string, Doc>({ onError: () => {} });
  table.set(docs);
  return table;
};

describe("createMoveRequest", () => {
  const setProjectRequest = (table: query.Table<string, Doc>) =>
    project.createMoveRequest<string>("/doc/dispatch", (keys, to) =>
      project.moveCached(table, keys, to),
    );

  it("should build one entry per key that sets the project and drops the group", () => {
    const actions = [
      { type: "set", set: { path: ["project"], value: P2, guards: [] } },
      { type: "remove", remove: { path: [], key: "group", guards: [] } },
    ];
    expect(setProjectRequest(createTable())(["a", "b"], P2)).toEqual([
      {
        endpoint: "/doc/dispatch",
        payload: { key: "a", actions },
        apply: expect.any(Function),
      },
      {
        endpoint: "/doc/dispatch",
        payload: { key: "b", actions },
        apply: expect.any(Function),
      },
    ]);
  });

  it("should move only the visualization of its entry when its write applies", () => {
    const table = createTable(
      { key: "a", project: P1, group: G1 },
      { key: "b", project: P1, group: G1 },
    );
    const [first] = setProjectRequest(table)(["a", "b"], P2);
    const rollback = first.apply?.();
    expect(table.get(["a", "b"])).toEqual([
      { key: "a", project: P2, group: undefined },
      { key: "b", project: P1, group: G1 },
    ]);
    rollback?.();
    expect(table.get("a")).toEqual({ key: "a", project: P1, group: G1 });
  });
});
