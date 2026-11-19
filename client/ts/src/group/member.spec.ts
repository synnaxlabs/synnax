// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { destructor, observe, type patch } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { actions } from "@/actions";
import { batch } from "@/batch";
import { group } from "@/group";
import { query } from "@/query";

const G1 = "11111111-1111-4111-8111-111111111111";
const G2 = "22222222-2222-4222-8222-222222222222";

interface Doc {
  key: string;
  name: string;
  group?: string;
}

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
  const build: group.EntryBuilder<string> = (key, acts) => ({
    endpoint: "/doc/dispatch",
    payload: { key, actions: acts },
  });

  it("should build one entry per key that sets the group", () => {
    const acts = [{ type: "set", set: { path: ["group"], value: G1, guards: [] } }];
    expect(group.createMoveRequest(build)(["a", "b"], G1)).toEqual([
      { endpoint: "/doc/dispatch", payload: { key: "a", actions: acts } },
      { endpoint: "/doc/dispatch", payload: { key: "b", actions: acts } },
    ]);
  });

  it("should set the group to null when it is null or undefined", () => {
    const [fromNull] = group.createMoveRequest(build)("a", null);
    const [fromUndefined] = group.createMoveRequest(build)("a");
    expect(fromNull.payload.actions).toEqual([
      { type: "set", set: { path: ["group"], value: null, guards: [] } },
    ]);
    expect(fromUndefined.payload).toEqual(fromNull.payload);
  });

  it("should not revert a later move when the echo of an earlier one arrives", async () => {
    const docs = new query.Table<string, Doc>({ onError: () => {} });
    docs.set([{ key: "a", name: "A" }]);
    const epochs = new observe.Observer<number>();
    const controller = new actions.Controller<string, Doc, patch.Action>({
      store: docs,
      onError: () => {},
      onEpoch: (callback) => epochs.onChange(callback),
      onStreamDown: () => destructor.NOOP,
      send: async () => [],
      reduce: (doc, acts) => ({
        next: acts.reduce(
          (d, a) => (a.type === "set" ? { ...d, group: String(a.set.value) } : d),
          doc,
        ),
        inverse: [],
        targets: ["group"],
      }),
    });
    epochs.notify(1);
    const send = vi.fn().mockResolvedValue({ responses: [{ actions: [] }] });
    const client = new batch.Client({ send, use: vi.fn() });
    const resZ = z.object({ actions: z.custom<patch.Action>().array() });
    const move = group.createMoveRequest<string>((key, acts) =>
      controller.request("/doc/dispatch", key, acts, resZ),
    );
    const [first] = move("a", G1);
    await client.send([first]);
    await client.send(move("a", G2));
    controller.applyRemote(
      "a",
      1,
      first.payload.dispatchKey ?? "",
      first.payload.actions,
    );
    expect(docs.get("a")?.group).toEqual(G2);
  });
});
