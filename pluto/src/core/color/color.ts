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
const rgba = z.array(z.number()).length(4).min(0).max(255);

export type RGBA = [number, number, number, number];
export type Hex = z.infer<typeof hex>;

export type ColorT = Hex | RGBA | Color | string;

const invalidHexError = (hex: string): Error => new Error(`Invalid hex color: ${hex}`);

const stripHash = (hex: string): string => (hex.startsWith("#") ? hex.slice(1) : hex);

const p = (s: string, n: number): number => parseInt(s.slice(n, n + 2), 16);

export class Color {
  private readonly internal: RGBA;

  static get zero(): Color {
    return new Color([0, 0, 0, 0]);
  }

  constructor(color: ColorT, alpha?: number) {
    if (color instanceof Color) {
      this.internal = color.internal;
    } else if (typeof color === "string") {
      this.internal = Color.fromHex(color, alpha);
    } else {
      if (color.length < 3 || color.length > 4)
        throw new Error(`Invalid color: [${color.join(", ")}]`);
      this.internal = color;
    }
  }

  equals(other: Color): boolean {
    return this.internal.every((v, i) => v === other.internal[i]);
  }

  private static fromHex(hex_: string, alpha: number = 1): RGBA {
    const valid = hex.safeParse(hex_);
    if (!valid.success) throw invalidHexError(hex_);
    hex_ = stripHash(hex_);
    return [
      p(hex_, 0),
      p(hex_, 2),
      p(hex_, 4),
      hex_.length === 8 ? p(hex_, 6) / 255 : alpha,
    ];
  }

  /*
   * Returns the hex representation of the color. If the color has an opacity of 1,
   * the returned hex will be 6 characters long. Otherwise, it will be 8 characters
   * long.
   */
  get hex(): string {
    const [r, g, b, a] = this.internal;
    return `#${toHex(r)}${toHex(g)}${toHex(b)}${a === 1 ? "" : toHex(a)}`;
  }

  /*
   *Returns the color as an RGBA tuple.
   */
  get rgba255(): RGBA {
    return this.internal;
  }

  get rgba1(): RGBA {
    return [
      this.internal[0] / 255,
      this.internal[1] / 255,
      this.internal[2] / 255,
      this.internal[3],
    ];
  }

  setOpacity(opacity: number): Color {
    const [r, g, b] = this.internal;
    if (opacity > 1) opacity = opacity / 100;
    return new Color([r, g, b, opacity]);
  }

  static readonly z = z
    .union([hex, rgba, z.instanceof(Color)])
    .transform((v) => new Color(v as string));
}

const toHex = (n: number): string => n.toString(16).padStart(2, "0");
