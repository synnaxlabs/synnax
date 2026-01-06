// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Task } from "@/hardware/common/task";

interface Test {
  name: string;
  key: string;
  type?: Task.WriteChannelType;
  expected: string;
}

const tests: Test[] = [
  {
    name: "should return the correct ID for a regular channel",
    key: "test",
    expected: "text-test",
  },
  {
    name: "should return the correct ID for a command channel",
    key: "test",
    type: "cmd",
    expected: "text-cmd-test",
  },
  {
    name: "should return the correct ID for a state channel",
    key: "test",
    type: "state",
    expected: "text-state-test",
  },
];

describe("getChannelNameID", () => {
  tests.forEach(({ name, key, type, expected }) => {
    it(name, () => {
      const id = Task.getChannelNameID(key, type);
      expect(id).toBe(expected);
    });
  });
});
