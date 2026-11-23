// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { color as clientColor, enumeration, schematic } from "@synnaxlabs/client";
import { useMemoDeepEqual } from "@synnaxlabs/lyra/memo";
import { theme } from "@synnaxlabs/lyra/theme";
import { Theming } from "@synnaxlabs/lyra/theming";
import { bounds, color } from "@synnaxlabs/x";

import { Color } from "@/color";
import { Enumeration } from "@/enumeration";

/** Paint is the stroke, fill, and visibility that one look gives a set of shapes. */
export interface Paint {
  /** selectors select the shapes, or the whole drawing when absent. */
  selectors?: string[];
  stroke?: color.Color;
  fill?: color.Color;
  hidden?: boolean;
}

/**
 * StateValues is the value of every state of a symbol, or the value of each state by
 * its key. A state with no value in the map, or a NaN value, reads its default entry,
 * or the lower bound of a number state.
 */
export type StateValues = number | ReadonlyMap<string, number>;

/** PaintContext holds what a look needs to resolve its colors. */
export interface PaintContext {
  /** value is the value of the states of the symbol. */
  value: StateValues;
  resolve: (ref: clientColor.Ref) => color.Color | undefined;
  enumerations: ReadonlyMap<enumeration.Key, enumeration.Enumeration>;
  /** nodeColor is the color that `node` paint draws. */
  nodeColor?: color.Color;
  /** defaultEntryColor is the color of an entry that has no color. */
  defaultEntryColor?: color.Color;
}

type Field = "stroke" | "fill";

// Band is how a drawing in a theme shows its paint.
interface Band {
  theme: theme.Theme;
  builtIn: boolean;
}

// banded says whether a drawing shows a paint type on a field in the symbol band of the
// theme. Every stroke bands. Only a built-in drawing bands its entry and node fills, as
// the built-in CSS does.
const banded = (
  field: Field,
  type: schematic.symbol.Paint["type"],
  builtIn: boolean,
): boolean => field === "stroke" || (builtIn && (type === "entry" || type === "node"));

/**
 * @returns the value of the state after its match. A number state with no value, or a
 * NaN value, reads its lower bound. An enumeration state with no value, or a NaN value,
 * reads its default entry, which needs no match.
 */
export const matched = (state: schematic.symbol.State, value: StateValues): number => {
  const v = typeof value === "number" ? value : value.get(state.key);
  if (v == null || Number.isNaN(v))
    return state.type === "enumeration" ? state.default : state.bounds.lower;
  if (state.type !== "enumeration" || state.match.type === "exact") return v;
  const { lower, upper } = state.match.bounds;
  return v >= lower && v < upper ? 1 : 0;
};

const entryColor = (
  state: schematic.symbol.State | undefined,
  ctx: PaintContext,
): color.Color | undefined => {
  if (state?.type !== "enumeration") return undefined;
  const entries =
    state.enumeration == null
      ? []
      : (ctx.enumerations.get(state.enumeration)?.entries ?? []);
  const value = matched(state, ctx.value);
  const entry =
    entries.find((e) => e.value === value) ??
    entries.find((e) => e.value === state.default);
  return (
    (entry?.color == null ? undefined : ctx.resolve(entry.color)) ??
    ctx.defaultEntryColor
  );
};

// The detail lines of built-in symbols draw contrast at 90% opacity. The steep ramp
// matches --pluto-symbol-contrast.
const contrastColor = (c: color.Color): color.Color => {
  const [l] = color.oklch(c);
  return color.fromOKLCH([bounds.clamp(bounds.DECIMAL, (0.6 - l) * 100), 0, 0, 0.9]);
};

const baseColor = (
  paint: schematic.symbol.Paint,
  states: Map<string, schematic.symbol.State>,
  ctx: PaintContext,
): color.Color | undefined => {
  switch (paint.type) {
    case "color":
      return ctx.resolve(paint.value);
    case "entry":
    case "contrast":
      return entryColor(states.get(paint.state), ctx);
    case "node":
      return ctx.nodeColor;
  }
};

