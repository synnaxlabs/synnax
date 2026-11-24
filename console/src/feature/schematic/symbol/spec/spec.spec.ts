// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import { color } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Spec } from "@/feature/schematic/symbol/spec";

const STATE = Spec.createState().key;
const RED = color.construct("#ff0000");
const BLUE = color.construct("#0000ff");
const GREEN = color.construct("#00ff00");

const fixed = (c: color.Color): schematic.symbol.Paint => ({
  type: "color",
  value: { type: "fixed", value: c },
});

const equals = (...values: number[]): schematic.symbol.Condition => ({
  type: "equals",
  state: STATE,
  values,
});

const createSpec = (
  overrides: Partial<schematic.symbol.Spec> = {},
): schematic.symbol.Spec => ({
  drawing: { type: "svg", markup: "<svg/>" },
  parts: [
    { key: "body", name: "Body", selectors: ["#body"] },
    { key: "seat", name: "Seat", selectors: ["#seat"] },
  ],
  handles: [],
  states: [Spec.createState()],
  commands: [],
  looks: [],
  interactions: [],
  variant: "static",
  scale: 1,
  strokeScaled: false,
  ...overrides,
});

describe("symbol editor spec", () => {
  describe("effective", () => {
    it("should return the field that wins for the part at the value", () => {
      const looks: schematic.symbol.Look[] = [
        { key: "a", part: "body", when: equals(0, 1), fill: fixed(RED) },
        { key: "b", part: "body", fill: fixed(BLUE) },
      ];
      expect(Spec.effective({ looks }, "body", "fill", 0)).toEqual(fixed(BLUE));
    });

    it("should count a later look of the whole drawing for a part", () => {
      const looks: schematic.symbol.Look[] = [
        { key: "a", part: "body", fill: fixed(BLUE) },
        { key: "b", when: equals(0), fill: fixed(RED) },
      ];
      expect(Spec.effective({ looks }, "body", "fill", 0)).toEqual(fixed(RED));
      expect(Spec.effective({ looks }, "body", "fill", 1)).toEqual(fixed(BLUE));
    });

    it("should not count the looks of a part for the whole drawing", () => {
      const looks: schematic.symbol.Look[] = [
        { key: "a", part: "body", fill: fixed(RED) },
      ];
      expect(Spec.effective({ looks }, undefined, "fill", 0)).toBeUndefined();
    });
  });

  describe("own", () => {
    it("should not count a look of the whole drawing for a part", () => {
      const looks: schematic.symbol.Look[] = [
        { key: "a", part: "body", fill: fixed(BLUE) },
        { key: "b", when: equals(0), fill: fixed(RED) },
      ];
      expect(Spec.own({ looks }, "body", "fill", 0)).toEqual(fixed(BLUE));
      expect(Spec.own({ looks: looks.slice(1) }, "body", "fill", 0)).toBeUndefined();
    });

    it("should return the look of the whole drawing for the whole drawing", () => {
      const looks: schematic.symbol.Look[] = [
        { key: "a", part: "body", fill: fixed(BLUE) },
        { key: "b", when: equals(0), fill: fixed(RED) },
      ];
      expect(Spec.own({ looks }, undefined, "fill", 0)).toEqual(fixed(RED));
      expect(Spec.own({ looks }, undefined, "fill", 1)).toBeUndefined();
    });
  });

  describe("setOwn", () => {
    it("should add a drawing look after its last hider, not after every look", () => {
      const spec = createSpec({
        looks: [
          { key: "a", part: "body", when: equals(0), fill: fixed(RED) },
          { key: "b", fill: fixed(BLUE) },
          { key: "c", part: "seat", when: equals(0), fill: fixed(RED) },
        ],
      });
      const next = Spec.setOwn(spec, undefined, "fill", fixed(GREEN), 0);
      expect(next.looks.map((l) => l.key)).toEqual(["a", "b", expect.any(String), "c"]);
    });

    it("should change the motion of a part that a whole drawing motion follows", () => {
      const spec = createSpec({
        looks: [
          {
            key: "a",
            part: "body",
            when: equals(0),
            motion: { type: "spin", rate: 1 },
          },
          { key: "b", when: equals(0), motion: { type: "flow", rate: 1 } },
        ],
      });
      const flow: schematic.symbol.Motion = { type: "flow", rate: 2 };
      expect(Spec.setOwn(spec, "body", "motion", flow, 0).looks).toEqual([
        { ...spec.looks[0], motion: flow },
        spec.looks[1],
      ]);
    });

    it("should change the blink of a part that a whole drawing blink follows", () => {
      const spec = createSpec({
        looks: [
          { key: "a", part: "body", when: equals(0), blink: { period: 1e9 } },
          { key: "b", when: equals(0), blink: { period: 2e9 } },
        ],
      });
      const blink: schematic.symbol.Blink = { period: 3e9 };
      expect(Spec.setOwn(spec, "body", "blink", blink, 0).looks).toEqual([
        { ...spec.looks[0], blink },
        spec.looks[1],
      ]);
    });

    it("should clear a field at every value and keep it on single values", () => {
      const spec = createSpec({
        looks: [
          { key: "a", part: "body", fill: fixed(RED), stroke: fixed(BLUE) },
          { key: "b", part: "body", fill: fixed(RED) },
          { key: "c", part: "body", when: equals(1), fill: fixed(BLUE) },
        ],
      });
      expect(Spec.setOwn(spec, "body", "fill", undefined).looks).toEqual([
        { key: "a", part: "body", stroke: fixed(BLUE) },
        spec.looks[2],
      ]);
    });

    it("should set the field on the look of the part for the value alone", () => {
      const looks: schematic.symbol.Look[] = [
        { key: "a", part: "body", when: equals(0), fill: fixed(BLUE) },
      ];
      expect(
        Spec.setOwn(createSpec({ looks }), "body", "stroke", fixed(RED), 0).looks,
      ).toEqual([{ ...looks[0], stroke: fixed(RED) }]);
    });

    it("should set a field that is not a paint", () => {
      const looks: schematic.symbol.Look[] = [
        { key: "a", part: "body", when: equals(1), fill: fixed(BLUE) },
      ];
      const blink = { period: 1e9 };
      expect(
        Spec.setOwn(createSpec({ looks }), "body", "blink", blink, 1).looks,
      ).toEqual([{ ...looks[0], blink }]);
    });

    it("should add a look when the part has none for the value", () => {
      const looks: schematic.symbol.Look[] = [
        { key: "a", part: "body", when: equals(1), stroke: fixed(BLUE) },
      ];
      const { looks: next } = Spec.setOwn(
        createSpec({ looks }),
        "body",
        "stroke",
        fixed(RED),
        0,
      );
      expect(next).toEqual([
        looks[0],
        {
          key: expect.any(String),
          part: "body",
          when: equals(0),
          stroke: fixed(RED),
        },
      ]);
    });

    it("should split a look on several values when it sets the field at one", () => {
      const entry: schematic.symbol.Look = {
        key: "entry",
        part: "body",
        when: equals(0, 1),
        fill: fixed(BLUE),
        stroke: { type: "entry", state: STATE },
      };
      expect(
        Spec.setOwn(createSpec({ looks: [entry] }), "body", "stroke", fixed(RED), 0)
          .looks,
      ).toEqual([
        { ...entry, when: equals(1) },
        { ...entry, key: expect.any(String), when: equals(0), stroke: fixed(RED) },
      ]);
    });

    it("should add a look at the end when a later look on every value wins", () => {
      const looks: schematic.symbol.Look[] = [
        { key: "a", part: "body", when: equals(0), stroke: fixed(BLUE) },
        { key: "b", part: "body", stroke: fixed(BLUE) },
      ];
      const { looks: next } = Spec.setOwn(
        createSpec({ looks }),
        "body",
        "stroke",
        fixed(RED),
        0,
      );
      expect(next).toEqual([
        looks[1],
        { key: expect.any(String), part: "body", when: equals(0), stroke: fixed(RED) },
      ]);
    });

    it("should take an old motion of another kind off the owned look", () => {
      const spin = {
        key: "a",
        part: "body",
        when: equals(0),
        motion: { type: "spin" as const, rate: 1 },
      };
      const flow = {
        key: "b",
        part: "body",
        motion: { type: "flow" as const, rate: 1 },
      };
      const spec = createSpec({ looks: [{ ...spin, stroke: fixed(RED) }, flow] });
      const next = Spec.setOwn(spec, "body", "motion", flow.motion, 0);
      expect(next.looks).toEqual([
        { key: "a", part: "body", when: equals(0), stroke: fixed(RED) },
        flow,
        { key: expect.any(String), part: "body", when: equals(0), motion: flow.motion },
      ]);
    });

    it("should take an old motion off an earlier owned look", () => {
      const spin = {
        key: "a",
        part: "body",
        when: equals(0),
        motion: { type: "spin" as const, rate: 1 },
      };
      const stroke = { key: "b", part: "body", stroke: fixed(BLUE) };
      const first = Spec.setOwn(
        createSpec({ looks: [spin, stroke] }),
        "body",
        "stroke",
        fixed(RED),
        0,
      );
      const flow = { type: "flow" as const, rate: 1 };
      expect(Spec.setOwn(first, "body", "motion", flow, 0).looks).toEqual([
        stroke,
        {
          key: expect.any(String),
          part: "body",
          when: equals(0),
          stroke: fixed(RED),
          motion: flow,
        },
      ]);
    });

    it("should set the field on the look of the part for every value", () => {
      const looks: schematic.symbol.Look[] = [
        { key: "a", part: "body", fill: fixed(BLUE) },
        { key: "b", part: "body", when: equals(0), stroke: fixed(BLUE) },
      ];
      expect(
        Spec.setOwn(createSpec({ looks }), "body", "stroke", fixed(RED)).looks,
      ).toEqual([{ ...looks[0], stroke: fixed(RED) }, looks[1]]);
    });

    it("should add a look on every value before the looks on one value", () => {
      const looks: schematic.symbol.Look[] = [
        { key: "a", part: "body", when: equals(0), hidden: false },
      ];
      expect(Spec.setOwn(createSpec({ looks }), "body", "hidden", true).looks).toEqual([
        { key: expect.any(String), part: "body", hidden: true },
        looks[0],
      ]);
    });

    it("should add a look of the whole drawing before the looks of the parts", () => {
      const looks: schematic.symbol.Look[] = [
        { key: "a", part: "body", fill: fixed(BLUE) },
      ];
      expect(
        Spec.setOwn(createSpec({ looks }), undefined, "fill", fixed(RED)).looks,
      ).toEqual([
        { key: expect.any(String), part: undefined, fill: fixed(RED) },
        looks[0],
      ]);
    });

    it("should add a look of a part after the drawing looks on one value", () => {
      const looks: schematic.symbol.Look[] = [
        { key: "a", when: equals(0), fill: fixed(BLUE) },
      ];
      expect(
        Spec.setOwn(createSpec({ looks }), "body", "fill", fixed(RED)).looks,
      ).toEqual([
        looks[0],
        { key: expect.any(String), part: "body", fill: fixed(RED) },
      ]);
    });

    it("should add a look at the end when a later drawing look wins", () => {
      const looks: schematic.symbol.Look[] = [
        { key: "a", part: "body", fill: fixed(BLUE) },
        { key: "b", fill: fixed(BLUE) },
      ];
      expect(
        Spec.setOwn(createSpec({ looks }), "body", "fill", fixed(RED)).looks,
      ).toEqual([
        looks[1],
        { key: expect.any(String), part: "body", fill: fixed(RED) },
      ]);
    });

    it("should clear the field and drop a look that then sets nothing", () => {
      const looks: schematic.symbol.Look[] = [
        { key: "a", part: "body", when: equals(0), stroke: fixed(RED) },
        { key: "b", part: "seat", when: equals(0), stroke: fixed(RED) },
      ];
      expect(
        Spec.setOwn(createSpec({ looks }), "body", "stroke", undefined, 0).looks,
      ).toEqual([looks[1]]);
    });

    it("should keep the other fields of a look when it clears one", () => {
      const looks: schematic.symbol.Look[] = [
        {
          key: "a",
          part: "body",
          when: equals(0),
          stroke: fixed(RED),
          fill: fixed(BLUE),
        },
      ];
      expect(
        Spec.setOwn(createSpec({ looks }), "body", "stroke", undefined, 0).looks,
      ).toEqual([{ ...looks[0], stroke: undefined }]);
    });

    it("should narrow a look on several values when it clears one of them", () => {
      const entry: schematic.symbol.Look = {
        key: "entry-body-fill",
        part: "body",
        when: equals(0, 1),
        fill: { type: "entry", state: STATE },
      };
      expect(
        Spec.setOwn(createSpec({ looks: [entry] }), "body", "fill", undefined, 1).looks,
      ).toEqual([{ ...entry, when: equals(0) }]);
    });

    it("should clear the field on every look of the part at the value", () => {
      const looks: schematic.symbol.Look[] = [
        {
          key: "entry-body-fill",
          part: "body",
          when: equals(0, 1),
          fill: { type: "entry", state: STATE },
        },
        { key: "a", part: "body", when: equals(1), fill: fixed(RED) },
      ];
      expect(
        Spec.setOwn(createSpec({ looks }), "body", "fill", undefined, 1).looks,
      ).toEqual([{ ...looks[0], when: equals(0) }]);
    });

    it("should keep the other fields at the value when it splits a look", () => {
      const look: schematic.symbol.Look = {
        key: "a",
        part: "body",
        when: equals(0, 1),
        stroke: fixed(RED),
        fill: fixed(BLUE),
      };
      expect(
        Spec.setOwn(createSpec({ looks: [look] }), "body", "stroke", undefined, 1)
          .looks,
      ).toEqual([
        { ...look, when: equals(0) },
        {
          key: expect.any(String),
          part: "body",
          when: equals(1),
          stroke: undefined,
          fill: fixed(BLUE),
        },
      ]);
    });
  });

  describe("setOwn on a spec with no state", () => {
    it("should add the state with a look on a value", () => {
      const next = Spec.setOwn(
        createSpec({ states: [] }),
        "body",
        "stroke",
        fixed(RED),
        0,
      );
      expect(next.states).toEqual([Spec.createState()]);
      expect(next.looks).toEqual([
        expect.objectContaining({ part: "body", when: equals(0), stroke: fixed(RED) }),
      ]);
    });

    it("should keep the other states", () => {
      const other: schematic.symbol.State = { ...Spec.createState(), key: "other" };
      const next = Spec.setOwn(
        createSpec({ states: [other] }),
        "body",
        "fill",
        fixed(RED),
        0,
      );
      expect(next.states).toEqual([other, Spec.createState()]);
    });

    it("should not add the state for a look on every value", () => {
      const next = Spec.setOwn(createSpec({ states: [] }), "body", "fill", fixed(RED));
      expect(next.states).toEqual([]);
    });

    it("should keep the spec when it clears a field that no look sets", () => {
      const spec = createSpec({ states: [] });
      expect(Spec.setOwn(spec, "body", "stroke", undefined, 0)).toBe(spec);
    });
  });

  describe("nextPartName", () => {
    it("should take the lowest number that no part has", () => {
      const parts = [
        { key: "a", name: "Part 2", selectors: [] },
        { key: "b", name: "Part 3", selectors: [] },
      ];
      expect(Spec.nextPartName(parts)).toBe("Part 1");
      expect(
        Spec.nextPartName([...parts, { key: "c", name: "Part 1", selectors: [] }]),
      ).toBe("Part 4");
    });
  });

  describe("removePart", () => {
    it("should remove the part and the looks on it", () => {
      const spec = createSpec({
        looks: [
          { key: "a", part: "body", stroke: fixed(RED) },
          { key: "b", part: "seat", stroke: fixed(RED) },
        ],
      });
      const next = Spec.removePart(spec, "body");
      expect(next.parts.map((p) => p.key)).toEqual(["seat"]);
      expect(next.looks.map((l) => l.key)).toEqual(["b"]);
    });
  });

  describe("toggleSelector", () => {
    it("should move a selector to the part from the other parts", () => {
      const next = Spec.toggleSelector(createSpec().parts, "body", "#seat");
      expect(next).toEqual([
        { key: "body", name: "Body", selectors: ["#body", "#seat"] },
        { key: "seat", name: "Seat", selectors: [] },
      ]);
    });

    it("should remove a selector that the part already has", () => {
      const next = Spec.toggleSelector(createSpec().parts, "body", "#body");
      expect(next[0].selectors).toEqual([]);
    });
  });

  describe("setVariant", () => {
    it("should copy a Base look of the whole drawing to Active", () => {
      const spec = createSpec({
        looks: [{ key: "a", when: equals(0), fill: fixed(RED) }],
      });
      const next = Spec.setVariant(spec, "actuator");
      expect(next.looks).toEqual([
        spec.looks[0],
        { key: expect.any(String), when: equals(1), fill: fixed(RED) },
      ]);
    });

    it("should copy a Base look of the whole drawing before the looks of parts", () => {
      const spec = createSpec({
        looks: [
          { key: "a", when: equals(0), fill: fixed(RED) },
          { key: "b", part: "body", when: equals(0, 1), fill: fixed(BLUE) },
        ],
      });
      const next = Spec.setVariant(spec, "actuator");
      expect(next.looks.map((l) => l.key)).toEqual(["a", expect.any(String), "b"]);
      expect(Spec.effective(next, "body", "fill", 1)).toEqual(fixed(BLUE));
    });

    it("should drop an Active look of the whole drawing from a static symbol", () => {
      const spec = createSpec({
        variant: "actuator",
        looks: [
          { key: "a", when: equals(0), fill: fixed(RED) },
          { key: "b", when: equals(1), fill: fixed(BLUE) },
        ],
      });
      expect(Spec.setVariant(spec, "static").looks.map((l) => l.key)).toEqual(["a"]);
    });

    it("should give an actuator the commands, the match, and the Base looks on Active", () => {
      const spec = createSpec({
        looks: [{ key: "a", part: "body", when: equals(0), stroke: fixed(RED) }],
      });
      const next = Spec.setVariant(spec, "actuator");
      expect(next.variant).toBe("actuator");
      expect(next.states[0]).toMatchObject({
        match: { type: "within", bounds: { lower: 0.9, upper: 1.1 } },
      });
      expect(next.commands.map((c) => [c.key, c.value])).toEqual([
        ["deactivate", { type: "fixed", value: 0 }],
        ["activate", { type: "fixed", value: 1 }],
      ]);
      expect(next.interactions).toEqual([
        { type: "next", key: "next", commands: ["deactivate", "activate"] },
      ]);
      expect(next.looks).toHaveLength(2);
      expect(next.looks[1]).toMatchObject({
        part: "body",
        when: equals(1),
        stroke: fixed(RED),
      });
    });

    it("should take the Active looks, commands, and clicks from a static symbol", () => {
      const actuator = Spec.setVariant(
        createSpec({
          looks: [
            { key: "a", part: "body", when: equals(0), stroke: fixed(RED) },
            { key: "b", part: "body", when: equals(0, 1), fill: fixed(BLUE) },
          ],
        }),
        "actuator",
      );
      const next = Spec.setVariant(actuator, "static");
      expect(next.looks.map((l) => l.key)).toEqual(["a", "b"]);
      expect(next.commands).toEqual([]);
      expect(next.interactions).toEqual([]);
      expect(next.states[0]).toMatchObject({ match: { type: "exact" } });
    });
  });
});
