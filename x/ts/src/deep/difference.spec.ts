// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { deep } from "@/deep";

describe("deep/difference", () => {
  it("should return a map of differences between two objects", () => {
    const obj1 = {
      a: 1,
      b: 2,
      c: {
        d: 4,
        e: 5,
        f: [6, 7, 8],
      },
    };
    const obj2 = {
      a: 1,
      b: 3,
      c: {
        d: 4,
        e: 6,
        f: [6, 7, 8],
      },
    };
    const diff = deep.difference(obj1, obj2);
    expect(Object.keys(diff).length).toEqual(2);
    expect(diff.b).toEqual([2, 3]);
    expect(diff["c.e"]).toEqual([5, 6]);
  });
});
