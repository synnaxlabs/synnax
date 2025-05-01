// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

const hexRegex = /^#?([0-9a-f]{6}|[0-9a-f]{8})$/i;
const hexZ = z.string().regex(hexRegex);
const rgbValueZ = z.number().min(0).max(255);
const alphaZ = z.number().min(0).max(1);
const rgbaZ = z.tuple([rgbValueZ, rgbValueZ, rgbValueZ, alphaZ]);
const rgbZ = z.tuple([rgbValueZ, rgbValueZ, rgbValueZ]);

export type RGBA = [number, number, number, number];
export type HSLA = [number, number, number, number];
export type RGB = [number, number, number];
export type Hex = z.infer<typeof hexZ>;

export const isCrude = (color: unknown): color is Crude =>
  colorZ.safeParse(color).success;

export const isColor = (color: unknown): color is Color =>
  rgbaZ.safeParse(color).success;

/**
 * An unparsed representation of a color i.e. a value that can be converted into
 * a Color object.
 */
export type Crude = Hex | RGBA | Color | string | RGB;

/**
 * Converts a crude color to its CSS representation. If the color cannot be parsed,
 *
 *
 * @param color -
 */
export const cssString = (color?: Crude): string | undefined => {
  if (color == null) return undefined;
  const res = colorZ.safeParse(color);
  if (res.success) return rgbaCSS(res.data);
  if (typeof color === "string") return color;
  throw res.error;
};

export type Color = RGBA;

/**
 * @constructor Creates a new color from the given color value. The color value can be
 * a hex string, an array of RGB or RGBA values, or another color.
 *
 * @param color - The color value to create the color from. If the color value is a
 * string, it must be a valid hex color (with or without the '#') with a hasheless
 * length 6 or 8. If the hex color is 8 characters long, the last twoc haracters are
 * used as the alpha value. If the color value is an array, it must be an array of
 * length 3 or 4, with each value between 0 and 255. If the color value is another
 * color, the color will be copied.
 *
 * @param alpha - An optional alpha value to set. If the color value carries its own
 * alpha value, this value will be ignored. Defaults to 1.
 */
export const construct = (color: Crude, alpha: number = 1): Color => {
  if (typeof color === "string") return fromHex(color, alpha);
  if (Array.isArray(color)) {
    if (color.length < 3 || color.length > 4)
      throw new Error(`Invalid color: [${color.join(", ")}]`);
    if (color.length === 3) return [...color, alpha] as RGBA;
    return color;
  }
  throw new Error(`Invalid color: ${JSON.stringify(color)}`);
};

/**
 * @returns true if the given color is semantically equal to this color. Different
 * representations of the same color are considered equal (e.g. hex and rgba).
 */
export const equals = (a?: Crude, b?: Crude): boolean => {
  if (a == null || b == null) return false;
  const a_ = construct(a);
  const b_ = construct(b);
  return a_.every((v, i) => v === b_[i]);
};

export interface ToHex {
  (color: Crude): string;
  (color?: Crude): string | undefined;
}

/**
 * @returns the hex representation of the color. If the color has an opacity of 1,
 * the returned hex will be 6 characters long. Otherwise, it will be 8 characters
 * long.
 */
export const hex = ((color?: Crude) => {
  if (color == null) return undefined;
  const [r, g, b, a] = construct(color);
  return `#${rgbaToHex(r)}${rgbaToHex(g)}${rgbaToHex(b)}${
    a === 1 ? "" : rgbaToHex(a * 255)
  }`;
}) as ToHex;

/**
 * @returns the color as a CSS RGBA string.
 */
export const rgbaCSS = (color: Crude): string => {
  const [r, g, b, a] = construct(color);
  return `rgba(${r}, ${g}, ${b}, ${a})`;
};

/**
 * @returns the color as a CSS RGB string with no alpha value.
 */
export const rgbCSS = (color: Crude): string => `rgb(${rgbString(color)})`;

/**
 * @returns the color as an RGB string, with each color value between 0 and 255.
 * @example "255, 255, 255"
 */
export const rgbString = (color: Crude): string => {
  const [r, g, b] = construct(color);
  return `${r}, ${g}, ${b}`;
};

/**
 * @returns the color as an RGBA tuple, with each color value between 0 and 1,
 * and the alpha value between 0 and 1.
 */
export const rgba1 = (color: Crude): RGBA => [...rgb1(color), aValue(color)];

const rgb1 = (color: Crude): RGB => [
  rValue(color) / 255,
  gValue(color) / 255,
  bValue(color) / 255,
];

/** @returns the red value of the color, between 0 and 255. */
export const rValue = (color: Crude): number => construct(color)[0];

/** @returns the green value of the color, between 0 and 255. */
export const gValue = (color: Crude): number => construct(color)[1];

/** @returns the blue value of the color, between 0 and 255. */
export const bValue = (color: Crude): number => construct(color)[2];

/** @returns the alpha value of the color, between 0 and 1. */
export const aValue = (color: Crude): number => construct(color)[3];

/** @returns true if all RGBA values are 0. */
export const isZero = (color?: Crude): boolean => equals(ZERO, color);

/** @returns the HSLA representation of the color. */
export const hsla = (color: Crude): HSLA => rgbaToHSLA(construct(color));

