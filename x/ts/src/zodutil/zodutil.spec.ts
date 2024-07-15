// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { z } from "zod";

import { zodutil } from "@/zodutil";

describe("zodutil", () => {
  describe("getFieldShemaPath", () => {
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
        expect(zodutil.getFieldSchemaPath(path)).toBe(expected);
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
    spec.forEach(({ path }) => {
      it(`should return not null for ${path}`, () => {
        schema.shape.arrayOfObjects.element.shape.a;
        expect(zodutil.getFieldSchema(schema, path)).not.toBeNull();
      });
    });
    it("should return null for invalid path and optional is true", () => {
      expect(zodutil.getFieldSchema(schema, "a.b.c.d", { optional: true })).toBeNull();
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
        const v = zodutil.getFieldSchema(schema, "a.b.c");
        expect(zodutil.getFieldSchema(schema, "a.b.c")).not.toBeNull();
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
        const v = zodutil.getFieldSchema(veryComplexSchema, "a.array.0.c");
        expect(v).not.toBeNull();
        expect(v).toBeInstanceOf(z.ZodNumber);
      });
    });
  });
});
