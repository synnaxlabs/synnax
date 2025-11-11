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

import { createPortValidator, createSimplePortValidator } from "./validation";

describe("validation", () => {
  describe("createPortValidator", () => {
    it("should allow channels with different ports on the same device", () => {
      const validator = createPortValidator("Test");
      const schema = z
        .array(z.object({ port: z.number(), device: z.string() }))
        .check(validator);

      const result = schema.safeParse([
        { port: 0, device: "dev1" },
        { port: 1, device: "dev1" },
      ]);

      expect(result.success).toBe(true);
    });

    it("should allow the same port on different devices", () => {
      const validator = createPortValidator("Test");
      const schema = z
        .array(z.object({ port: z.number(), device: z.string() }))
        .check(validator);

      const result = schema.safeParse([
        { port: 0, device: "dev1" },
        { port: 0, device: "dev2" },
      ]);

      expect(result.success).toBe(true);
    });

    it("should reject duplicate ports on the same device", () => {
      const validator = createPortValidator("Counter");
      const schema = z
        .array(z.object({ port: z.number(), device: z.string() }))
        .check(validator);

      const result = schema.safeParse([
        { port: 0, device: "dev1" },
        { port: 0, device: "dev1" },
      ]);

      expect(result.success).toBe(false);
      if (!result.success) {
        expect(result.error.issues).toHaveLength(2);
        expect(result.error.issues[0].path).toEqual([0, "port"]);
        expect(result.error.issues[1].path).toEqual([1, "port"]);
        expect(result.error.issues[0].message).toContain("Counter port 0");
        expect(result.error.issues[0].message).toContain(
          "already been used on another channel on the same device",
        );
      }
    });

    it("should use the provided port type label in error messages", () => {
      const validator = createPortValidator("Analog");
      const schema = z
        .array(z.object({ port: z.number(), device: z.string() }))
        .check(validator);

      const result = schema.safeParse([
        { port: 5, device: "dev1" },
        { port: 5, device: "dev1" },
      ]);

      expect(result.success).toBe(false);
      if (!result.success) 
        expect(result.error.issues[0].message).toContain("Analog port 5");
      
    });

    it("should handle missing port type label", () => {
      const validator = createPortValidator();
      const schema = z
        .array(z.object({ port: z.number(), device: z.string() }))
        .check(validator);

      const result = schema.safeParse([
        { port: 0, device: "dev1" },
        { port: 0, device: "dev1" },
      ]);

      expect(result.success).toBe(false);
      if (!result.success) {
        expect(result.error.issues[0].message).toContain("Port 0");
        expect(result.error.issues[0].message).not.toContain("  "); // No double space
        expect(result.error.issues[0].message).toBe(
          "Port 0 has already been used on another channel on the same device",
        );
      }
    });

    it("should handle multiple duplicate ports", () => {
      const validator = createPortValidator("Digital");
      const schema = z
        .array(z.object({ port: z.number(), device: z.string() }))
        .check(validator);

      const result = schema.safeParse([
        { port: 0, device: "dev1" },
        { port: 0, device: "dev1" },
        { port: 1, device: "dev1" },
        { port: 1, device: "dev1" },
      ]);

      expect(result.success).toBe(false);
      if (!result.success) 
        // Should have 4 errors (2 for port 0, 2 for port 1)
        expect(result.error.issues).toHaveLength(4);
      
    });
  });

  describe("createSimplePortValidator", () => {
    it("should allow channels with different ports", () => {
      const validator = createSimplePortValidator("Test");
      const schema = z.array(z.object({ port: z.number() })).check(validator);

      const result = schema.safeParse([{ port: 0 }, { port: 1 }, { port: 2 }]);

      expect(result.success).toBe(true);
    });

    it("should reject duplicate ports", () => {
      const validator = createSimplePortValidator("Counter");
      const schema = z.array(z.object({ port: z.number() })).check(validator);

      const result = schema.safeParse([{ port: 0 }, { port: 0 }]);

      expect(result.success).toBe(false);
      if (!result.success) {
        expect(result.error.issues).toHaveLength(2);
        expect(result.error.issues[0].path).toEqual([0, "port"]);
        expect(result.error.issues[1].path).toEqual([1, "port"]);
        expect(result.error.issues[0].message).toContain("Counter port 0");
        expect(result.error.issues[0].message).toContain(
          "already been used on another channel",
        );
      }
    });

    it("should use the provided port type label in error messages", () => {
      const validator = createSimplePortValidator("Test Port");
      const schema = z.array(z.object({ port: z.number() })).check(validator);

      const result = schema.safeParse([{ port: 3 }, { port: 3 }]);

      expect(result.success).toBe(false);
      if (!result.success) 
        expect(result.error.issues[0].message).toContain("Test Port port 3");
      
    });

    it("should handle multiple channels with mixed duplicates", () => {
      const validator = createSimplePortValidator("Test");
      const schema = z.array(z.object({ port: z.number() })).check(validator);

      const result = schema.safeParse([
        { port: 0 },
        { port: 1 },
        { port: 0 }, // Duplicate of index 0
        { port: 2 },
      ]);

      expect(result.success).toBe(false);
      if (!result.success) {
        expect(result.error.issues).toHaveLength(2);
        expect(result.error.issues[0].path).toEqual([0, "port"]);
        expect(result.error.issues[1].path).toEqual([2, "port"]);
      }
    });

    it("should handle empty arrays", () => {
      const validator = createSimplePortValidator("Test");
      const schema = z.array(z.object({ port: z.number() })).check(validator);

      const result = schema.safeParse([]);

      expect(result.success).toBe(true);
    });

    it("should handle missing port type label", () => {
      const validator = createSimplePortValidator();
      const schema = z.array(z.object({ port: z.number() })).check(validator);

      const result = schema.safeParse([{ port: 2 }, { port: 2 }]);

      expect(result.success).toBe(false);
      if (!result.success) {
        expect(result.error.issues[0].message).toContain("Port 2");
        expect(result.error.issues[0].message).not.toContain("  "); // No double space
        expect(result.error.issues[0].message).toBe(
          "Port 2 has already been used on another channel",
        );
      }
    });
  });
});
