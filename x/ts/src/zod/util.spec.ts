// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, test } from "vitest";
import { z } from "zod";

import { zod } from "@/zod";

describe("zod", () => {
  describe("functionOutput", () => {
    it("should return ZodUnknown for z.function() with no output", () => {
      expect(zod.functionOutput(z.function())).toBeInstanceOf(z.ZodUnknown);
    });

    it("should return ZodVoid for z.function({ output: z.void() })", () => {
      expect(zod.functionOutput(z.function({ output: z.void() }))).toBeInstanceOf(
        z.ZodVoid,
      );
    });

    it("should return ZodNumber for z.function({ output: z.number() })", () => {
      expect(zod.functionOutput(z.function({ output: z.number() }))).toBeInstanceOf(
        z.ZodNumber,
      );
    });

    it("should return ZodString for z.function({ output: z.string() })", () => {
      expect(zod.functionOutput(z.function({ output: z.string() }))).toBeInstanceOf(
        z.ZodString,
      );
    });

    it("should return ZodPromise for z.function({ output: z.promise(z.number()) })", () => {
      expect(
        zod.functionOutput(z.function({ output: z.promise(z.number()) })),
      ).toBeInstanceOf(z.ZodPromise);
    });
  });

  describe("getFieldSchemaPath", () => {
    interface Spec {
      path: string;
      expected: string;
    }
    const spec: Spec[] = [
      { path: "a.b.c", expected: "a.shape.b.shape.c" },
      { path: "a.0.c", expected: "a.element.shape.c" },
      { path: "a.0.1", expected: "a.element.element" },
      { path: "a.0.1.2", expected: "a.element.element.element" },
    ];
    spec.forEach(({ path, expected }) => {
      it(`should return ${expected} for ${path}`, () => {
        expect(zod.getFieldSchemaPath(path)).toBe(expected);
      });
    });
  });
  describe("getFieldSchema", () => {
    const schema = z.object({
      a: z.object({
        b: z.object({
          c: z.number(),
        }),
      }),
      array: z.array(z.array(z.array(z.number()))),
      arrayOfObjects: z.array(
        z.object({
          a: z.number(),
          b: z.string(),
        }),
      ),
    });
    interface Spec {
      path: string;
    }
    // just assert not null
    const spec: Spec[] = [
      { path: "a.b.c" },
      { path: "array.0.0.0" },
      { path: "arrayOfObjects.0.a" },
    ];
    spec.forEach(({ path }) =>
      it(`should return not null for ${path}`, () =>
        expect(zod.getFieldSchema(schema, path)).not.toBeNull()),
    );
    it("should return null for invalid path and optional is true", () => {
      expect(zod.getFieldSchema(schema, "a.b.c.d", { optional: true })).toBeNull();
    });
    describe("with a refinement", () => {
      const schema = z.object({
        a: z
          .object({
            b: z.object({
              c: z.number(),
            }),
          })
          .refine(() => true),
      });
      it("should return not null for valid path", () => {
        const v = zod.getFieldSchema(schema, "a.b.c");
        expect(zod.getFieldSchema(schema, "a.b.c")).not.toBeNull();
        expect(v).toBeInstanceOf(z.ZodNumber);
      });
      const veryComplexSchema = z.object({
        a: z
          .object({
            array: z
              .array(
                z
                  .object({
                    c: z.number(),
                  })
                  .refine(() => true),
              )
              .refine(() => true),
          })
          .refine(() => true),
      });
      it("should return not null for valid path", () => {
        const v = zod.getFieldSchema(veryComplexSchema, "a.array.0.c");
        expect(v).not.toBeNull();
        expect(v).toBeInstanceOf(z.ZodNumber);
      });
    });

    describe("regression", () => {
      test("reg 1", () => {
        const names = ["one"];
        const schema = z
          .object({
            name: z.string(),
          })
          .refine(({ name }) => !names.includes(name), {
            error: "Already in use",
          });
        const v = zod.getFieldSchema(schema, "name");
        expect(v).toBeInstanceOf(z.ZodString);
        expect(v.safeParse(undefined).success).toBe(false);
      });
    });
  });
});
