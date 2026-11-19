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

const P2 = "22222222-2222-4222-8222-222222222222";
const G1 = "33333333-3333-4333-8333-333333333333";

const build: project.EntryBuilder<string> = (key, actions) => ({
  endpoint: "/doc/dispatch",
  payload: { key, actions },
});

describe("createMoveRequest", () => {
  it("should build one entry per key that sets the project and drops the group", () => {
    const payload = (key: string) => ({
      key,
      actions: [
        { type: "set", set: { path: ["project"], value: P2, guards: [] } },
        { type: "remove", remove: { path: [], key: "group", guards: [] } },
      ],
    });
    expect(project.createMoveRequest(build)(["a", "b"], P2)).toEqual([
      { endpoint: "/doc/dispatch", payload: payload("a") },
      { endpoint: "/doc/dispatch", payload: payload("b") },
    ]);
  });
});

describe("createGroupMoveRequest", () => {
  it("should build one entry per key that sets the group", () => {
    const actions = [{ type: "set", set: { path: ["group"], value: G1, guards: [] } }];
    expect(project.createGroupMoveRequest(build)(["a", "b"], G1)).toEqual([
      { endpoint: "/doc/dispatch", payload: { key: "a", actions } },
      { endpoint: "/doc/dispatch", payload: { key: "b", actions } },
    ]);
  });
});
