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

import { JSON } from "@/json";

describe("JSON primitive", () => {
  describe("detectType", () => {
    it("should detect string", () => {
      expect(JSON.detectType("hello")).toBe("string");
      expect(JSON.detectType("")).toBe("string");
    });

    it("should detect number", () => {
      expect(JSON.detectType(42)).toBe("number");
      expect(JSON.detectType(0)).toBe("number");
      expect(JSON.detectType(-1.5)).toBe("number");
    });

    it("should detect boolean", () => {
      expect(JSON.detectType(true)).toBe("boolean");
      expect(JSON.detectType(false)).toBe("boolean");
    });

    it("should detect null", () => {
      expect(JSON.detectType(null)).toBe("null");
    });
  });

  describe("primitiveZ", () => {
    it("should accept strings", () => {
      expect(JSON.primitiveZ.parse("ok")).toBe("ok");
    });

    it("should accept numbers", () => {
      expect(JSON.primitiveZ.parse(42)).toBe(42);
    });

    it("should accept booleans", () => {
      expect(JSON.primitiveZ.parse(true)).toBe(true);
    });

    it("should accept null", () => {
      expect(JSON.primitiveZ.parse(null)).toBeNull();
    });

    it("should reject objects", () => {
      expect(() => JSON.primitiveZ.parse({})).toThrow(z.ZodError);
    });

    it("should reject arrays", () => {
      expect(() => JSON.primitiveZ.parse([])).toThrow(z.ZodError);
    });
  });
});
