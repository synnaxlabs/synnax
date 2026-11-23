// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { enumeration, schematic } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";

type Spec = schematic.symbol.Spec;
type Look = schematic.symbol.Look;

/** STATE is the key of the one state that the editor edits. */
export const STATE = "active";

/** Field is a field of a look that the editor sets. */
export type Field = "fill" | "stroke" | "hidden" | "blink" | "motion";

const FIELDS: Field[] = ["fill", "stroke", "hidden", "blink", "motion"];

/** Target is the key of a part, or undefined for the whole drawing. */
export type Target = string | undefined;

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
export const enumerationKey = (
  spec: Pick<Spec, "states">,
): enumeration.Key | undefined => {
  const state = spec.states.find((s) => s.key === STATE);
  return state?.type === "enumeration" ? state.enumeration : undefined;
};

const only = (look: Look, target: Target, value: number): boolean =>
  look.part === target &&
  look.when?.state === STATE &&
  look.when.values.length === 1 &&
  look.when.values[0] === value;

// owns is true for a look of target at value alone, or at every value when value is
// undefined.
const owns = (look: Look, target: Target, value?: number): boolean =>
  value == null ? look.part === target && look.when == null : only(look, target, value);

// reaches is true for a look that target shows at value, or at every value when value
// is undefined. A look of the whole drawing reaches every part.
const reaches = (look: Look, target: Target, value?: number): boolean =>
  (look.part == null || look.part === target) &&
  (look.when == null ||
    (value != null && look.when.state === STATE && look.when.values.includes(value)));

const setsNothing = (look: Look): boolean => FIELDS.every((f) => look[f] == null);

// split copies onto value alone each look of target that sets field on value and other
// values. An edit at value then leaves the other values as they are.
const split = (looks: Look[], target: Target, field: Field, value: number): Look[] =>
  looks.flatMap((l): Look[] => {
    if (l.part !== target || l[field] == null || l.when == null) return [l];
    const { values } = l.when;
    if (l.when.state !== STATE || values.length < 2 || !values.includes(value))
      return [l];
    return [
      { ...l, when: { ...l.when, values: values.filter((v) => v !== value) } },
      { ...l, key: `look-${id.create()}`, when: { ...l.when, values: [value] } },
    ];
  });

// rank orders the looks that the editor adds: a look on a part wins over a look on the
// whole drawing, and a look on one value wins over a look on every value.
const rank = (look: Look): number =>
  (look.part == null ? 0 : 2) + (look.when == null ? 0 : 1);

const slot = (looks: Look[], look: Look): number => {
  const i = looks.findIndex((l) => rank(l) > rank(look));
  return i < 0 ? looks.length : i;
};

const withActive = (states: Spec["states"]): Spec["states"] =>
  states.some((s) => s.key === STATE) ? states : [...states, createState()];

/**
 * @returns the field that target shows at value after the looks apply in order, or at
 * every value when value is undefined. Looks of the whole drawing count for a part.
 */
export const effective = <F extends Field>(
  spec: Pick<Spec, "looks">,
  target: Target,
  field: F,
  value?: number,
): Look[F] | undefined =>
  spec.looks.findLast((l) => reaches(l, target, value) && l[field] != null)?.[field];

/**
 * @returns spec in which target owns v for field at value alone, or at every value when
 * value is undefined. A new look on a value adds the state when spec lacks it. With no
 * v, it takes field off the looks that target owns there, and a look on more values
 * keeps field on the others.
 */
export const setOwn = <F extends Field>(
  spec: Spec,
  target: Target,
  field: F,
  v: Look[F] | undefined,
  value?: number,
): Spec => {
  const looks = value == null ? spec.looks : split(spec.looks, target, field, value);
  // Only the look at keep sets field, so no old motion of another kind keeps running.
  const clear = (ls: Look[], keep = -1): Look[] =>
    ls.flatMap((l, k): Look[] => {
      if (k === keep || l[field] == null || !owns(l, target, value)) return [l];
      const cleared = { ...l, [field]: undefined };
      return setsNothing(cleared) ? [] : [cleared];
    });
  if (v == null) {
    if (!looks.some((l) => owns(l, target, value) && l[field] != null)) return spec;
    return { ...spec, looks: clear(looks) };
  }
  // Pluto gives each part its own motion and blink, and one motion of each kind, so
  // only those looks hide v.
  const kind = field === "motion" ? (v as schematic.symbol.Motion).type : undefined;
  const hides = (l: Look): boolean =>
    reaches(l, target, value) &&
    l[field] != null &&
    ((field !== "motion" && field !== "blink") || l.part === target) &&
    (kind == null || l.motion?.type === kind);
  const last = looks.findLastIndex(hides);
  const i = looks.findLastIndex((l) => owns(l, target, value));
  if (i >= 0 && last <= i)
    return { ...spec, looks: clear(looks.with(i, { ...looks[i], [field]: v }), i) };
  const look: Look = { key: `look-${id.create()}`, part: target, [field]: v };
  if (value != null) look.when = { type: "equals", state: STATE, values: [value] };
  const rest = clear(looks);
  const at = Math.max(slot(rest, look), rest.findLastIndex(hides) + 1);
  return {
    ...spec,
    states: value == null ? spec.states : withActive(spec.states),
    looks: rest.toSpliced(at, 0, look),
  };
};

