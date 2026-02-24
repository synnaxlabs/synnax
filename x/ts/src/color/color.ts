// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

/** A regex to match hex colors. */
const hexRegex = /^#?([0-9a-f]{6}|[0-9a-f]{8})$/i;

/** A zod schema for a hex color. */
const hexZ = z.string().regex(hexRegex);
/** A zod schema for an RGB value. */
const rgbValueZ = z.number().min(0).max(255);
/** A zod schema for an alpha value between 0 and 1. */
const alphaZ = z.number().min(0).max(1);
/** A zod schema for an RGBA color. */
const rgbaZ = z.tuple([rgbValueZ, rgbValueZ, rgbValueZ, alphaZ]);
/** A zod schema for an RGB color. */
const rgbZ = z.tuple([rgbValueZ, rgbValueZ, rgbValueZ]);
/** A zod schema for a legacy color object. */
const legacyObjectZ = z.object({ rgba255: rgbaZ });
/** A zod schema for an RGBA struct (r, g, b, a fields). */
const rgbaStructZ = z.object({ r: rgbValueZ, g: rgbValueZ, b: rgbValueZ, a: alphaZ });
/** An RGBA struct with named fields. */
export type RGBAStruct = z.infer<typeof rgbaStructZ>;
/** A zod schema for a hue value between 0 and 360. */
const hueZ = z.number().min(0).max(360);
/** A zod schema for a saturation value between 0 and 100. */
const saturationZ = z.number().min(0).max(100);
/** A zod schema for a lightness value between 0 and 100. */
const lightnessZ = z.number().min(0).max(100);
/** A zod schema for an HSLA color. */
const hslaZ = z.tuple([hueZ, saturationZ, lightnessZ, alphaZ]);

/** A color in RGBA format. See https://en.wikipedia.org/wiki/RGBA_color_model */
export type RGBA = z.infer<typeof rgbaZ>;
/** A color in HSLA format. See https://en.wikipedia.org/wiki/HSL_and_HSV */
export type HSLA = z.infer<typeof hslaZ>;
/** A color in RGB format. See https://en.wikipedia.org/wiki/RGB_color_model */
export type RGB = z.infer<typeof rgbZ>;
/** A color in hex format. See https://en.wikipedia.org/wiki/Web_colors */
export type Hex = z.infer<typeof hexZ>;

/** A legacy color object. Used for backwards compatibility. */
type LegacyObject = z.infer<typeof legacyObjectZ>;

/** A zod schema for a crude color representation. */
export const crudeZ = z.union([hexZ, rgbZ, rgbaZ, hslaZ, legacyObjectZ, rgbaStructZ]);
/**
 * An unparsed representation of a color i.e. a value that can be converted into
 * a Color object.
 */
export type Crude = Hex | RGBA | Color | RGB | LegacyObject | RGBAStruct;

/** A zod schema to parse color values from various crude representations. */
export const colorZ = crudeZ.transform((v) => construct(v));

/**
 * A color in RGBA format. Used as the standard representation of a color in this package.
 * See https://en.wikipedia.org/wiki/RGBA_color_model
 */
export type Color = RGBA;

/** @returns true if the given color can be parsed into a valid color object. */
export const isCrude = (color: unknown): color is Crude =>
  colorZ.safeParse(color).success;

/** @returns true if the color is a true Color type. */
export const isColor = (color: unknown): color is Color =>
  rgbaZ.safeParse(color).success;

/**
 * Converts a crude color to its most meaningful CSS format.
 * @returns undefined if the color is undefined.
 * @returns an RGBA CSS string if the color can be parsed into a Color.
 * @returns the color directly if it is a css variable.
 * @throws if the color does not match any of the preceding conditions.
 */
export interface CSSString {
  (color: Crude): string;
  (color?: Crude): string | undefined;
}

export const cssString = ((color?: Crude): string | undefined => {
  if (color == null) return undefined;
  const res = colorZ.safeParse(color);
  if (res.success) return rgbaCSS(res.data);
  if (typeof color === "string") return color;
  throw res.error;
}) as CSSString;

