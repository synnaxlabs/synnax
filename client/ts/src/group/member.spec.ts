// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { group } from "@/group";
import { query } from "@/query";

const G1 = "11111111-1111-4111-8111-111111111111";
const G2 = "22222222-2222-4222-8222-222222222222";

interface Doc {
  key: string;
  name: string;
  group?: group.Key;
}

const createTable = (...docs: Doc[]) => {
  const table = new query.Table<string, Doc>({ onError: () => {} });
  table.set(docs);
  return table;
};

describe("MEMBER_LINES", () => {
  it("should match records in any of the groups", () => {
    expect(group.MEMBER_LINES.groups({ group: G1 }, [G1, G2])).toBe(true);
    expect(group.MEMBER_LINES.groups({ group: G2 }, [G1])).toBe(false);
    expect(group.MEMBER_LINES.groups({}, [G1])).toBe(false);
  });

  it("should match no record when the group list is empty", () => {
    expect(group.MEMBER_LINES.groups({ group: G1 }, [])).toBe(false);
  });

  it("should match only records with no group when grouped is false", () => {
    expect(group.MEMBER_LINES.grouped({}, false)).toBe(true);
    expect(group.MEMBER_LINES.grouped({ group: G1 }, false)).toBe(false);
  });

  it("should match only records with a group when grouped is true", () => {
    expect(group.MEMBER_LINES.grouped({ group: G1 }, true)).toBe(true);
    expect(group.MEMBER_LINES.grouped({}, true)).toBe(false);
  });
});

describe("createMoveRequest", () => {
  const setGroupRequest = (table: query.Table<string, Doc>) =>
    group.createMoveRequest<string>("/doc/dispatch", (keys, to) =>
      group.moveCached(table, keys, to),
    );

  it("should build one dispatch entry per key that sets the group", () => {
    expect(setGroupRequest(createTable())(["a", "b"], G1)).toEqual([
      {
        endpoint: "/doc/dispatch",
        payload: {
          key: "a",
          actions: [{ type: "set", set: { path: ["group"], value: G1, guards: [] } }],
        },
        apply: expect.any(Function),
      },
      {
        endpoint: "/doc/dispatch",
        payload: {
          key: "b",
          actions: [{ type: "set", set: { path: ["group"], value: G1, guards: [] } }],
        },
        apply: expect.any(Function),
      },
    ]);
  });

  it("should set the group to null when it is null or undefined", () => {
    const [fromNull] = setGroupRequest(createTable())("a", null);
    const [fromUndefined] = setGroupRequest(createTable())("a");
    expect(fromNull.payload.actions).toEqual([
      { type: "set", set: { path: ["group"], value: null, guards: [] } },
    ]);
    expect(fromUndefined.payload).toEqual(fromNull.payload);
  });

  it("should move only the record of its entry when its cache write applies", () => {
    const table = createTable(
      { key: "a", name: "A", group: G1 },
      { key: "b", name: "B", group: G1 },
    );
    const [first] = setGroupRequest(table)(["a", "b"], G2);
    const rollback = first.apply?.();
    expect(table.get(["a", "b"]).map((r) => r.group)).toEqual([G2, G1]);
    rollback?.();
    expect(table.get("a")?.group).toEqual(G1);
  });
});

describe("moveCached", () => {
  it("should set the group of each cached record", () => {
    const table = createTable({ key: "a", name: "A" }, { key: "b", name: "B" });
    group.moveCached(table, ["a", "b"], G1);
    expect(table.get(["a", "b"]).map((r) => r.group)).toEqual([G1, G1]);
  });

  it("should move records out of their group when the group is undefined", () => {
    const table = createTable({ key: "a", name: "A", group: G1 });
    group.moveCached(table, ["a"]);
    expect(table.get("a")?.group).toBeUndefined();
  });

  it("should skip keys the table does not hold", () => {
    const table = createTable({ key: "a", name: "A" });
    group.moveCached(table, ["a", "missing"], G1);
    expect(table.has("missing")).toBe(false);
  });

  it("should restore only the group on rollback", () => {
    const table = createTable({ key: "a", name: "A", group: G1 });
    const rollback = group.moveCached(table, ["a"], G2);
    table.set("a", (r) => (r == null ? undefined : { ...r, name: "renamed" }));
    rollback();
    expect(table.get("a")).toEqual({ key: "a", name: "renamed", group: G1 });
  });
});
