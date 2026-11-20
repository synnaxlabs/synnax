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

describe("path utilities", () => {
  describe("transformPath", () => {
    it("should transform a path", () => {
      expect(deep.transformPath("a.b.c", (part) => part.toUpperCase())).toEqual(
        "A.B.C",
      );
    });
    it("should inject additional parts into the path", () => {
      expect(deep.transformPath("a.b.c", (p) => [p, "d"])).toEqual("a.d.b.d.c.d");
    });
    it("should remove parts from the path", () => {
      expect(deep.transformPath("a.b.c", (p, i) => (i === 1 ? undefined : p))).toEqual(
        "a.c",
      );
    });
  });

  describe("pathsMatch", () => {
    it("should return true if two paths are equal", () => {
      expect(deep.pathsMatch("a.b.c", "a.b.c")).toEqual(true);
    });
    it("should return true if the pattern is a prefix of the path", () => {
      expect(deep.pathsMatch("a.b.c", "a.b")).toEqual(true);
    });
    it("should return true if the pattern has a wildcard", () => {
      expect(deep.pathsMatch("a.b.c", "a.*.c")).toEqual(true);
    });
    it("should return true for an empty pattern", () => {
      expect(deep.pathsMatch("a.b.c", "")).toEqual(true);
    });
    it("should return false if pattern is longer than path", () => {
      expect(deep.pathsMatch("a.b", "a.b.c")).toEqual(false);
    });
    it("should return false if paths don't match", () => {
      expect(deep.pathsMatch("a.b.c", "a.b.d")).toEqual(false);
    });
  });

  describe("element", () => {
    it("should get element at index", () => {
      expect(deep.element("a.b.c", 1)).toEqual("b");
    });

    it("should handle negative index", () => {
      expect(deep.element("a.b.c", -1)).toEqual("c");
      expect(deep.element("a.b.c", -2)).toEqual("b");
    });

    it("should get first element", () => {
      expect(deep.element("a.b.c", 0)).toEqual("a");
    });

    it("should throw for an index past the end", () => {
      expect(() => deep.element("a.b.c", 3)).toThrow("no part at index 3");
      expect(() => deep.element("a.b.c", -4)).toThrow("no part at index -4");
    });
  });
});
