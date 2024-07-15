// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect,it } from "vitest";

import { deep } from "@/deep";

interface TestRecord {
  a: number;
  b: {
    c?: number;
    d?: number;
  };
}

describe("deepDelete", () => {
  it("should delete a key", () => {
    const a: TestRecord = {
      a: 1,
      b: {
        c: 2,
      },
    };
    const b: TestRecord = {
      a: 1,
      b: {},
    };
    expect(deep.deleteD<TestRecord, 2>(a, "b.c")).toEqual(b);
  });
});
