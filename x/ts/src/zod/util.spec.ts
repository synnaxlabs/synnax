// Copyright 2026 Synnax Labs, Inc.
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
    describe("wrappers", () => {
      const schema = z.object({
        a: z.object({ b: z.number() }).optional(),
        c: z.object({ d: z.string() }).default({ d: "x" }),
        e: z.object({ f: z.boolean() }).prefault({ f: true }),
        g: z.object({ h: z.number() }).nullable(),
        i: z.object({ j: z.number() }).transform((v) => v.j),
        k: z.lazy(() => z.object({ l: z.string() })),
      });
      const spec = [
        ["a.b", z.ZodNumber],
        ["c.d", z.ZodString],
        ["e.f", z.ZodBoolean],
        ["g.h", z.ZodNumber],
        ["i.j", z.ZodNumber],
        ["k.l", z.ZodString],
      ] as const;
      spec.forEach(([path, type]) =>
        it(`should descend through the wrapper on ${path}`, () =>
          expect(zod.getFieldSchema(schema, path)).toBeInstanceOf(type)),
      );
    });

    describe("discriminated unions", () => {
      const schema = z.object({
        config: z.discriminatedUnion("type", [
          z.object({ type: z.literal("a"), value: z.number() }),
          z.object({ type: z.literal("b"), value: z.string().optional() }),
        ]),
      });
      it("should select the member the values name", () => {
        const values = { config: { type: "b", value: "x" } };
        const v = zod.getFieldSchema(schema, "config.value", { values });
        expect(v).toBeInstanceOf(z.ZodOptional);
        expect(z.validate(v, undefined)).toBe(true);
      });
      it("should select the member at the root of the schema", () => {
        const root = z.discriminatedUnion("variant", [
          z.object({ variant: z.literal("x"), count: z.number() }),
          z.object({ variant: z.literal("y"), name: z.string() }),
        ]);
        const v = zod.getFieldSchema(root, "count", { values: { variant: "x" } });
        expect(v).toBeInstanceOf(z.ZodNumber);
      });
      it("should not contain a path of a member the values do not name", () => {
        const values = { config: { type: "a" } };
        expect(() => zod.getFieldSchema(schema, "config.value.x", { values })).toThrow(
          "Schema does not contain the path config.value.x",
        );
      });
      it("should not contain any path when the discriminator is absent", () => {
        expect(
          zod.getFieldSchema(schema, "config.value", { optional: true }),
        ).toBeNull();
        expect(() => zod.getFieldSchema(schema, "config.value")).toThrow(
          "Schema does not contain the path config.value",
        );
      });
    });

    describe("keyed arrays", () => {
      const schema = z.object({
        channels: z.array(z.object({ key: z.string(), port: z.number() })),
      });
      it("should map a key segment to the element", () => {
        expect(zod.getFieldSchema(schema, "channels.ch-1.port")).toBeInstanceOf(
          z.ZodNumber,
        );
      });
      it("should map an index segment to the element", () => {
        expect(zod.getFieldSchema(schema, "channels.0.port")).toBeInstanceOf(
          z.ZodNumber,
        );
      });
    });

    describe("records and delegation", () => {
      const schema = z.object({
        props: z.record(z.string(), z.object({ v: z.number() })),
        extra: z.unknown(),
        loose: z.any(),
      });
      it("should map a record segment to the value type", () => {
        expect(zod.getFieldSchema(schema, "props.any-key.v")).toBeInstanceOf(
          z.ZodNumber,
        );
      });
      it("should stop at an unknown and report it as optional", () => {
        const v = zod.getFieldSchema(schema, "extra.deep.path");
        expect(v).toBeInstanceOf(z.ZodUnknown);
        expect(z.validate(v, undefined)).toBe(true);
      });
      it("should stop at an any", () => {
        expect(zod.getFieldSchema(schema, "loose.x")).toBeInstanceOf(z.ZodAny);
      });
    });

    describe("dotted keys", () => {
      const schema = z.object({ "a.b": z.object({ c: z.number() }) });
      it("should match the longest key", () => {
        expect(zod.getFieldSchema(schema, "a.b.c")).toBeInstanceOf(z.ZodNumber);
      });
    });

    it("should throw for a path the schema does not contain", () => {
      const schema = z.object({ a: z.number() });
      expect(() => zod.getFieldSchema(schema, "b")).toThrow(
        "Schema does not contain the path b",
      );
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
        expect(z.validate(v, undefined)).toBe(false);
      });
    });
  });
});
