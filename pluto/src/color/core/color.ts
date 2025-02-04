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
const crudeColor = z.object({ rgba255: rgbaZ });
type CrudeBase = z.infer<typeof crudeColor>;

export const toHex = ((color?: Crude): string | undefined =>
  color == null ? undefined : new Color(color).hex) as ((color: Crude) => string) &
  ((color?: Crude) => string | undefined);

/**
 * An unparsed representation of a color i.e. a value that can be converted into
 * a Color object.
 */
export type Crude = Hex | RGBA | Color | string | RGB | CrudeBase;

/**
 * Converts a crude color to its CSS representation. If the color cannot be parsed,
 *
 *
 * @param color -
 */
export const cssString = (color?: Crude): string | undefined => {
  if (color == null) return undefined;
  const res = Color.z.safeParse(color);
  if (res.success) return res.data.rgbaCSS;
  if (typeof color === "string") return color;
  throw res.error;
};

/**
 * A color with an alpha channel. It can be used to easily transform
 * color values from one format to another, as well as make modifications to the color.
 */
export class Color {
  /**
   * @property the color as an RGBA tuple, with each color value between 0 and 255,
   * and the alpha value between 0 and 1.
   */
  readonly rgba255: RGBA;

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
  constructor(color: Crude, alpha: number = 1) {
    if (typeof color === "string") this.rgba255 = Color.fromHex(color, alpha);
    else if (Array.isArray(color)) {
      if (color.length < 3 || color.length > 4)
        throw new Error(`Invalid color: [${color.join(", ")}]`);
      this.rgba255 = color.length === 3 ? [...color, alpha ?? 1] : color;
    } else this.rgba255 = color.rgba255;
  }

  /**
   * @returns true if the given color is semantically equal to this color. Different
   * representations of the same color are considered equal (e.g. hex and rgba).
   */
  equals(other?: Crude): boolean {
    if (other == null) return false;
    const other_ = new Color(other);
    return this.rgba255.every((v, i) => v === other_.rgba255[i]);
  }

  /**
   * @returns the hex representation of the color. If the color has an opacity of 1,
   * the returned hex will be 6 characters long. Otherwise, it will be 8 characters
   * long.
   */
  get hex(): string {
    const [r, g, b, a] = this.rgba255;
    return `#${rgbaToHex(r)}${rgbaToHex(g)}${rgbaToHex(b)}${
      a === 1 ? "" : rgbaToHex(a * 255)
    }`;
  }

  /**
   * @returns the color as a CSS RGBA string.
   */
  get rgbaCSS(): string {
    const [r, g, b, a] = this.rgba255;
    return `rgba(${r}, ${g}, ${b}, ${a})`;
  }

  /**
   * @returns the color as a CSS RGB string with no alpha value.
   */
  get rgbCSS(): string {
    return `rgb(${this.rgbString})`;
  }

  /**
   * @returns the color as an RGB string, with each color value between 0 and 255.
   * @example "255, 255, 255"
   */
  get rgbString(): string {
    const [r, g, b] = this.rgba255;
    return `${r}, ${g}, ${b}`;
  }

  /**
   * @returns the color as an RGBA tuple, with each color value between 0 and 1,
   * and the alpha value between 0 and 1.
   */
  get rgba1(): RGBA {
    return [...this.rgb1, this.rgba255[3]];
  }

  get rgb1(): RGB {
    return [this.rgba255[0] / 255, this.rgba255[1] / 255, this.rgba255[2] / 255];
  }

  /** @returns the red value of the color, between 0 and 255. */
  get r(): number {
    return this.rgba255[0];
  }

  /** @returns the green value of the color, between 0 and 255. */
  get g(): number {
    return this.rgba255[1];
  }

  /** @returns the blue value of the color, between 0 and 255. */
  get b(): number {
    return this.rgba255[2];
  }

  /** @returns the alpha value of the color, between 0 and 1. */
  get a(): number {
    return this.rgba255[3];
  }

  /** @returns true if all RGBA values are 0. */
  get isZero(): boolean {
    return this.equals(ZERO);
  }

  get hsla(): HSLA {
    return rgbaToHSLA(this.rgba255);
  }

  /**
   * Creates a new color with the given alpha.
   *
   * @param alpha - The alpha value to set. If the value is greater than 1, it will be
   * divided by 100.
   * @returns A new color with the given alpha.
   */
  setAlpha(alpha: number): Color {
    const [r, g, b] = this.rgba255;
    if (alpha > 100)
      throw new Error(`Color opacity must be between 0 and 100, got ${alpha}`);
    if (alpha > 1) alpha /= 100;
    return new Color([r, g, b, alpha]);
  }

  /**
   * @returns the luminance of the color, between 0 and 1.
   */
  get luminance(): number {
    const [r, g, b] = this.rgb1.map((v) =>
      v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4,
    );
    return Number((0.2126 * r + 0.7152 * g + 0.0722 * b).toFixed(3));
  }

  /**
   * @returns the grayness of the color, between 0 and 1.
   */
  get grayness(): number {
    const [r, g, b] = this.rgb1;
    const deviation = Math.max(r, g, b) - Math.min(r, g, b);
    return 1 - deviation;
  }

  /**
   * @returns the contrast ratio between this color and the given color. The contrast
   * ratio is a number between 1 and 21, where 1 is the lowest contrast and 21 is the
   * highest.
   * @param other
   * @returns
   */
  contrast(other: Crude): number {
    const other_ = new Color(other);
    const l1 = this.luminance;
    const l2 = other_.luminance;
    return (Math.max(l1, l2) + 0.5) / (Math.min(l1, l2) + 0.5);
  }

  pickByContrast(...colors: Crude[]): Color {
    if (colors.length === 0)
      throw new Error("[Color.pickByContrast] - must provide at least one color");
    const [best] = colors
      .map((c) => new Color(c))
      .sort((a, b) => this.contrast(b) - this.contrast(a));
    return best;
  }

  get isDark(): boolean {
    return this.luminance < 0.5;
  }

  get isLight(): boolean {
    return !this.isDark;
  }

  static readonly z = z
    .union([hexZ, rgbaZ, rgbZ, z.instanceof(Color), crudeColor])
    .transform((v) => new Color(v as string));

  private static fromHex(hex_: string, alpha: number = 1): RGBA {
    const valid = hexZ.safeParse(hex_);
    if (!valid.success) throw new Error(`Invalid hex color: ${hex_}`);
    hex_ = stripHash(hex_);
    return [
      hexToRgba(hex_, 0),
      hexToRgba(hex_, 2),
      hexToRgba(hex_, 4),
      hex_.length === 8 ? hexToRgba(hex_, 6) / 255 : alpha,
    ];
  }
}

/** A totally zero color with no alpha. */
export const ZERO = new Color([0, 0, 0, 0]);

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

export const crudeZ = z.union([
  hexZ,
  rgbaZ,
  z.instanceof(Color),
  z.string(),
  rgbZ,
  crudeColor,
]);
