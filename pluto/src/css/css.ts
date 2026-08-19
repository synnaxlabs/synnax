// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, direction, location, type spatial, type text } from "@synnaxlabs/x";
import clsx, { type ClassValue } from "clsx";
import { type CSSProperties } from "react";

import { newBEM } from "@/css/bem";
import { CSSGridBuilder } from "@/css/grid";
import { applyCSSVars, removeCSSVars } from "@/css/vars";
import { type Theming } from "@/theming";

const PREFIX = "pluto";

export const {
  /** @returns the class name for a block. */
  B,
  /** @returns the class name for an element of the enclosing block. */
  E,
  /** @returns the class name for a modifier. */
  M,
  /** @returns the class name for an element of the given block. */
  BE,
  /** @returns the class name for a modifier of the given block. */
  BM,
  /** @returns the class name for a modified element of the given block. */
  BEM,
  /** @returns the name of a custom property, including the leading dashes. */
  variable,
} = newBEM(PREFIX);

/** Joins class values into a single class name, dropping the falsy ones. */
export const cls = (...classes: ClassValue[]): string => clsx(...classes);

export const visible = (visible: boolean): string => M(visible ? "visible" : "hidden");

export const expanded = (expanded: boolean): string =>
  M(expanded ? "expanded" : "collapsed");

export const level = (level: text.Level): string => M(`level-${level}`);

export const loc = (l: location.Crude): string => M("location", location.construct(l));

export const align = (position: spatial.Alignment | ""): string => M(position);

export const dir = (dir?: direction.Crude): string | false =>
  dir != null && M("direction", direction.construct(dir));

export const disabled = (disabled?: boolean): string | false =>
  disabled === true && M("disabled");

export const bordered = (
  loc?: location.Crude | spatial.Alignment | boolean,
): string | false => {
  if (typeof loc === "boolean") return loc && M("bordered");
  return loc != null ? M(`bordered-${loc.toString()}`) : M("bordered");
};

export const noSelect = M("no-select");

export const selected = (selected: boolean): string | false =>
  selected && M("selected");

export const altColor = (secondary: boolean): string | false =>
  secondary && M("alt-color");

export const editable = (editable: boolean): string | false =>
  editable && M("editable");

export const applyVars = applyCSSVars;

export const removeVars = removeCSSVars;

export const newGridBuilder = (prefix?: string): CSSGridBuilder =>
  new CSSGridBuilder(prefix);

export const inheritDims = (inherit = true): string | false =>
  inherit && M("inherit-dims");

export const dropRegion = (active: boolean): string | false =>
  active && B("haul-drop-region");

export const px = (value: number): string => `${value}px`;

export function shade(value: Theming.Shade): string;
export function shade(value?: Theming.Shade): string | false;
export function shade(value?: Theming.Shade): string | false {
  return value != null && M(`shade-${value}`);
}

export const colorVar = (
  value?: false | Theming.Shade | color.Crude,
): string | undefined => {
  if (value == null || value === false) return undefined;
  if (typeof value === "number") return `var(--${PREFIX}-gray-l${value})`;
  return color.cssString(value);
};

export const levelSizeVar = (value: string): string => `var(--${PREFIX}-${value}-size)`;

/**
 * A style object that also accepts CSS custom properties. Use it in place of
 * `CSSProperties`, which rejects any `--` key.
 */
export type VarProperties = CSSProperties &
  Record<`--${string}`, string | number | undefined>;
