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
const limitsZ = z.object({ low: z.number().default(0), high: z.number() });
const vectorCellZ = z.object({ value: z.string(), width: z.number() });
const docZ = z.object({
  name: z.string(),
  group: z.string().optional(),
  config: configZ,
  limits: limitsZ.optional(),
  tags: z.string().array(),
  cells: z.record(z.string(), vectorCellZ).optional(),
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
  cells: patch
    .mapZ({
      k: z.string(),
      v: vectorCellZ,
      u: z.strictObject({
        value: patch.changeZ(z.string()).optional(),
        width: patch.changeZ(z.number()).optional(),
      }),
    })
    .optional(),
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

// These schemas take the shapes that Oracle generates. An unmarked field sets a value,
// and a union with base fields joins a strict base update to its variants.
const scaleZ = z.discriminatedUnion("type", [
  z.object({
    type: z.literal("linear"),
    slope: z.number().default(1),
    offset: z.number().default(0),
  }),
  z.object({ type: z.literal("map"), min: z.number().default(0), max: z.number() }),
  z.object({ type: z.literal("none") }),
]);
const scaleUpdateZ = z.discriminatedUnion("type", [
  z.object({
    type: z.literal("linear"),
    slope: z.number().optional(),
    offset: z.number().optional(),
  }),
  z.object({
    type: z.literal("map"),
    min: z.number().optional(),
    max: z.number().optional(),
  }),
  z.object({ type: z.literal("none") }),
]);
const sensorBaseZ = z.object({ key: z.string(), name: z.string(), port: z.number() });
const sensorZ = z.discriminatedUnion("type", [
  sensorBaseZ.extend({
    type: z.literal("voltage"),
    maxVal: z.number().default(10),
    scale: scaleZ.prefault({ type: "none" }),
  }),
  sensorBaseZ.extend({ type: z.literal("thermocouple"), couple: z.string() }),
]);
const sensorBaseUpdateZ = z.object({
  key: z.string(),
  name: z.string().optional(),
  port: z.number().optional(),
});
const voltageUpdateZ = sensorBaseUpdateZ.extend({
  type: z.literal("voltage"),
  maxVal: z.number().optional(),
  scale: scaleUpdateZ.optional(),
});
const thermocoupleUpdateZ = sensorBaseUpdateZ.extend({
  type: z.literal("thermocouple"),
  couple: z.string().optional(),
});
const sensorUpdateZ = z.union([
  sensorBaseUpdateZ.strict(),
  z.discriminatedUnion("type", [voltageUpdateZ, z.lazy(() => thermocoupleUpdateZ)]),
]);
const taskZ = z.object({
  name: z.string(),
  props: z.record(z.string(), z.unknown()),
  scale: scaleZ,
  fallback: scaleZ.optional(),
  sensors: sensorZ.array(),
});
type Task = z.infer<typeof taskZ>;
const taskUpdateZ = z.object({
  name: z.string().optional(),
  props: z.record(z.string(), z.unknown()).optional(),
  scale: scaleUpdateZ.optional(),
  fallback: patch.changeZ(scaleUpdateZ).optional(),
  sensors: patch.listZ({ e: sensorZ, k: z.string(), u: sensorUpdateZ }).optional(),
});
const taskSchemas = { u: taskUpdateZ, t: taskZ };
const taskPayloadZ = patch.updateZ(taskUpdateZ);
const viewZ = z.discriminatedUnion("type", [
  z.object({
    key: z.string(),
    name: z.string(),
    type: z.literal("range"),
    where: z.string().optional(),
  }),
  z.object({
    key: z.string(),
    name: z.string(),
    type: z.literal("status"),
    level: z.string().default("info"),
  }),
]);
const viewUpdateZ = z.union([
  z.object({ name: z.string().optional() }).strict(),
  z.discriminatedUnion("type", [
    z.object({
      type: z.literal("range"),
      name: z.string().optional(),
      where: patch.changeZ(z.string()).optional(),
    }),
    z.object({
      type: z.literal("status"),
      name: z.string().optional(),
      level: z.string().optional(),
    }),
  ]),
]);
type View = z.infer<typeof viewZ>;
const viewSchemas = { u: viewUpdateZ, t: viewZ };
const viewPayloadZ = patch.updateZ(viewUpdateZ);
const view: View = { key: "v1", name: "Ranges", type: "range", where: "a" };

const task: Task = {
  name: "Bench",
  props: { rack: 1 },
  scale: { type: "linear", slope: 2, offset: 0 },
  fallback: { type: "map", min: 0, max: 5 },
  sensors: [
    {
      key: "s1",
      name: "Inlet",
      port: 0,
      type: "voltage",
      maxVal: 10,
      scale: { type: "linear", slope: 1, offset: 0 },
    },
    { key: "s2", name: "Outlet", port: 1, type: "thermocouple", couple: "K" },
  ],
};
const { fallback: _, ...taskWithoutFallback } = task;

const applyTask = (value: unknown, doc: Task = task): Task =>
  patch.applyUpdate(taskSchemas, doc, taskPayloadZ.parse({ value })).next;

// Skips the parse, so the engine meets an update that the schema rejects.
const applyUnparsed = (value: unknown, doc: Task = task): Task =>
  patch.applyUpdate(taskSchemas, doc, { value, guards: [] } as patch.Update<
    typeof taskUpdateZ
  >).next;

describe("generated update shapes", () => {
  describe("applyUpdate", () => {
    it("should set an unmarked scalar", () => {
      expect(applyTask({ name: "Rig" }).name).toBe("Rig");
    });

    it("should replace an unmarked record whole", () => {
      expect(applyTask({ props: { slot: 2 } }).props).toEqual({ slot: 2 });
    });

    it("should throw for null on an unmarked required field", () => {
      expect(() => applyUnparsed({ name: null })).toThrow(
        "a required field cannot be cleared",
      );
    });

    it("should throw for an unmarked object or array field", () => {
      const schemas = {
        u: z.object({
          inner: z.object({ a: z.number() }).optional(),
          list: z.number().array().optional(),
        }),
        t: z.object({ inner: z.object({ a: z.number() }), list: z.number().array() }),
      };
      const doc = { inner: { a: 1 }, list: [1] };
      expect(() =>
        patch.applyUpdate(schemas, doc, { value: { inner: { a: 2 } }, guards: [] }),
      ).toThrow("inner is not an update field");
      expect(() =>
        patch.applyUpdate(schemas, doc, { value: { list: [2] }, guards: [] }),
      ).toThrow("list is not an update field");
    });

    it("should merge into a union field of the same variant", () => {
      expect(applyTask({ scale: { type: "linear", offset: 1 } }).scale).toEqual({
        type: "linear",
        slope: 2,
        offset: 1,
      });
    });

    it("should fill the defaults of a union field that changes variant", () => {
      expect(applyTask({ scale: { type: "map", max: 3 } }).scale).toEqual({
        type: "map",
        min: 0,
        max: 3,
      });
    });

    it("should clear an optional union field with null", () => {
      expect(applyTask({ fallback: null })).toEqual(taskWithoutFallback);
    });

    it("should build an absent optional union field from its defaults", () => {
      expect(
        applyTask({ fallback: { type: "linear" } }, taskWithoutFallback).fallback,
      ).toEqual({ type: "linear", slope: 1, offset: 0 });
    });

    it("should throw for a union partial without a type on an absent field", () => {
      expect(() => applyUnparsed({ fallback: {} }, taskWithoutFallback)).toThrow(
        "type is required",
      );
    });

    it("should merge a list entry of the same variant", () => {
      const { sensors } = applyTask({
        sensors: {
          update: [{ key: "s1", type: "voltage", maxVal: 5, scale: { type: "none" } }],
        },
      });
      expect(sensors[0]).toEqual({
        ...task.sensors[0],
        maxVal: 5,
        scale: { type: "none" },
      });
    });

    it("should keep base fields and fill defaults on a variant change", () => {
      const { sensors } = applyTask({
        sensors: { update: [{ key: "s2", type: "voltage" }] },
      });
      expect(sensors[1]).toEqual({
        key: "s2",
        name: "Outlet",
        port: 1,
        type: "voltage",
        maxVal: 10,
        scale: { type: "none" },
      });
    });

    it("should throw when a variant change leaves out a required field", () => {
      expect(() =>
        applyTask({ sensors: { update: [{ key: "s1", type: "thermocouple" }] } }),
      ).toThrow("s1: couple is required");
    });

    it("should keep a stale error stale when it names the entry", () => {
      const entryZ = z.object({ key: z.string(), tags: z.string().array() });
      const entryUpdateZ = z.object({
        key: z.string(),
        tags: patch.valueListZ(z.string()).optional(),
      });
      const u = z.object({
        entries: patch.listZ({ e: entryZ, k: z.string(), u: entryUpdateZ }).optional(),
      });
      const t = z.object({ entries: entryZ.array() });
      let caught: unknown;
      try {
        patch.applyUpdate(
          { u, t },
          { entries: [{ key: "e1", tags: ["a"] }] },
          {
            value: { entries: { update: [{ key: "e1", tags: { remove: ["t9"] } }] } },
            guards: [],
          },
        );
      } catch (e) {
        caught = e;
      }
      expect(patch.StaleError.matches(caught)).toBe(true);
      expect((caught as Error).message).toBe("e1: t9 not found");
    });

    it("should merge base fields into an entry without a type", () => {
      const { sensors } = applyTask({
        sensors: { update: [{ key: "s2", name: "Exit", port: 4 }] },
      });
      expect(sensors[1]).toEqual({ ...task.sensors[1], name: "Exit", port: 4 });
    });

    it("should throw for a variant field without a type", () => {
      const value = { sensors: { update: [{ key: "s1", maxVal: 5 }] } };
      expect(taskPayloadZ.safeParse({ value }).success).toBe(false);
      expect(() => applyUnparsed(value)).toThrow(
        "The update schema holds no field maxVal",
      );
    });

    it("should apply an update to a root union", () => {
      const apply = (value: unknown) =>
        patch.applyUpdate(viewSchemas, view, viewPayloadZ.parse({ value })).next;
      expect(apply({ name: "All" })).toEqual({ ...view, name: "All" });
      expect(apply({ type: "range", where: null })).toEqual({
        key: "v1",
        name: "Ranges",
        type: "range",
      });
      expect(apply({ type: "status" })).toEqual({
        key: "v1",
        name: "Ranges",
        type: "status",
        level: "info",
      });
    });
  });

  describe("diffUpdate", () => {
    it("should set unmarked fields", () => {
      expect(
        patch.diffUpdate(taskUpdateZ, task, { ...task, name: "Rig", props: {} }),
      ).toEqual({ value: { name: "Rig", props: {} }, guards: [] });
    });

    it("should merge into a union field under a guard", () => {
      const next: Task = { ...task, scale: { type: "linear", slope: 3, offset: 0 } };
      expect(patch.diffUpdate(taskUpdateZ, task, next)).toEqual({
        value: { scale: { type: "linear", slope: 3 } },
        guards: [{ path: ["scale", "type"], value: "linear" }],
      });
    });

    it("should take every variant field of an entry that changes variant", () => {
      const next = structuredClone(task);
      next.sensors[1] = {
        key: "s2",
        name: "Outlet",
        port: 1,
        type: "voltage",
        maxVal: 10,
        scale: { type: "map", min: 0, max: 2 },
      };
      expect(patch.diffUpdate(taskUpdateZ, task, next)).toEqual({
        value: {
          sensors: {
            update: [
              {
                key: "s2",
                type: "voltage",
                maxVal: 10,
                scale: { type: "map", min: 0, max: 2 },
              },
            ],
          },
        },
        guards: [],
      });
    });

    it("should clear an optional union field with null", () => {
      expect(patch.diffUpdate(taskUpdateZ, task, taskWithoutFallback)).toEqual({
        value: { fallback: null },
        guards: [],
      });
    });

    const changed = (edit: (next: Task) => void): Task => {
      const next = structuredClone(task);
      edit(next);
      return next;
    };
    const roundTrips: [string, Task, Task][] = [
      ["unmarked fields", task, { ...task, name: "Rig", props: { slot: 2 } }],
      [
        "a union field that changes variant",
        task,
        { ...task, scale: { type: "map", min: 1, max: 4 } },
      ],
      ["a cleared optional union field", task, taskWithoutFallback],
      [
        "an added optional union field",
        taskWithoutFallback,
        { ...task, fallback: { type: "linear", slope: 3, offset: 1 } },
      ],
      [
        "an entry that changes variant",
        task,
        changed((next) => {
          next.sensors[1] = {
            key: "s2",
            name: "Outlet",
            port: 1,
            type: "voltage",
            maxVal: 2,
            scale: { type: "linear", slope: 2, offset: 0 },
          };
        }),
      ],
      [
        "an entry that merges a union field",
        task,
        changed((next) => {
          next.sensors[0] = {
            key: "s1",
            name: "Inlet",
            port: 0,
            type: "voltage",
            maxVal: 3,
            scale: { type: "map", min: 0, max: 9 },
          };
        }),
      ],
    ];
    roundTrips.forEach(([name, base, next]) =>
      it(`should give an update that applies: ${name}`, () => {
        const diffed = patch.diffUpdate(taskUpdateZ, base, next);
        expect(taskPayloadZ.parse(diffed)).toEqual(diffed);
        const applied = patch.applyUpdate(taskSchemas, base, diffed);
        expect(applied.next).toEqual(next);
        expect(patch.applyUpdate(taskSchemas, next, applied.inverse).next).toEqual(
          base,
        );
      }),
    );

    it("should give an update that applies to a root union", () => {
      const next: View = { key: "v1", name: "Errors", type: "status", level: "error" };
      const diffed = patch.diffUpdate<typeof viewUpdateZ, View>(
        viewUpdateZ,
        view,
        next,
      );
      expect(diffed).toEqual({
        value: { type: "status", name: "Errors", level: "error" },
        guards: [],
      });
      expect(viewPayloadZ.parse(diffed)).toEqual(diffed);
      expect(patch.applyUpdate(viewSchemas, view, diffed).next).toEqual(next);
    });
  });

  describe("nest", () => {
    it("should nest into a union field with the variant that doc holds", () => {
      expect(patch.nest(taskUpdateZ, task, ["scale", "slope"], 3)).toEqual({
        scale: { type: "linear", slope: 3 },
      });
    });

    it("should nest into an optional union field", () => {
      expect(patch.nest(taskUpdateZ, task, ["fallback", "max"], 4)).toEqual({
        fallback: { type: "map", max: 4 },
      });
    });

    it("should nest into a union field of an untagged list entry", () => {
      const path = ["sensors", "s1", "scale", "offset"];
      const value = patch.nest(taskUpdateZ, task, path, 2);
      expect(value).toEqual({
        sensors: {
          update: [
            { key: "s1", type: "voltage", scale: { type: "linear", offset: 2 } },
          ],
        },
      });
      expect(applyTask(value).sensors[0]).toEqual({
        ...task.sensors[0],
        scale: { type: "linear", slope: 1, offset: 2 },
      });
    });

    it("should treat an unmarked field as a leaf", () => {
      expect(patch.nest(taskUpdateZ, task, ["props"], { slot: 2 })).toEqual({
        props: { slot: 2 },
      });
      expect(() => patch.nest(taskUpdateZ, task, ["props", "rack"], 2)).toThrow(
        "[props] holds no fields",
      );
    });
  });
});

const cellZ = z.object({ value: z.string(), width: z.number().default(1) });
const cellUpdateZ = z.object({
  value: z.string().optional(),
  width: z.number().optional(),
});
const sheetZ = z.object({
  cells: z.record(z.string(), cellZ),
  widths: z.record(z.string(), z.number()),
  scales: z.record(z.string(), scaleZ),
});
type Sheet = z.infer<typeof sheetZ>;
const sheetUpdateZ = z.object({
  cells: patch.mapZ({ k: z.string(), v: cellZ, u: cellUpdateZ }).optional(),
  widths: patch.mapZ({ k: z.string(), v: z.number(), u: z.number() }).optional(),
  scales: patch.mapZ({ k: z.string(), v: scaleZ, u: scaleUpdateZ }).optional(),
});
const sheetSchemas = { u: sheetUpdateZ, t: sheetZ };
const sheetPayloadZ = patch.updateZ(sheetUpdateZ);
const sheet: Sheet = {
  cells: { c1: { value: "Pressure", width: 2 }, c2: { value: "Flow", width: 1 } },
  widths: { c1: 10 },
  scales: { s1: { type: "linear", slope: 2, offset: 0 } },
};

const applySheet = (value: unknown, doc: Sheet = sheet): Sheet =>
  patch.applyUpdate(sheetSchemas, doc, sheetPayloadZ.parse({ value })).next;

describe("typed maps", () => {
  describe("applyUpdate", () => {
    it("should remove, merge, and insert by key", () => {
      expect(
        applySheet({
          cells: {
            remove: ["c2"],
            update: { c1: { value: "Temperature" } },
            insert: { c3: { value: "Level", width: 1 } },
          },
        }).cells,
      ).toEqual({
        c1: { value: "Temperature", width: 2 },
        c3: { value: "Level", width: 1 },
      });
    });

    it("should remove a key before an insert of the same key", () => {
      expect(
        applySheet({
          cells: { remove: ["c2"], insert: { c2: { value: "Mass", width: 3 } } },
        }).cells.c2,
      ).toEqual({ value: "Mass", width: 3 });
    });

    it("should replace the whole map with set", () => {
      expect(applySheet({ cells: { set: {} } }).cells).toEqual({});
    });

    it("should replace a plain value with its update", () => {
      expect(applySheet({ widths: { update: { c1: 4 } } }).widths).toEqual({ c1: 4 });
    });

    it("should merge a union value of the same variant", () => {
      expect(
        applySheet({ scales: { update: { s1: { type: "linear", offset: 1 } } } }).scales
          .s1,
      ).toEqual({ type: "linear", slope: 2, offset: 1 });
    });

    it("should fill the defaults of a union value that changes variant", () => {
      expect(
        applySheet({ scales: { update: { s1: { type: "map", max: 5 } } } }).scales.s1,
      ).toEqual({ type: "map", min: 0, max: 5 });
    });

    const failures: [string, unknown, Required<Vector>["error"]][] = [
      [
        "a set beside another operator",
        { cells: { set: {}, remove: ["c1"] } },
        { kind: "validation", message: "set cannot be combined with other operators" },
      ],
      [
        "a remove of a missing key",
        { cells: { remove: ["c9"] } },
        { kind: "stale", message: "c9 not found" },
      ],
      [
        "an update of a missing key",
        { cells: { update: { c9: { value: "Mass" } } } },
        { kind: "stale", message: "c9 not found" },
      ],
      [
        "an insert of a key that exists",
        { cells: { insert: { c1: { value: "Mass", width: 1 } } } },
        { kind: "unique", message: "c1 already exists" },
      ],
      [
        "a variant change without a required field",
        { scales: { update: { s1: { type: "map" } } } },
        { kind: "validation", message: "s1: max is required" },
      ],
    ];
    failures.forEach(([name, value, error]) =>
      it(`should throw for ${name}`, () => expectError(() => applySheet(value), error)),
    );
  });

  describe("diffUpdate", () => {
    it("should remove gone keys, insert new keys, and update changed values", () => {
      const next: Sheet = {
        cells: {
          c1: { value: "Pressure", width: 3 },
          c3: { value: "Level", width: 1 },
        },
        widths: { c1: 11 },
        scales: { s1: { type: "linear", slope: 3, offset: 0 } },
      };
      expect(patch.diffUpdate(sheetUpdateZ, sheet, next)).toEqual({
        value: {
          cells: {
            remove: ["c2"],
            update: { c1: { width: 3 } },
            insert: { c3: { value: "Level", width: 1 } },
          },
          widths: { update: { c1: 11 } },
          scales: { update: { s1: { type: "linear", slope: 3 } } },
        },
        guards: [{ path: ["scales", "s1", "type"], value: "linear" }],
      });
    });

    const roundTrips: [string, Sheet, Sheet][] = [
      [
        "changed values",
        sheet,
        {
          cells: {
            c1: { value: "Pressure", width: 3 },
            c3: { value: "Level", width: 1 },
          },
          widths: {},
          scales: { s1: { type: "linear", slope: 3, offset: 0 } },
        },
      ],
      [
        "a union value that changes variant",
        sheet,
        { ...sheet, scales: { s1: { type: "map", min: 0, max: 4 } } },
      ],
      ["emptied maps", sheet, { cells: {}, widths: {}, scales: {} }],
      ["filled maps", { cells: {}, widths: {}, scales: {} }, sheet],
    ];
    roundTrips.forEach(([name, base, next]) =>
      it(`should give an update that applies: ${name}`, () => {
        const diffed = patch.diffUpdate(sheetUpdateZ, base, next);
        expect(sheetPayloadZ.parse(diffed)).toEqual(diffed);
        const applied = patch.applyUpdate(sheetSchemas, base, diffed);
        expect(applied.next).toEqual(next);
        expect(patch.applyUpdate(sheetSchemas, next, applied.inverse).next).toEqual(
          base,
        );
      }),
    );
  });

  describe("nest", () => {
    it("should nest into a map value by key", () => {
      const value = patch.nest(sheetUpdateZ, sheet, ["cells", "c1", "value"], "Mass");
      expect(value).toEqual({ cells: { update: { c1: { value: "Mass" } } } });
      expect(applySheet(value).cells.c1).toEqual({ value: "Mass", width: 2 });
    });

    it("should nest into a union value with the variant that doc holds", () => {
      expect(patch.nest(sheetUpdateZ, sheet, ["scales", "s1", "slope"], 3)).toEqual({
        scales: { update: { s1: { type: "linear", slope: 3 } } },
      });
    });

    it("should nest a plain value as a leaf", () => {
      expect(patch.nest(sheetUpdateZ, sheet, ["widths", "c1"], 4)).toEqual({
        widths: { update: { c1: 4 } },
      });
      expect(() => patch.nest(sheetUpdateZ, sheet, ["widths", "c1", "px"], 4)).toThrow(
        "[widths, c1] holds no fields",
      );
    });
  });
});