/**
 * @constructor Creates a new color from the given color value. The color value can be
 * a hex string, an array of RGB or RGBA values, or another color.
 *
 * @param color - The color value to create the color from. If the color value is a
 * string, it must be a valid hex color (with or without the '#') with a hash-less
 * length 6 or 8. If the hex color is 8 characters long, the last two characters are
 * used as the alpha value. If the color value is an array, it must be an array of
 * length 3 or 4, with each value between 0 and 255. If the color value is another
 * color, the color will be copied.
 *
 * @param alpha - An optional alpha value to set. If the color value carries its own
 * alpha value, this value will be ignored. Defaults to 1.
 */
export const construct = (color: Crude, alpha: number = 1): Color => {
  color = crudeZ.parse(color);
  if (typeof color === "string") return fromHex(color, alpha);
  if (Array.isArray(color)) {
    if (color.length < 3 || color.length > 4)
      throw new Error(`Invalid color: [${color.join(", ")}]`);
    if (color.length === 3) return [...color, alpha];
    return color;
  }
  if ("a" in color && "r" in color) return [color.r, color.g, color.b, color.a];
  return color.rgba255;
};

/**
 * @returns true if the given color is semantically equal to this color. Different
 * representations of the same color are considered equal (e.g. hex and rgba).
 */
export const equals = (a?: Crude, b?: Crude): boolean => {
  if (a == null || b == null) return a == b;
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
  const alphaByte = Math.round(a * 255);
  return `#${rgbaToHex(r)}${rgbaToHex(g)}${rgbaToHex(b)}${
    alphaByte === 255 ? "" : rgbaToHex(alphaByte)
  }`;
}) as ToHex;

/** @returns the color as a CSS RGBA string. i.e. rgba(r, g, b, a) */
export const rgbaCSS = (color: Crude): string => {
  const [r, g, b, a] = construct(color);
  return `rgba(${r}, ${g}, ${b}, ${a})`;
};

/** @returns the color as a CSS RGB string with no alpha value. i.e. rgb(r, g, b) */
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

/** @returns the color normalized as an RGB tuple between 0 and 1. */
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
 * @returns A new color with the given alpha.
 * @param color - The color to set the alpha value on.
 * @param alpha - The alpha value to set. If the value is greater than 1, it will be
 * divided by 100.
 */
export const setAlpha = (color: Crude, alpha: number): Color => {
  const [r, g, b] = construct(color);
  if (alpha > 100)
    throw new Error(`Color opacity must be between 0 and 100, got ${alpha}`);
  if (alpha > 1) alpha /= 100;
  return [r, g, b, alpha];
};
/**
 * @returns the luminance of the color, between 0 and 1.
 * @see https://en.wikipedia.org/wiki/Relative_luminance for more information.
 */
export const luminance = (color: Crude): number => {
  const [r, g, b] = rgb1(color).map((v) =>
    v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4,
  );
  return Number((0.2126 * r + 0.7152 * g + 0.0722 * b).toFixed(3));
};

/**
 * @returns an approximation of the colors 'grayness' from 0 to 1 by measuring the
 * deviation between the RGB values of the color.
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
 * @param a - The first color to compare.
 * @param b - The second color to compare.
 * @returns The contrast ratio between the two colors.
 *
 * @see https://www.accessibility-developer-guide.com/knowledge/colours-and-contrast/how-to-calculate/
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
  const [best] = colors.sort((a, b) => contrast(source_, b) - contrast(source_, a));
  return construct(best);
};

/** @returns true if the color is dark i.e. it has a luminance less than 0.5. */
export const isDark = (color: Crude): boolean => luminance(color) < 0.5;

/** @returns true if the color is light i.e. the luminance is greater than 0.5. */
export const isLight = (color: Crude): boolean => luminance(color) > 0.5;

/** @returns a color parsed from a hex string with an alpha value. */
const fromHex = (hex: string, alpha: number = 1): RGBA => {
  hex = hexZ.parse(hex);
  hex = stripHash(hex);
  return [
    hexToRgba(hex, 0),
    hexToRgba(hex, 2),
    hexToRgba(hex, 4),
    hex.length === 8 ? hexToRgba(hex, 6) / 255 : alpha,
  ];
};

