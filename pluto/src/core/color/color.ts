// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

const hexRegex = /^#?([0-9a-f]{6}|[0-9a-f]{8})$/i;
const hex = z.string().regex(hexRegex);
const rgbValue = z.number().min(0).max(255);
const alpha = z.number().min(0).max(1);
const rgba = z.tuple([rgbValue, rgbValue, rgbValue, alpha]);
const rgb = z.tuple([rgbValue, rgbValue, rgbValue]);

export type RGBA = [number, number, number, number];
export type RGB = [number, number, number];
export type Hex = z.infer<typeof hex>;
const crudeColor = z.object({ rgba255: rgba });
type CrudeColor = z.infer<typeof crudeColor>;

export type ColorT = Hex | RGBA | Color | string | RGB | CrudeColor;

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
  constructor(color: ColorT, alpha: number = 1) {
    if (typeof color === "string") {
      this.rgba255 = Color.fromHex(color, alpha);
    } else if (Array.isArray(color)) {
      if (color.length < 3 || color.length > 4)
        throw new Error(`Invalid color: [${color.join(", ")}]`);
      this.rgba255 = color.length === 3 ? [...color, alpha ?? 1] : color;
    } else this.rgba255 = color.rgba255;
  }

  /**
   * @returns true if the given color is semantically equal to this color. Different
   * representations of the same color are considered equal (e.g. hex and rgba).
   */
  equals(other: ColorT): boolean {
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
    return `#${toHex(r)}${toHex(g)}${toHex(b)}${a === 1 ? "" : toHex(a * 255)}`;
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
    return [
      this.rgba255[0] / 255,
      this.rgba255[1] / 255,
      this.rgba255[2] / 255,
      this.rgba255[3],
    ];
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
    if (alpha > 1) alpha = alpha / 100;
    return new Color([r, g, b, alpha]);
  }

  /** A totally zero color with no alpha. */
  static ZERO = new Color([0, 0, 0, 0]);

  static readonly z = z
    .union([hex, rgba, rgb, z.instanceof(Color), crudeColor])
    .transform((v) => new Color(v as string));

  private static fromHex(hex_: string, alpha: number = 1): RGBA {
    const valid = hex.safeParse(hex_);
    if (!valid.success) throw new Error(`Invalid hex color: ${hex_}`);
    hex_ = stripHash(hex_);
    return [
      fromHex(hex_, 0),
      fromHex(hex_, 2),
      fromHex(hex_, 4),
      hex_.length === 8 ? fromHex(hex_, 6) / 255 : alpha,
    ];
  }
}

const toHex = (n: number): string => Math.floor(n).toString(16).padStart(2, "0");
const fromHex = (s: string, n: number): number => parseInt(s.slice(n, n + 2), 16);
const stripHash = (hex: string): string => (hex.startsWith("#") ? hex.slice(1) : hex);
