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

import { detectType, primitiveZ } from "@/json/primitive";

describe("JSON primitive", () => {
  describe("detectType", () => {
    it("should detect string", () => {
      expect(detectType("hello")).toBe("string");
      expect(detectType("")).toBe("string");
    });

    it("should detect number", () => {
      expect(detectType(42)).toBe("number");
      expect(detectType(0)).toBe("number");
      expect(detectType(-1.5)).toBe("number");
    });

    it("should detect boolean", () => {
      expect(detectType(true)).toBe("boolean");
      expect(detectType(false)).toBe("boolean");
    });

    it("should detect null", () => {
      expect(detectType(null)).toBe("null");
    });
  });

  describe("primitiveZ", () => {
    it("should accept strings", () => {
      expect(primitiveZ.parse("ok")).toBe("ok");
    });

    it("should accept numbers", () => {
      expect(primitiveZ.parse(42)).toBe(42);
    });

    it("should accept booleans", () => {
      expect(primitiveZ.parse(true)).toBe(true);
    });

    it("should accept null", () => {
      expect(primitiveZ.parse(null)).toBeNull();
    });

    it("should reject objects", () => {
      expect(() => primitiveZ.parse({})).toThrow(z.ZodError);
    });

    it("should reject arrays", () => {
      expect(() => primitiveZ.parse([])).toThrow(z.ZodError);
    });
  });
});
