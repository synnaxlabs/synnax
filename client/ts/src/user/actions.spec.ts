// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { reduceAll } from "@/user/actions";
import { set, setUsername } from "@/user/actions.gen";
import { type User } from "@/user/types.gen";

const USER: User = {
  key: "0198b5a4-0000-7000-8000-000000000001",
  name: "User",
  rootUser: false,
  role: "0198b5a4-0000-7000-8000-000000000002",
};

describe("user actions", () => {
  it("should apply the set of root_user that the Core makes", () => {
    const { next } = reduceAll(USER, [set({ path: ["root_user"], value: true })]);
    expect(next.rootUser).toBe(true);
  });

  it("should leave the user unchanged on a set of its username", () => {
    expect(reduceAll(USER, [setUsername({ username: "other" })]).next).toBe(USER);
  });
});
