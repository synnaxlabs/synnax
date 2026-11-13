// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type theme } from "@synnaxlabs/lyra/theme";
import { type color } from "@synnaxlabs/x";

const toLinear = (c: number): number => {
  const v = c / 255;
  return v <= 0.04045 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4;
};

const fromLinear = (v: number): number => {
  const c = v <= 0.0031308 ? 12.92 * v : 1.055 * v ** (1 / 2.4) - 0.055;
  return Math.round(Math.min(1, Math.max(0, c)) * 255);
};

interface OKLCH {
  l: number;
  c: number;
  h: number;
}

const toOKLCH = ([r8, g8, b8]: color.Color): OKLCH => {
  const r = toLinear(r8);
  const g = toLinear(g8);
  const b = toLinear(b8);
  const l = Math.cbrt(0.4122214708 * r + 0.5363325363 * g + 0.0514459929 * b);
  const m = Math.cbrt(0.2119034982 * r + 0.6806995451 * g + 0.1073969566 * b);
  const s = Math.cbrt(0.0883024619 * r + 0.2817188376 * g + 0.6299787005 * b);
  const L = 0.2104542553 * l + 0.793617785 * m - 0.0040720468 * s;
  const A = 1.9779984951 * l - 2.428592205 * m + 0.4505937099 * s;
  const B = 0.0259040371 * l + 0.7827717662 * m - 0.808675766 * s;
  return { l: L, c: Math.hypot(A, B), h: Math.atan2(B, A) };
};

const fromOKLCH = ({ l: L, c, h }: OKLCH, alpha: number): color.Color => {
  const A = c * Math.cos(h);
  const B = c * Math.sin(h);
  const l = (L + 0.3963377774 * A + 0.2158037573 * B) ** 3;
  const m = (L - 0.1055613458 * A - 0.0638541728 * B) ** 3;
  const s = (L - 0.0894841775 * A - 1.291485548 * B) ** 3;
  return [
    fromLinear(4.0767416621 * l - 3.3077115913 * m + 0.2309699292 * s),
    fromLinear(-1.2684380046 * l + 2.6097574011 * m - 0.3413193965 * s),
    fromLinear(-0.0041960863 * l - 0.7034186147 * m + 1.707614701 * s),
    alpha,
  ];
};

/**
 * @returns the color with its OKLCH lightness clamped into the theme's symbol band, the
 * same rule as the `lyra-symbol-colored` CSS class, so a user color reads on the theme.
 */
export const legible = (c: color.Color, t: theme.Theme): color.Color => {
  const { lMin, lMax, lExtreme, cThresh } = t.symbol;
  const o = toOKLCH(c);
  const gray = Math.min(1, Math.max(0, (cThresh - o.c) / cThresh));
  const l = Math.min(lMax, Math.max(lMin, o.l)) * (1 - gray) + lExtreme * gray;
  return fromOKLCH({ ...o, l }, c[3]);
};

/** @returns the color with its OKLCH lightness set, keeping its chroma and hue. */
export const withLightness = (c: color.Color, l: number, alpha = 255): color.Color =>
  fromOKLCH({ ...toOKLCH(c), l }, alpha);

/** @returns fg composited over bg at the given alpha, as an opaque color. */
export const over = (fg: color.Color, bg: color.Color, alpha: number): color.Color => [
  Math.round(fg[0] * alpha + bg[0] * (1 - alpha)),
  Math.round(fg[1] * alpha + bg[1] * (1 - alpha)),
  Math.round(fg[2] * alpha + bg[2] * (1 - alpha)),
  255,
];