/** @returns whether a look of target sets field on some values alone. */
export const variesByValue = (
  spec: Pick<Spec, "looks">,
  target: Target,
  field: Field,
): boolean =>
  spec.looks.some(
    (l) => l.part === target && l.when?.state === STATE && l[field] != null,
  );

/**
 * @returns spec in which field of target varies by value, with each value starting from
 * the field on every value. With varies false, it returns spec in which every value
 * takes the field that target shows at the first value.
 */
export const setVariesByValue = (
  spec: Spec,
  target: Target,
  field: Field,
  values: number[],
  varies: boolean,
): Spec => {
  if (varies) {
    const base = spec.looks.findLast((l) => owns(l, target) && l[field] != null);
    if (base == null) return spec;
    const cleared = setOwn(spec, target, field, undefined);
    return values.reduce((s, v) => setOwn(s, target, field, base[field], v), cleared);
  }
  // Looks of the whole drawing stay out, so target takes no field that it did not set.
  const mine = spec.looks.filter((l) => l.part === target);
  const first = effective({ looks: mine }, target, field, values[0]);
  const cleared = values.reduce((s, v) => setOwn(s, target, field, undefined, v), spec);
  return setOwn(cleared, target, field, first);
};

const CLIP_STYLE = /(^|;)\s*clip-path\s*:\s*(?!none\s*(;|$))[^;\s]/;

const clipped = (el: Element): boolean => {
  const clip = el.getAttribute("clip-path")?.trim() ?? "none";
  return (
    !["", "none"].includes(clip) || CLIP_STYLE.test(el.getAttribute("style") ?? "")
  );
};

/**
 * LevelIssue is why a level motion cannot draw a part. Level clips each shape of the
 * part by its own box, so "shapes" is a part of more than one shape. It also replaces
 * the clip-path of the shape, so "clip" is a shape that has its own.
 */
export type LevelIssue = "shapes" | "clip";

/**
 * @returns each part whose level motion cannot draw, mapped to the reason. Only the
 * looks that the editor reaches count, so the user can remove each one.
 */
export const levelIssues = (
  spec: Pick<Spec, "drawing" | "parts" | "looks">,
): Map<string, LevelIssue> => {
  const issues = new Map<string, LevelIssue>();
  const levels = new Set(
    spec.looks.flatMap((l) =>
      l.motion?.type === "level" &&
      l.part != null &&
      (l.when == null || l.when.state === STATE)
        ? [l.part]
        : [],
    ),
  );
  if (levels.size === 0) return issues;
  const doc = new DOMParser().parseFromString(spec.drawing.markup, "image/svg+xml");
  for (const p of spec.parts) {
    if (!levels.has(p.key)) continue;
    const matched = [
      ...new Set(p.selectors.flatMap((s) => [...doc.querySelectorAll(s)])),
    ];
    // The renderer moves only the outermost elements of a part.
    const shapes = matched.filter(
      (el) => !matched.some((o) => o !== el && o.contains(el)),
    );
    if (shapes.length > 1) issues.set(p.key, "shapes");
    else if (shapes.some(clipped)) issues.set(p.key, "clip");
  }
  return issues;
};

/** @returns the name "Part N" with the lowest N that no part in parts has. */
export const nextPartName = (parts: schematic.symbol.Part[]): string => {
  const taken = new Set(parts.map((p) => p.name));
  let n = 1;
  while (taken.has(`Part ${n}`)) n++;
  return `Part ${n}`;
};

