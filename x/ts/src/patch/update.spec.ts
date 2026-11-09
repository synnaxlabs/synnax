// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { readFileSync } from "node:fs";

import { describe, expect, it } from "vitest";
import { z } from "zod";

import { errors } from "@/errors";
import { patch } from "@/patch";

// The vectors are shared with the Go implementation so that both runtimes apply an
// update alike. The canonical copy lives in x/testdata/update.
const VECTORS_URL = new URL("../../../testdata/update/vectors.json", import.meta.url);

const aiZ = z.object({
  key: z.string(),
  type: z.literal("ai"),
  port: z.number(),
  range: z.number().default(10),
  units: z.string().default("V"),
});
const diZ = z.object({
  key: z.string(),
  type: z.literal("di"),
  port: z.number(),
  line: z.number(),
});
const channelZ = z.discriminatedUnion("type", [aiZ, diZ]);
const configZ = z.object({ rate: z.number(), channels: channelZ.array() });
const limitsZ = z.object({ low: z.number(), high: z.number() });
const docZ = z.object({
  name: z.string(),
  group: z.string().optional(),
  config: configZ,
  limits: limitsZ.optional(),
  tags: z.string().array(),
});
type Doc = z.infer<typeof docZ>;

const channelUpdateZ = z.discriminatedUnion("type", [
  z.strictObject({
    key: z.string(),
    type: z.literal("ai"),
    port: patch.changeZ(z.number()).optional(),
    range: patch.changeZ(z.number()).optional(),
    units: patch.changeZ(z.string()).optional(),
  }),
  z.strictObject({
    key: z.string(),
    type: z.literal("di"),
    port: patch.changeZ(z.number()).optional(),
    line: patch.changeZ(z.number()).optional(),
  }),
]);
const configUpdateZ = z.strictObject({
  rate: patch.changeZ(z.number()).optional(),
  channels: patch.listZ({ e: channelZ, k: z.string(), u: channelUpdateZ }).optional(),
});
const limitsUpdateZ = z.strictObject({
  low: patch.changeZ(z.number()).optional(),
  high: patch.changeZ(z.number()).optional(),
});
const docUpdateZ = z.strictObject({
  name: patch.changeZ(z.string()).optional(),
  group: patch.changeZ(z.string()).optional(),
  config: patch.structZ({ u: configUpdateZ, t: configZ }).optional(),
  limits: patch.changeZ(patch.structZ({ u: limitsUpdateZ, t: limitsZ })).optional(),
  tags: patch.valueListZ(z.string()).optional(),
});
const schemas = { u: docUpdateZ, t: docZ };
const updateZ = patch.updateZ(docUpdateZ);

interface Vector {
  name: string;
  doc?: Doc;
  update: unknown;
  expected?: Doc;
  error?: { kind: "stale" | "unique" | "validation"; message: string };
}

const vectors = JSON.parse(readFileSync(VECTORS_URL, "utf-8")) as {
  doc: Doc;
  cases: Vector[];
};

const expectError = (
  run: () => unknown,
  { kind, message }: Required<Vector>["error"],
) => {
  let caught: unknown;
  try {
    run();
  } catch (e) {
    caught = e;
  }
  expect(caught).toBeInstanceOf(Error);
  expect(patch.StaleError.matches(caught)).toBe(kind === "stale");
  expect((caught as Error).message).toContain(message);
};

describe("update vectors", () => {
  vectors.cases.forEach(({ name, doc = vectors.doc, update, expected, error }) => {
    const parsed = updateZ.parse(update);

    it(name, () => {
      const run = () => patch.applyUpdate(schemas, doc, parsed);
      if (error != null) return expectError(run, error);
      const { next } = run();
      expect(next).toEqual(expected);
    });

    if (expected == null) return;

    it(`undoes: ${name}`, () => {
      const { next, inverse } = patch.applyUpdate(schemas, doc, parsed);
      expect(patch.applyUpdate(schemas, next, inverse).next).toEqual(doc);
    });

    it(`diffs: ${name}`, () => {
      const diffed = patch.diffUpdate(docUpdateZ, doc, expected);
      expect(updateZ.parse(diffed)).toEqual(diffed);
      expect(patch.applyUpdate(schemas, doc, diffed).next).toEqual(expected);
    });
  });
});

