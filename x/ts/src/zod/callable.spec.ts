// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { z } from "zod";

import { callable, isVoid } from "@/zod/callable";

describe("callable", () => {
  describe("callable()", () => {
    it("should create a callable with void args and void returns by default", () => {
      const c = callable();
      expect(isVoid(c.args)).toBe(true);
      expect(isVoid(c.returns)).toBe(true);
    });
  });

  describe("callable().args()", () => {
    it("should create a callable with specified args and void returns", () => {
      const c = callable().args(z.string());
      expect(c.args).toBeInstanceOf(z.ZodString);
      expect(isVoid(c.returns)).toBe(true);
    });

    it("should work with complex arg types", () => {
      const c = callable().args(z.object({ x: z.number(), y: z.number() }));
      expect(c.args).toBeInstanceOf(z.ZodObject);
      expect(isVoid(c.returns)).toBe(true);
    });
  });

  describe("callable().returns()", () => {
    it("should create a callable with void args and specified returns", () => {
      const c = callable().returns(z.number());
      expect(isVoid(c.args)).toBe(true);
      expect(c.returns).toBeInstanceOf(z.ZodNumber);
    });

    it("should work with complex return types", () => {
      const c = callable().returns(z.array(z.string()));
      expect(isVoid(c.args)).toBe(true);
      expect(c.returns).toBeInstanceOf(z.ZodArray);
    });
  });

  describe("callable().args().returns()", () => {
    it("should create a callable with specified args and returns", () => {
      const c = callable().args(z.string()).returns(z.number());
      expect(c.args).toBeInstanceOf(z.ZodString);
      expect(c.returns).toBeInstanceOf(z.ZodNumber);
    });

    it("should work with complex types", () => {
      const c = callable()
        .args(z.object({ name: z.string() }))
        .returns(z.object({ id: z.number() }));
      expect(c.args).toBeInstanceOf(z.ZodObject);
      expect(c.returns).toBeInstanceOf(z.ZodObject);
    });
  });

  describe("isVoid", () => {
    it("should return true for void schemas", () => {
      expect(isVoid(z.void())).toBe(true);
      expect(isVoid(callable().args)).toBe(true);
      expect(isVoid(callable().returns)).toBe(true);
    });

    it("should return false for non-void schemas", () => {
      expect(isVoid(z.string())).toBe(false);
      expect(isVoid(z.number())).toBe(false);
      expect(isVoid(callable().returns(z.number()).returns)).toBe(false);
    });
  });

  describe("type inference", () => {
    it("should correctly infer types for use in method schemas", () => {
      const methodsSchema = {
        onClick: callable(),
        setName: callable().args(z.string()),
        getValue: callable().returns(z.number()),
        calculate: callable().args(z.object({ x: z.number() })).returns(z.number()),
      };

      // Runtime checks
      expect(isVoid(methodsSchema.onClick.args)).toBe(true);
      expect(methodsSchema.setName.args).toBeInstanceOf(z.ZodString);
      expect(methodsSchema.getValue.returns).toBeInstanceOf(z.ZodNumber);
      expect(methodsSchema.calculate.args).toBeInstanceOf(z.ZodObject);
      expect(methodsSchema.calculate.returns).toBeInstanceOf(z.ZodNumber);
    });
  });
});
