// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type color as colorRef,
  type enumeration,
  type schematic,
} from "@synnaxlabs/client";
import { useMemoDeepEqual } from "@synnaxlabs/lyra/memo";
import { color } from "@synnaxlabs/x";

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

/** PaintContext holds what a look needs to resolve its colors. */
export interface PaintContext {
  /** value is the value of each state of the symbol. */
  value: number;
  resolve: (ref: colorRef.Ref) => color.Color | undefined;
  enumerations: ReadonlyMap<enumeration.Key, enumeration.Enumeration>;
  /** nodeColor is the color that `node` paint draws. */
  nodeColor?: color.Color;
}

const matched = (state: schematic.symbol.State, value: number): number => {
  if (state.type !== "enumeration" || state.match.type === "exact") return value;
  const { lower, upper } = state.match.bounds;
  return value >= lower && value < upper ? 1 : 0;
};

const entryColor = (
  state: schematic.symbol.State | undefined,
  ctx: PaintContext,
): color.Color | undefined => {
  if (state?.type !== "enumeration" || state.enumeration == null) return undefined;
  const entries = ctx.enumerations.get(state.enumeration)?.entries ?? [];
  const value = matched(state, ctx.value);
  const entry =
    entries.find((e) => e.value === value) ??
    entries.find((e) => e.value === state.default);
  return entry?.color == null ? undefined : ctx.resolve(entry.color);
};

const resolvePaint = (
  paint: schematic.symbol.Paint | undefined,
  states: Map<string, schematic.symbol.State>,
  ctx: PaintContext,
): color.Color | undefined => {
  switch (paint?.type) {
    case undefined:
      return undefined;
    case "color":
      return ctx.resolve(paint.value);
    case "entry":
      return entryColor(states.get(paint.state), ctx);
    case "node":
      return ctx.nodeColor;
  }
};

const applies = (
  look: schematic.symbol.Look,
  states: Map<string, schematic.symbol.State>,
  value: number,
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
  spec: schematic.symbol.Spec,
  ctx: PaintContext,
): ResolvedLook[] => {
  const states = new Map(spec.states.map((s) => [s.key, s]));
  return spec.looks
    .filter((look) => applies(look, states, ctx.value))
    .map((look) => ({
      part: look.part,
      stroke: resolvePaint(look.stroke, states, ctx),
      fill: resolvePaint(look.fill, states, ctx),
      hidden: look.hidden,
    }));
};

// resolvePaints returns the paints of spec at ctx.value, in the order that they apply.
// A state override at index ctx.value replaces the looks with its regions.
const resolvePaints = (
  spec: schematic.symbol.Spec,
  overrides: schematic.symbol.StateOverride[] | undefined,
  ctx: PaintContext,
): Paint[] => {
  const override = overrides?.[ctx.value];
  if (override != null)
    return override.regions.map(({ selectors, strokeColor, fillColor }) => ({
      selectors,
      stroke: strokeColor,
      fill: fillColor,
    }));
  const parts = new Map(spec.parts.map((p) => [p.key, p]));
  return resolveLooks(spec, ctx).map(({ part, ...rest }) => ({
    selectors: part == null ? undefined : (parts.get(part)?.selectors ?? []),
    ...rest,
  }));
};

/**
 * @returns a region for each part of spec, with the colors that its looks give it at
 * ctx.value.
 */
export const resolveRegions = (
  spec: schematic.symbol.Spec,
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

const enumerationKeys = (spec: schematic.symbol.Spec | undefined): enumeration.Key[] =>
  (spec?.states ?? []).flatMap((s) =>
    s.type === "enumeration" && s.enumeration != null ? [s.enumeration] : [],
  );

/**
 * @returns the context that resolves the looks of spec, with no value, or undefined
 * while the enumerations of its states load. An enumeration that fails to load draws
 * no entry colors.
 */
export const usePaintContext = (
  spec: schematic.symbol.Spec | undefined,
  nodeColor?: color.Crude,
): Omit<PaintContext, "value"> | undefined => {
  const resolve = Color.useResolve();
  const keys = enumerationKeys(spec);
  const res = Enumeration.useResultMultiple(keys.length === 0 ? null : { keys });
  if (keys.length > 0 && res.variant === "loading") return undefined;
  return {
    resolve,
    enumerations: new Map(
      (res.variant === "success" ? res.data : []).map((e) => [e.key, e]),
    ),
    nodeColor: nodeColor == null ? undefined : color.construct(nodeColor),
  };
};

const NO_PAINTS: Paint[] = [];

/**
 * @returns the paints of spec at value, kept equal across renders until a color that
 * they draw changes, or none while spec loads. It loads the enumerations of the states
 * and the named colors.
 */
export const usePaints = (
  spec: schematic.symbol.Spec | undefined,
  value: number,
  overrides?: schematic.symbol.StateOverride[],
  nodeColor?: color.Crude,
): Paint[] => {
  const ctx = usePaintContext(spec, nodeColor);
  return useMemoDeepEqual(
    spec == null || ctx == null
      ? NO_PAINTS
      : resolvePaints(spec, overrides, { ...ctx, value }),
  );
};
