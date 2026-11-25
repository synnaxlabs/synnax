// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { enumeration, schematic } from "@synnaxlabs/client";

import { Primitive } from "@/schematic/node/common/primitive";

const ACTIVE = "active";

// The root of each built-in drawing carries its stroke width, so a copy draws as wide.
const STROKE_WIDTH = 2;

interface Family {
  enumeration: enumeration.Key;
  /** commands name the deactivate and activate commands of a family that clicks. */
  commands?: [deactivate: string, activate: string];
}

// The enumeration keys are the Core's built-in enumerations, which never change.
const FAMILIES = {
  valve: {
    enumeration: "0fdb5bf9-8929-4839-84f4-5bc0cda63344",
    commands: ["Close", "Open"],
  },
  normallyOpenValve: {
    enumeration: "18dd0627-e014-4c38-9fc8-cd077bdf3888",
    commands: ["Open", "Close"],
  },
  pump: {
    enumeration: "21709d6c-c7a8-41ca-89d0-6e946236491e",
    commands: ["Stop", "Start"],
  },
  switch: {
    enumeration: "d396f0f9-46ba-45a3-b9b5-36fc8b610625",
    commands: ["Turn off", "Turn on"],
  },
  light: { enumeration: "ed3c515d-10eb-468a-95bf-15b6204bb5f2" },
} as const satisfies Record<string, Family>;

/** FamilyKey names the states and commands that one kind of equipment shares. */
export type FamilyKey = keyof typeof FAMILIES;

const ENTRY: schematic.symbol.Paint = { type: "entry", state: ACTIVE };

/** STROKE_LOOK strokes the whole drawing with the color of the active entry. */
export const STROKE_LOOK: schematic.symbol.Look = { key: "stroke", stroke: ENTRY };

/**
 * @returns a look that fills part, or the whole drawing when part is absent, with the
 * color of the active entry while the active state is value.
 */
export const createFillLook = (
  value: number,
  part?: string,
): schematic.symbol.Look => ({
  key: part ?? "fill",
  part,
  when: { type: "equals", state: ACTIVE, values: [value] },
  fill: ENTRY,
});

const ACTIVE_LOOKS = [STROKE_LOOK, createFillLook(1)];

/**
 * DETAIL_LINES gives the `.pluto--detail` lines of a drawing the black or white that
 * reads on its fill while active. It keeps the default stroke and fill looks.
 */
export const DETAIL_LINES: Pick<schematic.symbol.Spec, "parts" | "looks"> = {
  parts: [{ key: "detail", name: "Detail", selectors: [".pluto--detail"] }],
  looks: [
    ...ACTIVE_LOOKS,
    {
      key: "detail",
      part: "detail",
      when: { type: "equals", state: ACTIVE, values: [1] },
      stroke: { type: "contrast", state: ACTIVE },
    },
  ],
};

type Behavior = Pick<
  schematic.symbol.Spec,
  "states" | "commands" | "looks" | "interactions" | "variant"
>;

const STATIC: Behavior = {
  states: [],
  commands: [],
  looks: [],
  interactions: [],
  variant: "static",
};

const createBehavior = (key: FamilyKey): Behavior => {
  const family: Family = FAMILIES[key];
  const states: schematic.symbol.State[] = [
    {
      type: "enumeration",
      key: ACTIVE,
      name: "Active",
      enumeration: family.enumeration,
      match: { type: "within", bounds: { lower: 0.9, upper: 1.1 } },
      default: 0,
    },
  ];
  if (family.commands == null) return { ...STATIC, states, looks: ACTIVE_LOOKS };
  const [deactivate, activate] = family.commands;
  return {
    states,
    commands: [
      {
        key: "deactivate",
        name: deactivate,
        state: ACTIVE,
        value: { type: "fixed", value: 0 },
      },
      {
        key: "activate",
        name: activate,
        state: ACTIVE,
        value: { type: "fixed", value: 1 },
      },
    ],
    looks: ACTIVE_LOOKS,
    interactions: [
      { type: "next", key: "click", commands: ["deactivate", "activate"] },
    ],
    variant: "actuator",
  };
};

export interface BuiltInParams
  extends
    Pick<schematic.symbol.Spec, "handles">,
    Partial<Pick<schematic.symbol.Spec, "parts" | "looks" | "interactions">> {
  /** svg is the markup of the drawing. */
  svg: string;
  /**
   * family gives the symbol its active state, and the commands and click of a family
   * that clicks. Looks default to a stroke of the whole drawing and a fill of it while
   * active. Looks and interactions in the params replace those of the family.
   */
  family?: FamilyKey;
}

/**
 * @returns the spec of a built-in symbol, static unless family says otherwise. Its
 * scale is the shrink Primitive.SVG gives a built-in drawing, so Drawing draws it at
 * the size of that drawing.
 */
export const createBuiltIn = ({
  svg,
  handles,
  family,
  parts = [],
  ...overrides
}: BuiltInParams): schematic.symbol.Spec => ({
  drawing: {
    type: "svg",
    markup: svg.replace(/<svg\b/, `<svg stroke-width="${STROKE_WIDTH}"`),
  },
  parts,
  handles,
  ...(family == null ? STATIC : createBehavior(family)),
  ...overrides,
  scale: Primitive.BASE_SCALE,
  strokeScaled: false,
});
