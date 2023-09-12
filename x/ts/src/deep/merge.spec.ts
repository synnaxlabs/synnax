// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, it, expect } from "vitest";

import { deep } from "@/deep";

interface TestRecord {
  a: number;
  b?: {
    c?: number;
    d?: number;
  };
}

describe("deepMerge", () => {
  it("should deep two objects", () => {
    const a: TestRecord = {
      a: 1,
      b: {
        c: 2,
      },
    };
    const b: TestRecord = {
      a: 3,
      b: {
        d: 4,
      },
    };
    const c: TestRecord = {
      a: 3,
      b: {
        c: 2,
        d: 4,
      },
    };
    expect(deep.merge(a, b)).toEqual(c);
  });
  it("Should set a value even when its parent is undefined", () => {
    const a: TestRecord = {
      a: 1,
    };
    const b: TestRecord = {
      a: 3,
      b: {
        d: 4,
      },
    };
    const c: TestRecord = {
      a: 3,
      b: {
        d: 4,
      },
    };
    expect(deep.merge(a, b)).toEqual(c);
  });
});
