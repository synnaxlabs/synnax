// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { color } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Spec } from "@/feature/schematic/symbol/spec";

const STATE = Spec.createState().key;
const RED = color.construct("#ff0000");
const BLUE = color.construct("#0000ff");

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
  describe("setColor", () => {
    it("should set the field on the look of the part for the value alone", () => {
      const looks: schematic.symbol.Look[] = [
        { key: "a", part: "body", when: equals(0), fill: fixed(BLUE) },
      ];
      expect(
        Spec.setColor(createSpec({ looks }), "body", 0, "stroke", RED).looks,
      ).toEqual([
        {
          key: "a",
          part: "body",
          when: equals(0),
          fill: fixed(BLUE),
          stroke: fixed(RED),
        },
      ]);
    });

    it("should add a look when the part has none for the value", () => {
      const looks: schematic.symbol.Look[] = [
        { key: "a", part: "body", when: equals(1), stroke: fixed(BLUE) },
      ];
      const { looks: next } = Spec.setColor(
        createSpec({ looks }),
        "body",
        0,
        "stroke",
        RED,
      );
      expect(next).toHaveLength(2);
      expect(next[0]).toEqual(looks[0]);
      expect(next[1]).toMatchObject({
        part: "body",
        when: equals(0),
        stroke: fixed(RED),
      });
    });

    it("should add a look after a later look that sets the field", () => {
      const looks: schematic.symbol.Look[] = [
        { key: "a", part: "body", when: equals(0), fill: fixed(BLUE) },
        {
          key: "entry",
          part: "body",
          when: equals(0, 1),
          stroke: { type: "entry", state: STATE },
        },
      ];
      const { looks: next } = Spec.setColor(
        createSpec({ looks }),
        "body",
        0,
        "stroke",
        RED,
      );
      expect(next.slice(0, 2)).toEqual(looks);
      expect(next[2]).toMatchObject({ part: "body", stroke: fixed(RED) });
    });

    it("should clear the field and drop a look that then sets nothing", () => {
      const looks: schematic.symbol.Look[] = [
        { key: "a", part: "body", when: equals(0), stroke: fixed(RED) },
        { key: "b", part: "seat", when: equals(0), stroke: fixed(RED) },
      ];
      expect(Spec.setColor(createSpec({ looks }), "body", 0, "stroke").looks).toEqual([
        looks[1],
      ]);
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
      expect(Spec.setColor(createSpec({ looks }), "body", 0, "stroke").looks).toEqual([
        {
          key: "a",
          part: "body",
          when: equals(0),
          stroke: undefined,
          fill: fixed(BLUE),
        },
      ]);
    });

    it("should narrow a look on several values when it clears one of them", () => {
      const entry: schematic.symbol.Look = {
        key: "entry-body-fill",
        part: "body",
        when: equals(0, 1),
        fill: { type: "entry", state: STATE },
      };
      expect(
        Spec.setColor(createSpec({ looks: [entry] }), "body", 1, "fill").looks,
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
      expect(Spec.setColor(createSpec({ looks }), "body", 1, "fill").looks).toEqual([
        { ...looks[0], when: equals(0) },
      ]);
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
        Spec.setColor(createSpec({ looks: [look] }), "body", 1, "stroke").looks,
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

  describe("setColor on a spec with no state", () => {
    it("should add the state with the look", () => {
      const next = Spec.setColor(createSpec({ states: [] }), "body", 0, "stroke", RED);
      expect(next.states).toEqual([Spec.createState()]);
      expect(next.looks).toEqual([
        expect.objectContaining({ part: "body", when: equals(0), stroke: fixed(RED) }),
      ]);
    });

    it("should keep the other states", () => {
      const other: schematic.symbol.State = { ...Spec.createState(), key: "other" };
      const next = Spec.setColor(
        createSpec({ states: [other] }),
        "body",
        0,
        "fill",
        RED,
      );
      expect(next.states).toEqual([other, Spec.createState()]);
    });

    it("should not add the state when it clears a field", () => {
      const spec = createSpec({ states: [] });
      expect(Spec.setColor(spec, "body", 0, "stroke")).toBe(spec);
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