/** @returns spec without part and the looks on it. */
export const removePart = (spec: Spec, part: string): Spec => ({
  ...spec,
  parts: spec.parts.filter((p) => p.key !== part),
  looks: spec.looks.filter((l) => l.part !== part),
});

/**
 * @returns the part that holds the shape of selector: the shape itself, a span of the
 * text, or the nearest group around it.
 */
export const owner = (
  spec: Pick<Spec, "drawing" | "parts">,
  selector: string,
): schematic.symbol.Part | undefined => {
  const doc = new DOMParser().parseFromString(spec.drawing.markup, "image/svg+xml");
  const shape = doc.querySelector(selector);
  if (shape == null) return undefined;
  const around: Element[] = [];
  for (let el = shape.parentElement; el != null; el = el.parentElement) around.push(el);
  // The spans of a text are part of its shape, but the shapes of a group are not.
  const spans = shape.tagName === "text" ? [...shape.querySelectorAll("*")] : [];
  for (const el of [shape, ...spans, ...around]) {
    const found = spec.parts.find((p) => p.selectors.some((s) => el.matches(s)));
    if (found != null) return found;
  }
  return undefined;
};

// holds returns whether every shape of inner sits inside a shape of outer.
const holds = (doc: Document, outer: string, inner: string): boolean => {
  const groups = [...doc.querySelectorAll(outer)];
  const shapes = [...doc.querySelectorAll(inner)];
  return (
    shapes.length > 0 &&
    shapes.every((s) => groups.some((g) => g !== s && g.contains(s)))
  );
};

/**
 * @returns parts in which part selects selector, or no longer selects it when it
 * already did. A shape belongs to one part, and selector covers the shapes inside its
 * shape, so every part drops their selectors, and the other parts drop selector. A
 * shape inside a group that part holds takes the group out of part, and a shape inside
 * a group that another part holds stays there, so the parts stay the same.
 */
export const toggleSelector = (
  spec: Pick<Spec, "drawing" | "parts">,
  part: string,
  selector: string,
): schematic.symbol.Part[] => {
  const target = spec.parts.find((p) => p.key === part);
  if (target == null) return spec.parts;
  const doc = new DOMParser().parseFromString(spec.drawing.markup, "image/svg+xml");
  const covered = (s: string): boolean => s === selector || holds(doc, selector, s);
  // The spans of a text are part of its shape, so a part that holds one holds the text.
  const text = doc.querySelector(selector)?.tagName === "text";
  if (text ? target.selectors.some(covered) : target.selectors.includes(selector))
    return spec.parts.map((p) => ({
      ...p,
      selectors: p.selectors.filter((s) => !covered(s)),
    }));
  const around = target.selectors.filter((s) => holds(doc, s, selector));
  if (around.length > 0)
    return spec.parts.map((p) =>
      p.key === part
        ? { ...p, selectors: p.selectors.filter((s) => !around.includes(s)) }
        : p,
    );
  if (spec.parts.some((p) => p.selectors.some((s) => holds(doc, s, selector))))
    return spec.parts;
  return spec.parts.map((p) => {
    const rest = p.selectors.filter((s) => !covered(s));
    return { ...p, selectors: p.key === part ? [...rest, selector] : rest };
  });
};

const withState = (spec: Spec, match: schematic.symbol.Matcher): Spec["states"] =>
  withActive(spec.states).map((s) =>
    s.key === STATE && s.type === "enumeration" ? { ...s, match } : s,
  );

/**
 * @returns looks as a static symbol draws them, with only Base: each look on Base
 * reaches every value, and the looks on other values go.
 */
export const staticLooks = (looks: Look[]): Look[] =>
  looks.flatMap((l): Look[] => {
    if (l.when?.state !== STATE) return [l];
    if (!l.when.values.includes(BASE)) return [];
    const { when: _, ...rest } = l;
    return [rest];
  });

/**
 * @returns spec as variant. An actuator gets the Base looks again on Active, `within`
 * 0.9 to 1.1, the commands Off and On, and a click that picks between them, as the
 * migration gives a stored actuator. A static symbol keeps its Base looks on every
 * value and loses its other looks on the state, the commands, and the clicks.
 */
export const setVariant = (spec: Spec, variant: string): Spec => {
  if (variant === "actuator") {
    const copies = spec.looks
      .filter((l) => only(l, l.part, BASE))
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
      looks: copies.reduce((ls, c) => ls.toSpliced(slot(ls, c), 0, c), spec.looks),
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
    looks: staticLooks(spec.looks),
    commands: [],
    interactions: [],
  };
};
