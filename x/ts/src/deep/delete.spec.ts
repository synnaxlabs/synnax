// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { deep } from "@/deep";

interface TestRecord {
  a: number;
  b: {
    c?: number;
    d?: number;
  };
  e?: {
    f?: {
      g?: {};
    };
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
  it("should be fine when depth of recursion is greater than depth of object", () => {
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
    expect(deep.deleteD<TestRecord, 5>(a, "b.c")).toEqual(b);
  });
  it("should be fine when key is not found", () => {
    const a: TestRecord = {
      a: 1,
      b: {
        c: 2,
      },
    };
    expect(deep.deleteD<TestRecord, 2>(a)).toEqual(a);
  });
  it("shouldn't cause errors when deleting nested objects that don't exist", () => {
    const a: TestRecord = {
      a: 1,
      b: {
        c: 2,
      },
    };
    expect(deep.deleteD<TestRecord, 2>(a, "b.d")).toEqual(a);
    expect(deep.deleteD<TestRecord, 2>(a, "e.f.g")).toEqual(a);
  });
});