// resolvePaint returns the color of paint on field as band shows it, or the stored
// color when band is absent.
const resolvePaint = (
  paint: schematic.symbol.Paint | undefined,
  field: Field,
  states: Map<string, schematic.symbol.State>,
  ctx: PaintContext,
  band?: Band,
): color.Color | undefined => {
  if (paint == null) return undefined;
  const c = baseColor(paint, states, ctx);
  if (c == null) return undefined;
  const show = (f: Field, type: schematic.symbol.Paint["type"]): color.Color =>
    band != null && banded(f, type, band.builtIn) ? theme.legible(c, band.theme) : c;
  // Contrast reads on the entry color as a fill of the drawing shows it.
  if (paint.type === "contrast") return contrastColor(show("fill", "entry"));
  return show(field, paint.type);
};

const applies = (
  look: schematic.symbol.Look,
  states: Map<string, schematic.symbol.State>,
  value: StateValues,
): boolean => {
  if (look.when == null) return true;
  const state = states.get(look.when.state);
  return state != null && look.when.values.includes(matched(state, value));
};

interface ResolvedLook {
  part?: string;
  stroke?: color.Color;
  fill?: color.Color;
  hidden?: boolean;
}

const resolveLooks = (
  spec: Pick<schematic.symbol.Spec, "states" | "looks">,
  ctx: PaintContext,
  band?: Band,
): ResolvedLook[] => {
  const states = new Map(spec.states.map((s) => [s.key, s]));
  return spec.looks
    .filter((look) => applies(look, states, ctx.value))
    .map((look) => ({
      part: look.part,
      stroke: resolvePaint(look.stroke, "stroke", states, ctx, band),
      fill: resolvePaint(look.fill, "fill", states, ctx, band),
      hidden: look.hidden,
    }));
};

// resolvePaints returns the paints of spec at ctx.value as band shows them, in the
// order that they apply. An override replaces the looks with its regions.
const resolvePaints = (
  spec: schematic.symbol.Spec,
  override: schematic.symbol.StateOverride | undefined,
  ctx: PaintContext,
  band: Band,
): Paint[] => {
  // An override color draws as a `color` paint does.
  const overridden = (field: Field, c: color.Color | undefined) =>
    c != null && banded(field, "color", band.builtIn)
      ? theme.legible(c, band.theme)
      : c;
  if (override != null)
    return override.regions.map(({ selectors, strokeColor, fillColor }) => ({
      selectors,
      stroke: overridden("stroke", strokeColor),
      fill: overridden("fill", fillColor),
    }));
  const parts = new Map(spec.parts.map((p) => [p.key, p]));
  return resolveLooks(spec, ctx, band).map(({ part, ...rest }) => ({
    selectors: part == null ? undefined : (parts.get(part)?.selectors ?? []),
    ...rest,
  }));
};

/**
 * @returns a region for each part of spec, with the colors that its looks give it at
 * ctx.value, before the theme band.
 */
export const resolveRegions = (
  spec: Pick<schematic.symbol.Spec, "parts" | "states" | "looks">,
  ctx: PaintContext,
): schematic.symbol.Region[] => {
  const looks = resolveLooks(spec, ctx);
  return spec.parts.map(({ key, name, selectors }) => {
    const region: schematic.symbol.Region = { key, name, selectors };
    for (const look of looks) {
      if (look.part != null && look.part !== key) continue;
      region.strokeColor = look.stroke ?? region.strokeColor;
      region.fillColor = look.fill ?? region.fillColor;
    }
    return region;
  });
};

/** Value is a value of a symbol's state that its node draws. */
export interface Value {
  key: string;
  value: number;
  name: string;
}

const UNNAMED: Value[] = [
  { key: "base", value: 0, name: "Base" },
  { key: "active", value: 1, name: "Active" },
];

/**
 * @returns the values that the node of spec draws: Base and Active for an actuator, and
 * Base for a static symbol. A value takes the key and name of its entry in entries.
 */
export const listValues = (
  spec: Pick<schematic.symbol.Spec, "variant">,
  entries: enumeration.Entry[] = [],
): Value[] =>
  UNNAMED.slice(0, spec.variant === "actuator" ? 2 : 1).map((v) => {
    const entry = entries.find((e) => e.value === v.value);
    return entry == null ? v : { key: entry.key, value: v.value, name: entry.name };
  });