/**
 * Creates a new color with the given alpha.
 *
 * @param alpha - The alpha value to set. If the value is greater than 1, it will be
 * divided by 100.
 * @returns A new color with the given alpha.
 */
export const setAlpha = (color: Crude, alpha: number): Color => {
  const [r, g, b] = construct(color);
  if (alpha > 100)
    throw new Error(`Color opacity must be between 0 and 100, got ${alpha}`);
  if (alpha > 1) alpha /= 100;
  return [r, g, b, alpha] as Color;
};

/**
 * @returns the luminance of the color, between 0 and 1.
 */
export const luminance = (color: Crude): number => {
  const [r, g, b] = rgb1(color).map((v) =>
    v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4,
  );
  return Number((0.2126 * r + 0.7152 * g + 0.0722 * b).toFixed(3));
};

/**
 * @returns the grayness of the color, between 0 and 1.
 */
export const grayness = (color: Crude): number => {
  const [r, g, b] = rgb1(color);
  const deviation = Math.max(r, g, b) - Math.min(r, g, b);
  return 1 - deviation;
};

/**
 * @returns the contrast ratio between this color and the given color. The contrast
 * ratio is a number between 1 and 21, where 1 is the lowest contrast and 21 is the
 * highest.
 * @param other
 * @returns
 */
export const contrast = (a: Crude, b: Crude): number => {
  const a_ = construct(a);
  const b_ = construct(b);
  const l1 = luminance(a_);
  const l2 = luminance(b_);
  return (Math.max(l1, l2) + 0.5) / (Math.min(l1, l2) + 0.5);
};

/**
 * @returns the color with the highest contrast ratio to the given colors.
 */
export const pickByContrast = (source: Crude, ...colors: Crude[]): Color => {
  if (colors.length === 0)
    throw new Error("[Color.pickByContrast] - must provide at least one color");
  const source_ = construct(source);
  const [best] = colors
    .map((c) => construct(c))
    .sort((a, b) => contrast(source_, b) - contrast(source_, a));
  return best;
};

/**
 * @returns true if the color is dark.
 */
export const isDark = (color: Crude): boolean => luminance(color) < 0.5;

/**
 * @returns true if the color is light.
 */
export const isLight = (color: Crude): boolean => !isDark(color);

export const colorZ = z
  .union([hexZ, rgbaZ, rgbZ])
  .transform((v) => construct(v as string));

const fromHex = (hex_: string, alpha: number = 1): RGBA => {
  const valid = hexZ.safeParse(hex_);
  if (!valid.success) throw new Error(`Invalid hex color: ${hex_}`);
  hex_ = stripHash(hex_);
  return [
    hexToRgba(hex_, 0),
    hexToRgba(hex_, 2),
    hexToRgba(hex_, 4),
    hex_.length === 8 ? hexToRgba(hex_, 6) / 255 : alpha,
  ];
};

/** A totally zero color with no alpha. */
export const ZERO: Color = [0, 0, 0, 0];

const rgbaToHex = (n: number): string => Math.floor(n).toString(16).padStart(2, "0");
const hexToRgba = (s: string, n: number): number => parseInt(s.slice(n, n + 2), 16);
const stripHash = (hex: string): string => (hex.startsWith("#") ? hex.slice(1) : hex);

export const fromHSLA = (hsla: RGBA): RGBA => {
  let [h, s, l] = hsla;
  const a = hsla[3];
  h /= 360;
  s /= 100;
  l /= 100;
  let r, g, b;

  if (s === 0)
    r = g = b = l; // achromatic
  else {
    const q = l < 0.5 ? l * (1 + s) : l + s - l * s;
    const p = 2 * l - q;
    r = hueToRgb(p, q, h + 1 / 3);
    g = hueToRgb(p, q, h);
    b = hueToRgb(p, q, h - 1 / 3);
  }

  return [Math.round(r * 255), Math.round(g * 255), Math.round(b * 255), a];
};

const hueToRgb = (p: number, q: number, t: number): number => {
  if (t < 0) t += 1;
  if (t > 1) t -= 1;
  if (t < 1 / 6) return p + (q - p) * 6 * t;
  if (t < 1 / 2) return q;
  if (t < 2 / 3) return p + (q - p) * (2 / 3 - t) * 6;
  return p;
};

const rgbaToHSLA = (rgba: RGBA): HSLA => {
  let [r, g, b] = rgba;
  const a = rgba[3];
  r /= 255;
  g /= 255;
  b /= 255;

  r /= 255;
  g /= 255;
  b /= 255;

  const max = Math.max(r, g, b);
  const min = Math.min(r, g, b);
  let h: number;
  let s: number;
  let l: number = (max + min) / 2;

  if (max === min)
    h = s = 0; // achromatic
  else {
    const d = max - min;
    s = l > 0.5 ? d / (2 - max - min) : d / (max + min);
    if (max === r) h = (g - b) / d + (g < b ? 6 : 0);
    else if (max === g) h = (b - r) / d + 2;
    else h = (r - g) / d + 4;
    h /= 6;
  }

  // Convert hue to degrees
  h *= 360;
  s *= 100;
  l *= 100;

  return [Math.round(h), Math.round(s), Math.round(l), a];
};

export const crudeZ = z.union([hexZ, rgbaZ, z.string(), rgbZ]);
