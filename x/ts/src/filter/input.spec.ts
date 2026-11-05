// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { z } from "zod";

import { filter } from "@/filter";

describe("filter input", () => {
  describe("setInputZ", () => {
    const keyZ = filter.setInputZ(z.number());

    it("should read a value as eq", () => {
      expect(keyZ.parse(12)).toEqual({ eq: 12 });
    });

    it("should read a list as in", () => {
      expect(keyZ.parse([1, 2])).toEqual({ in: [1, 2] });
    });

    it("should keep an operator", () => {
      expect(keyZ.parse({ in: [1], eq: 1 })).toEqual({ in: [1], eq: 1 });
    });

    it("should reject a value of the wrong type", () => {
      expect(() => keyZ.parse("12")).toThrow();
    });

    it("should reject an unknown operator", () => {
      expect(() => keyZ.parse({ gt: 1 })).toThrow();
    });
  });

  describe("orderedInputZ", () => {
    const createdZ = filter.orderedInputZ(z.number());

    it("should read a value as eq", () => {
      expect(createdZ.parse(3)).toEqual({ eq: 3 });
    });

    it("should read a list as in", () => {
      expect(createdZ.parse([3, 4])).toEqual({ in: [3, 4] });
    });

    it("should keep an operator", () => {
      expect(createdZ.parse({ gt: 1, lte: 9 })).toEqual({ gt: 1, lte: 9 });
    });
  });

  describe("stringInputZ", () => {
    it("should read a value as eq", () => {
      expect(filter.stringInputZ.parse("a")).toEqual({ eq: "a" });
    });

    it("should read a list as in", () => {
      expect(filter.stringInputZ.parse(["a", "b"])).toEqual({ in: ["a", "b"] });
    });

    it("should keep an operator", () => {
      expect(
        filter.stringInputZ.parse({ startsWith: "Hot", caseInsensitive: true }),
      ).toEqual({ startsWith: "Hot", caseInsensitive: true });
    });
  });

  describe("boolInputZ", () => {
    it("should read a value as eq", () => {
      expect(filter.boolInputZ.parse(false)).toEqual({ eq: false });
    });

    it("should keep an operator", () => {
      expect(filter.boolInputZ.parse({ eq: true })).toEqual({ eq: true });
    });

    it("should reject a list", () => {
      expect(() => filter.boolInputZ.parse([true])).toThrow();
    });
  });

  describe("containsInputZ", () => {
    const integrationsZ = filter.containsInputZ(z.string());

    it("should read a value as any of one", () => {
      expect(integrationsZ.parse("ni")).toEqual({ any: ["ni"] });
    });

    it("should read a list as any", () => {
      expect(integrationsZ.parse(["ni", "labjack"])).toEqual({
        any: ["ni", "labjack"],
      });
    });

    it("should keep an operator", () => {
      expect(integrationsZ.parse({ any: ["ni"] })).toEqual({ any: ["ni"] });
    });
  });

  describe("listInputZ", () => {
    const labelsZ = filter.listInputZ(z.string());

    it("should read a value as a list of one", () => {
      expect(labelsZ.parse("a")).toEqual(["a"]);
    });

    it("should keep a list", () => {
      expect(labelsZ.parse(["a", "b"])).toEqual(["a", "b"]);
    });
  });

  describe("nonEmpty", () => {
    it("should return a where that sets a line", () => {
      expect(filter.nonEmpty({ key: { eq: 1 } })).toEqual({ key: { eq: 1 } });
    });

    it("should return undefined for a where that sets no line", () => {
      expect(filter.nonEmpty({})).toBeUndefined();
    });
  });
});
