// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { reduceAll } from "@/group/actions";
import { setParent } from "@/group/actions.gen";
import { type Group } from "@/group/types.gen";

const GROUP: Group = {
  key: "0198b5a4-0000-7000-8000-000000000001",
  name: "Group",
  scope: "channel",
  parent: { type: "group", key: "0198b5a4-0000-7000-8000-000000000002" },
};

describe("group actions", () => {
  it("should move a group under a parent", () => {
    const parent = {
      type: "group" as const,
      key: "0198b5a4-0000-7000-8000-000000000003",
    };
    expect(reduceAll(GROUP, [setParent({ parent })]).next.parent).toEqual(parent);
  });

  it("should move a group to the top level", () => {
    const { next } = reduceAll(GROUP, [setParent({})]);
    expect("parent" in next).toBe(false);
  });
});
