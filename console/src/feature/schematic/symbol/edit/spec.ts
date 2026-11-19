// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type enumeration, type schematic } from "@synnaxlabs/client";
import { type color, id } from "@synnaxlabs/x";

type Spec = schematic.symbol.Spec;
type Look = schematic.symbol.Look;

// STATE is the key of the one state that the editor edits.
const STATE = "active";

/** Field is a paint field of a look. */
export type Field = "stroke" | "fill";

const BASE = 0;
const ACTIVE = 1;
const DEACTIVATE = "deactivate";
const ACTIVATE = "activate";

/** createState returns the state of a new symbol. */
export const createState = (): schematic.symbol.State => ({
  type: "enumeration",
  key: STATE,
  name: "Active",
  match: { type: "exact" },
  default: 0,
});

/** @returns the key of the enumeration of the state, if it has one. */
export const enumerationKey = (spec: Spec): enumeration.Key | undefined => {
  const state = spec.states.find((s) => s.key === STATE);
  return state?.type === "enumeration" ? state.enumeration : undefined;
};

const only = (look: Look, part: string, value: number): boolean =>
  look.part === part &&
  look.when?.state === STATE &&
  look.when.values.length === 1 &&
  look.when.values[0] === value;

const reaches = (look: Look, part: string, value: number): boolean =>
  (look.part == null || look.part === part) &&
  (look.when == null ||
    (look.when.state === STATE && look.when.values.includes(value)));

const setsNothing = ({ fill, stroke, hidden, motion, blink }: Look): boolean =>
  fill == null && stroke == null && hidden == null && motion == null && blink == null;

/**
 * @returns looks in which part shows c in field at value. It sets the field on the
 * part's look for value alone, or adds a look when that look is missing or a later look
 * sets the field. With no c, it clears the field on that look, so the other looks or
 * the drawing color the part. Looks that other parts or values read stay as they are.
 */
export const setColor = (
  looks: Look[],
  part: string,
  value: number,
  field: Field,
  c?: color.Color,
): Look[] => {
  const i = looks.findLastIndex((l) => only(l, part, value));
  if (c == null) {
    if (i < 0) return looks;
    const cleared = { ...looks[i], [field]: undefined };
    if (setsNothing(cleared)) return looks.filter((_, j) => j !== i);
    return looks.map((l, j) => (j === i ? cleared : l));
  }
  const paint: schematic.symbol.Paint = {
    type: "color",
    value: { type: "fixed", value: c },
  };
  const overridden = looks
    .slice(i + 1)
    .some((l) => reaches(l, part, value) && l[field] != null);
  if (i >= 0 && !overridden)
    return looks.map((l, j) => (j === i ? { ...l, [field]: paint } : l));
  return [
    ...looks,
    {
      key: `look-${id.create()}`,
      part,
      when: { type: "equals", state: STATE, values: [value] },
      [field]: paint,
    },
  ];
};

/** @returns spec without part and the looks on it. */
export const removePart = (spec: Spec, part: string): Spec => ({
  ...spec,
  parts: spec.parts.filter((p) => p.key !== part),
  looks: spec.looks.filter((l) => l.part !== part),
});

/**
 * @returns parts in which part selects selector, or no longer selects it when it
 * already did. A shape belongs to one part, so the other parts drop selector.
 */
export const toggleSelector = (
  parts: schematic.symbol.Part[],
  part: string,
  selector: string,
): schematic.symbol.Part[] => {
  const target = parts.find((p) => p.key === part);
  if (target == null) return parts;
  const selected = target.selectors.includes(selector);
  return parts.map((p) => {
    const rest = p.selectors.filter((s) => s !== selector);
    if (p.key !== part || selected) return { ...p, selectors: rest };
    return { ...p, selectors: [...rest, selector] };
  });
};

const withState = (spec: Spec, match: schematic.symbol.Matcher): Spec["states"] => {
  const states = spec.states.some((s) => s.key === STATE)
    ? spec.states
    : [...spec.states, createState()];
  return states.map((s) =>
    s.key === STATE && s.type === "enumeration" ? { ...s, match } : s,
  );
};

/**
 * @returns spec as variant. An actuator gets the Base looks again on Active, `within`
 * 0.9 to 1.1, the commands Off and On, and a click that picks between them, as the
 * migration gives a stored actuator. A static symbol loses the looks of Active alone,
 * the commands, and the clicks.
 */
export const setVariant = (spec: Spec, variant: string): Spec => {
  if (variant === "actuator") {
    const copies = spec.looks
      .filter((l) => l.part != null && only(l, l.part, BASE))
      .map((l) => ({
        ...l,
        key: `look-${id.create()}`,
        when: { type: "equals" as const, state: STATE, values: [ACTIVE] },
      }));
    return {
      ...spec,
      variant,
      states: withState(spec, {
        type: "within",
        bounds: { lower: 0.9, upper: 1.1 },
      }),
      looks: [...spec.looks, ...copies],
      commands: [
        {
          key: DEACTIVATE,
          name: "Off",
          state: STATE,
          value: { type: "fixed", value: BASE },
        },
        {
          key: ACTIVATE,
          name: "On",
          state: STATE,
          value: { type: "fixed", value: ACTIVE },
        },
      ],
      interactions: [{ type: "next", key: "next", commands: [DEACTIVATE, ACTIVATE] }],
    };
  }
  return {
    ...spec,
    variant,
    states: withState(spec, { type: "exact" }),
    looks: spec.looks.filter((l) => l.part == null || !only(l, l.part, ACTIVE)),
    commands: [],
    interactions: [],
  };
};