/** A totally zero color with no alpha. */
export const ZERO: Color = [0, 0, 0, 0];

const rgbaToHex = (n: number): string => Math.floor(n).toString(16).padStart(2, "0");
const hexToRgba = (s: string, n: number): number => parseInt(s.slice(n, n + 2), 16);
const stripHash = (hex: string): string => (hex.startsWith("#") ? hex.slice(1) : hex);

const NAMED: Record<string, string> = {
  black: "#000000",
  white: "#ffffff",
  red: "#ff0000",
  green: "#008000",
  blue: "#0000ff",
  yellow: "#ffff00",
  cyan: "#00ffff",
  magenta: "#ff00ff",
  silver: "#c0c0c0",
  gray: "#808080",
  grey: "#808080",
  maroon: "#800000",
  olive: "#808000",
  lime: "#00ff00",
  aqua: "#00ffff",
  teal: "#008080",
  navy: "#000080",
  fuchsia: "#ff00ff",
  purple: "#800080",
  orange: "#ffa500",
  brown: "#a52a2a",
  tan: "#d2b48c",
  gold: "#ffd700",
  indigo: "#4b0082",
  violet: "#ee82ee",
  pink: "#ffc0cb",
  coral: "#ff7f50",
  salmon: "#fa8072",
  khaki: "#f0e68c",
  crimson: "#dc143c",
  transparent: "transparent",
};

/**
 * Parses a CSS color string into a Color.
 * Supports hex colors, rgb/rgba functions, and named colors.
 * @param cssColor - The CSS color string to parse
 * @returns The parsed color or undefined if invalid
 */
export const fromCSS = (cssColor: string): Color | undefined => {
  if (!cssColor) return undefined;
  const trimmed = cssColor.trim().toLowerCase();
  if (trimmed === "transparent" || trimmed === "none") return undefined;
  if (trimmed.startsWith("#")) {
    if (trimmed.length === 4) {
      const r = trimmed[1];
      const g = trimmed[2];
      const b = trimmed[3];
      const expanded = `#${r}${r}${g}${g}${b}${b}`;
      if (hexZ.safeParse(expanded).success) return fromHex(expanded);
    }
    if (
      (trimmed.length === 7 || trimmed.length === 9) &&
      hexZ.safeParse(trimmed).success
    )
      return fromHex(trimmed);
    return undefined;
  }
  if (trimmed.startsWith("rgb")) {
    const match = trimmed.match(
      /rgba?\s*\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*(?:,\s*([\d.]+))?\s*\)/,
    );
    if (match) {
      const [, r, g, b, a] = match;
      return [parseInt(r), parseInt(g), parseInt(b), a ? parseFloat(a) : 1];
    }
  }
  if (NAMED[trimmed]) return fromHex(NAMED[trimmed]);
  return undefined;
};

/** @returns parse a color from an HSLA tuple. */
export const fromHSLA = (hsla: HSLA): RGBA => {
  hsla = hslaZ.parse(hsla);
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
    r = hueToRGB(p, q, h + 1 / 3);
    g = hueToRGB(p, q, h);
    b = hueToRGB(p, q, h - 1 / 3);
  }

  return [Math.round(r * 255), Math.round(g * 255), Math.round(b * 255), a];
};

const hueToRGB = (p: number, q: number, t: number): number => {
  if (t < 0) t += 1;
  if (t > 1) t -= 1;
  if (t < 1 / 6) return p + (q - p) * 6 * t;
  if (t < 1 / 2) return q;
  if (t < 2 / 3) return p + (q - p) * (2 / 3 - t) * 6;
  return p;
};

const rgbaToHSLA = (rgba: RGBA): HSLA => {
  rgba = rgbaZ.parse(rgba);
  let [r, g, b] = rgba;
  const a = rgba[3];
  r /= 255;
  g /= 255;
  b /= 255;

  const max = Math.max(r, g, b);
  const min = Math.min(r, g, b);
  let h: number;
  let s: number;
  let l = (max + min) / 2;

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

/** The color black. */
export const BLACK = construct("#000000");
/** The color white. */
export const WHITE = construct("#ffffff");
