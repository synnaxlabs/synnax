// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { omit } from "@/omit";

type Object = { [key: string]: number };

describe("omit", () => {
  it("should return the object if no keys are provided", () =>
    expect(omit({ a: 1, b: 2, c: 3 })).toEqual({ a: 1, b: 2, c: 3 }));

  it("should return the object for a single key", () =>
    expect(omit({ a: 1, b: 2, c: 3 }, "a")).toEqual({ b: 2, c: 3 }));

  it("should return the object for multiple keys", () =>
    expect(omit({ a: 1, b: 2, c: 3 }, "a", "b")).toEqual({ c: 3 }));

  it("should not mutate the original object", () => {
    const obj = { a: 1, b: 2, c: 3 };
    const result = omit(obj, "a", "b");
    expect(result).toEqual({ c: 3 });
    expect(obj).toEqual({ a: 1, b: 2, c: 3 });
  });

  it("should be a no-op if the keys are not present", () => {
    const obj: Object = { a: 1, b: 2, c: 3 };
    const result = omit(obj, "d", "e");
    expect(result).toEqual({ a: 1, b: 2, c: 3 });
  });

  it("should handle empty objects", () => {
    const obj: Object = {};
    const result = omit(obj, "a", "b", "c");
    expect(result).toEqual({});
  });
});
