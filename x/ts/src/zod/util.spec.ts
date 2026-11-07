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
        m: z.preprocess((v) => v, z.object({ n: z.number() })),
      });
      const spec = [
        ["a.b", z.ZodNumber],
        ["c.d", z.ZodString],
        ["e.f", z.ZodBoolean],
        ["g.h", z.ZodNumber],
        ["i.j", z.ZodNumber],
        ["k.l", z.ZodString],
        ["m.n", z.ZodNumber],
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

    describe("plain unions", () => {
      const schema = z.object({
        props: z.union([
          z.object({ version: z.literal(1), count: z.number() }),
          z.object({ version: z.literal(0), name: z.string() }).transform((v) => v),
        ]),
      });
      it("should serve a path from the first member that contains it", () => {
        expect(zod.getFieldSchema(schema, "props.count")).toBeInstanceOf(z.ZodNumber);
        expect(zod.getFieldSchema(schema, "props.name")).toBeInstanceOf(z.ZodString);
      });
      it("should not contain a path no member holds", () => {
        expect(
          zod.getFieldSchema(schema, "props.other", { optional: true }),
        ).toBeNull();
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
      describe("of discriminated unions", () => {
        const schema = z.object({
          channels: z.array(
            z.discriminatedUnion("type", [
              z.object({ key: z.string(), type: z.literal("a"), value: z.number() }),
              z.object({ key: z.string(), type: z.literal("b"), value: z.string() }),
            ]),
          ),
        });
        const values = {
          channels: [
            { key: "1", type: "b", value: "x" },
            { key: "0", type: "a", value: 1 },
          ],
        };
        it("should read the variant of the entry a key names before its index", () => {
          expect(
            zod.getFieldSchema(schema, "channels.0.value", { values }),
          ).toBeInstanceOf(z.ZodNumber);
        });
        it("should read the variant of the entry at an index no key names", () => {
          const values = { channels: [{ key: "x", type: "b", value: "x" }] };
          expect(
            zod.getFieldSchema(schema, "channels.0.value", { values }),
          ).toBeInstanceOf(z.ZodString);
        });
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

    describe("catchall", () => {
      const schema = z.object({ a: z.string() }).catchall(z.number());
      it("should serve a field the shape does not name", () => {
        expect(zod.getFieldSchema(schema, "other")).toBeInstanceOf(z.ZodNumber);
      });
    });

    describe("tuples", () => {
      const schema = z.object({
        t: z.tuple([z.string(), z.number()]),
        r: z.tuple([z.string()], z.boolean()),
      });
      it("should map an index to its item", () => {
        expect(zod.getFieldSchema(schema, "t.1")).toBeInstanceOf(z.ZodNumber);
      });
      it("should map an index past the items to the rest", () => {
        expect(zod.getFieldSchema(schema, "r.4")).toBeInstanceOf(z.ZodBoolean);
      });
      it("should not contain an index past the items with no rest", () => {
        expect(zod.getFieldSchema(schema, "t.2", { optional: true })).toBeNull();
      });
      it("should not contain a segment that is not an index", () => {
        expect(zod.getFieldSchema(schema, "t.x", { optional: true })).toBeNull();
      });
    });

    describe("dotted keys", () => {
      const schema = z.object({ "a.b": z.object({ c: z.number() }) });
      it("should match the longest key", () => {
        expect(zod.getFieldSchema(schema, "a.b.c")).toBeInstanceOf(z.ZodNumber);
      });
      it("should read the values under the longest key", () => {
        const schema = z.object({
          "a.b": z.discriminatedUnion("type", [
            z.object({ type: z.literal("x"), v: z.number() }),
            z.object({ type: z.literal("y"), v: z.string() }),
          ]),
        });
        const values = { "a.b": { type: "y", v: "s" } };
        expect(zod.getFieldSchema(schema, "a.b.v", { values })).toBeInstanceOf(
          z.ZodString,
        );
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

  describe("getDiscriminators", () => {
    const channelZ = z.discriminatedUnion("type", [
      z.object({ key: z.string(), type: z.literal("ai_voltage"), maxVal: z.number() }),
      z.object({
        key: z.string(),
        type: z.literal("ai_thermocouple"),
        units: z.string(),
        scale: z
          .discriminatedUnion("kind", [
            z.object({ kind: z.literal("linear"), slope: z.number() }),
            z.object({ kind: z.literal("none") }),
          ])
          .optional(),
      }),
    ]);
    const schema = z.object({
      name: z.string(),
      config: z.object({ channels: z.array(channelZ), extra: z.unknown() }),
    });
    const values = {
      name: "task",
      config: {
        channels: [
          { key: "a", type: "ai_voltage", maxVal: 10 },
          {
            key: "b",
            type: "ai_thermocouple",
            units: "C",
            scale: { kind: "linear", slope: 2 },
          },
        ],
        extra: { anything: { goes: 1 } },
      },
    };

    it("should list nothing for a path that crosses no union", () => {
      expect(zod.getDiscriminators(schema, ["name"], values)).toEqual([]);
    });

    it("should list the union a path goes into", () => {
      expect(
        zod.getDiscriminators(schema, ["config", "channels", "a", "maxVal"], values),
      ).toEqual([{ path: ["config", "channels", "a", "type"], value: "ai_voltage" }]);
    });

    it("should not list a union that the path ends at", () => {
      expect(
        zod.getDiscriminators(schema, ["config", "channels", "a"], values),
      ).toEqual([]);
    });

    it("should list nested unions from the root down", () => {
      expect(
        zod.getDiscriminators(
          schema,
          ["config", "channels", "b", "scale", "slope"],
          values,
        ),
      ).toEqual([
        { path: ["config", "channels", "b", "type"], value: "ai_thermocouple" },
        { path: ["config", "channels", "b", "scale", "kind"], value: "linear" },
      ]);
    });

    it("should list nothing below an unknown schema", () => {
      expect(
        zod.getDiscriminators(schema, ["config", "extra", "anything", "goes"], values),
      ).toEqual([]);
    });

    it("should throw for a field of a variant the values do not hold", () => {
      expect(() =>
        zod.getDiscriminators(schema, ["config", "channels", "a", "units"], values),
      ).toThrow("Schema does not contain the path config.channels.a.units");
    });

    it("should throw for a path through a missing list entry", () => {
      expect(() =>
        zod.getDiscriminators(schema, ["config", "channels", "x", "maxVal"], values),
      ).toThrow("Schema does not contain the path config.channels.x.maxVal");
    });

    it("should find a list entry by index when no key matches", () => {
      expect(
        zod.getDiscriminators(schema, ["config", "channels", "1", "units"], values),
      ).toEqual([
        { path: ["config", "channels", "1", "type"], value: "ai_thermocouple" },
      ]);
    });

    describe("containers", () => {
      const voltage = { key: "v", type: "ai_voltage", maxVal: 10 };
      const schema = z.object({
        pair: z.tuple([z.string(), channelZ]),
        byName: z.record(z.string(), channelZ),
        loose: z.object({}).catchall(channelZ),
        "a.b": channelZ,
        list: z.array(channelZ),
      });
      const values = {
        pair: ["x", voltage],
        byName: { v: voltage },
        loose: { v: voltage },
        "a.b": voltage,
        list: [
          { ...voltage, key: "1" },
          { key: "0", type: "ai_thermocouple", units: "C" },
        ],
      };

      it.each([
        ["a tuple item", ["pair", "1", "maxVal"]],
        ["a record value", ["byName", "v", "maxVal"]],
        ["a catchall field", ["loose", "v", "maxVal"]],
        ["a field whose key holds the separator", ["a.b", "maxVal"]],
      ])("should list the union in %s", (_, path) => {
        expect(zod.getDiscriminators(schema, path, values)).toEqual([
          { path: [...path.slice(0, -1), "type"], value: "ai_voltage" },
        ]);
      });

      it("should find a list entry by key before its index", () => {
        expect(zod.getDiscriminators(schema, ["list", "0", "units"], values)).toEqual([
          { path: ["list", "0", "type"], value: "ai_thermocouple" },
        ]);
      });
    });

    it("should not list a union from a plain union member that lacks the path", () => {
      const schema = z.object({
        u: z.union([channelZ, z.object({ units: z.string(), extra: z.number() })]),
      });
      const values = { u: { key: "a", type: "ai_voltage", units: "V", extra: 1 } };
      expect(zod.getDiscriminators(schema, ["u", "extra"], values)).toEqual([]);
    });
  });
});
