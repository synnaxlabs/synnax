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
  b: {
    c?: number;
    d?: number;
  };
  c: number[];
}


describe("path", () => {
  describe("get", () => {
    it("should get a key", () => {
      const a: TestRecord = {
        a: 1,
        b: {
          c: 2,
        },
      };
      expect(deep.get(a, "b.c")).toEqual(2);
    });
    it("should get an array index", () => {
        const a: TestRecord = {
            a: 1,
            b: {
            c: 2,
            },
            c: [1, 2, 3],
        };
        expect(deep.get(a, "c.1")).toEqual(2);
    });
  });
  describe("set", () => {
    it("should set a key", () => {
      const a: TestRecord = {
        a: 1,
        b: {
          c: 2,
        },
      };
      const b: TestRecord = {
        a: 1,
        b: {
          c: 3,
        },
      };
      deep.set(a, "b.c", 3);
        expect(a).toEqual(b);
    });
    it("should set an array index", () => {
        const a: TestRecord = {
            a: 1,
            b: {
            c: 2,
            },
            c: [1, 2, 3],
        };
        const b: TestRecord = {
            a: 1,
            b: {
            c: 2,
            },
            c: [1, 4, 3],
        };
        deep.set(a, "c.1", 4);
        expect(a).toEqual(b);
   });
  });
});