describe("applyUpdate", () => {
  it("should not change the record it applies to", () => {
    const doc = structuredClone(vectors.doc);
    patch.applyUpdate(schemas, doc, {
      value: {
        name: "Cold flow",
        config: { channels: { update: [{ key: "c1", type: "ai", port: 3 }] } },
        tags: { remove: ["a"] },
      },
      guards: [],
    });
    expect(doc).toEqual(vectors.doc);
  });

  it("should keep the fields that the update does not touch", () => {
    const { next } = patch.applyUpdate(schemas, vectors.doc, {
      value: { name: "Cold flow" },
      guards: [],
    });
    expect(next.config).toBe(vectors.doc.config);
  });

  it("should round trip a StaleError through the error registry", () => {
    const decoded = errors.decode(errors.encode(new patch.StaleError("c9 not found")));
    expect(patch.StaleError.matches(decoded)).toBe(true);
    expect(decoded?.message).toBe("c9 not found");
  });
});

describe("diffUpdate", () => {
  it("should guard a union that it merges into", () => {
    const next = structuredClone(vectors.doc);
    next.config.channels[0] = { ...next.config.channels[0], port: 7 };
    expect(patch.diffUpdate(docUpdateZ, vectors.doc, next)).toEqual({
      value: { config: { channels: { update: [{ key: "c1", type: "ai", port: 7 }] } } },
      guards: [{ path: ["config", "channels", "c1", "type"], value: "ai" }],
    });
  });

  it("should take every variant field of a union that changes variant", () => {
    const next = structuredClone(vectors.doc);
    next.config.channels[1] = { key: "c2", type: "ai", port: 1, range: 5, units: "V" };
    expect(patch.diffUpdate(docUpdateZ, vectors.doc, next)).toEqual({
      value: {
        config: {
          channels: {
            update: [{ key: "c2", type: "ai", range: 5, units: "V" }],
          },
        },
      },
      guards: [],
    });
  });

  it("should move the fewest entries", () => {
    const base = { ...vectors.doc, tags: ["a", "b", "c", "d", "e"] };
    expect(
      patch.diffUpdate(docUpdateZ, base, { ...base, tags: ["b", "c", "d", "e", "a"] }),
    ).toEqual({ value: { tags: { move: [{ key: "a", after: "e" }] } }, guards: [] });
  });

  it("should replace an optional struct that base does not hold", () => {
    const { limits: _, ...base } = vectors.doc;
    expect(patch.diffUpdate(docUpdateZ, base, vectors.doc)).toEqual({
      value: { limits: { set: { low: 0, high: 100 } } },
      guards: [],
    });
  });

  it("should throw for a field that the schema does not hold", () => {
    expect(() =>
      patch.diffUpdate(docUpdateZ, vectors.doc, { ...vectors.doc, extra: 1 } as Doc),
    ).toThrow("The update schema holds no field extra");
  });
});

describe("nest", () => {
  it("should nest a leaf into the structs on its path", () => {
    expect(patch.nest(docUpdateZ, vectors.doc, ["config", "rate"], 5)).toEqual({
      config: { rate: 5 },
    });
  });

  it("should nest into a list entry by key, with the variant that doc holds", () => {
    expect(
      patch.nest(docUpdateZ, vectors.doc, ["config", "channels", "c2", "line"], 9),
    ).toEqual({
      config: { channels: { update: [{ key: "c2", type: "di", line: 9 }] } },
    });
  });

  it("should nest a partial entry at the key of an entry", () => {
    expect(
      patch.nest(docUpdateZ, vectors.doc, ["config", "channels", "c1"], { port: 2 }),
    ).toEqual({
      config: { channels: { update: [{ key: "c1", type: "ai", port: 2 }] } },
    });
  });

  it("should nest into an optional struct", () => {
    expect(patch.nest(docUpdateZ, vectors.doc, ["limits", "low"], -5)).toEqual({
      limits: { low: -5 },
    });
  });

  it("should give an update that applies", () => {
    const value = patch.nest(
      docUpdateZ,
      vectors.doc,
      ["config", "channels", "c1", "port"],
      3,
    );
    const { next } = patch.applyUpdate(schemas, vectors.doc, { value, guards: [] });
    expect(next.config.channels[0]).toEqual({
      ...vectors.doc.config.channels[0],
      port: 3,
    });
  });

  it("should throw for a path that the schema does not hold", () => {
    expect(() => patch.nest(docUpdateZ, vectors.doc, ["config", "speed"], 1)).toThrow(
      "The update schema holds no field speed",
    );
    expect(() => patch.nest(docUpdateZ, vectors.doc, ["name", "first"], 1)).toThrow(
      "[name] holds no fields",
    );
  });

  it("should throw for a union that doc holds no variant of", () => {
    expect(() =>
      patch.nest(docUpdateZ, vectors.doc, ["config", "channels", "c9", "port"], 1),
    ).toThrow("type undefined names no variant");
  });
});