const enumerationKeys = (
  spec: Pick<schematic.symbol.Spec, "states"> | undefined,
): enumeration.Key[] =>
  (spec?.states ?? []).flatMap((s) =>
    s.type === "enumeration" && s.enumeration != null ? [s.enumeration] : [],
  );

/**
 * @returns the context that resolves the looks of spec, with no value, or undefined
 * while the enumerations of its states load. An enumeration that fails to load draws no
 * entry colors.
 */
export const usePaintContext = (
  spec: Pick<schematic.symbol.Spec, "states"> | undefined,
  nodeColor?: color.Crude,
): Omit<PaintContext, "value"> | undefined => {
  const resolve = Color.useResolve();
  const enumerations = useEnumerations(spec);
  if (enumerations == null) return undefined;
  return {
    resolve,
    enumerations,
    nodeColor: nodeColor == null ? undefined : color.construct(nodeColor),
  };
};

// useEnumerations returns the enumerations of the states of spec by key, or undefined
// while they load. An enumeration that fails to load is absent.
const useEnumerations = (
  spec: Pick<schematic.symbol.Spec, "states"> | undefined,
): ReadonlyMap<enumeration.Key, enumeration.Enumeration> | undefined => {
  const keys = enumerationKeys(spec);
  const res = Enumeration.useResultMultiple(keys.length === 0 ? null : { keys });
  if (keys.length > 0 && res.variant === "loading") return undefined;
  return new Map((res.variant === "success" ? res.data : []).map((e) => [e.key, e]));
};

/**
 * @returns whether a state that has a value in value and an enumeration finds no entry
 * of that enumeration after its match. It is false while the enumerations load.
 */
export const useUnmatched = (
  spec: Pick<schematic.symbol.Spec, "states"> | undefined,
  value: StateValue,
): boolean => {
  const enumerations = useEnumerations(spec);
  if (spec == null || enumerations == null || typeof value === "number") return false;
  return spec.states.some((state) => {
    if (state.type !== "enumeration" || state.enumeration == null) return false;
    if (!value.has(state.key)) return false;
    const v = matched(state, value);
    const entries = enumerations.get(state.enumeration)?.entries ?? [];
    return !entries.some((e) => e.value === v);
  });
};

/**
 * @returns a region for each part of spec, with the colors that its looks give it at
 * value, or undefined while spec or the enumerations of its states load.
 */
export const useRegions = (
  spec: Pick<schematic.symbol.Spec, "parts" | "states" | "looks"> | undefined,
  value: number,
): schematic.symbol.Region[] | undefined => {
  const ctx = usePaintContext(spec);
  return useMemoDeepEqual(
    spec == null || ctx == null ? undefined : resolveRegions(spec, { ...ctx, value }),
  );
};

const NO_PAINTS: Paint[] = [];

export interface UsePaintsParams {
  /** override replaces the looks with its regions. */
  override?: schematic.symbol.StateOverride;
  /** nodeColor is the color that `node` paint draws. */
  nodeColor?: color.Crude;
  /**
   * builtIn marks a drawing that ships with Pluto. It bands its entry and node fills,
   * and an entry with no color draws the node color, else the theme gray.
   */
  builtIn?: boolean;
}

/**
 * @returns the paints of spec at value as the theme shows them, kept equal across
 * renders until a color that they draw changes, or none while spec loads. It loads the
 * enumerations of the states and the named colors.
 */
export const usePaints = (
  spec: schematic.symbol.Spec | undefined,
  value: StateValues,
  { override, nodeColor, builtIn = false }: UsePaintsParams = {},
): Paint[] => {
  const ctx = usePaintContext(spec, nodeColor);
  const t = Theming.use();
  const defaultEntryColor = builtIn ? (ctx?.nodeColor ?? t.colors.gray.l11) : undefined;
  return useMemoDeepEqual(
    spec == null || ctx == null
      ? NO_PAINTS
      : resolvePaints(
          spec,
          override,
          { ...ctx, value, defaultEntryColor },
          { theme: t, builtIn },
        ),
  );
};
