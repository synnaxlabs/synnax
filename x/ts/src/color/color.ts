// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { caseconv } from "@/caseconv";
import { NAMES } from "@/color/names";
import { zod } from "@/zod";

/**
 * A zod schema for a color string: a CSS color name or the transparent keyword, a hex
 * value of 3, 4, 6, or 8 digits with or without a leading '#', an rgb() or rgba()
 * function, or an hsl() or hsla() function.
 */
const stringZ = z.string().transform((s, ctx): RGBA => {
  const c = parseString(s);
  if (c != null) return c;
  ctx.addIssue({ code: "custom", message: `invalid color: ${s}` });
  return z.NEVER;
});
/** A zod schema for an RGB value. */
const rgbValueZ = zod.uint8;
/** A zod schema for an alpha value between 0 and 255. */
const alphaZ = zod.uint8;
/** A zod schema for an RGBA color. */
const rgbaZ = z.tuple([rgbValueZ, rgbValueZ, rgbValueZ, alphaZ]);
/** A zod schema for an RGB color. */
const rgbZ = z.tuple([rgbValueZ, rgbValueZ, rgbValueZ]);
/** A zod schema for an alpha value between 0 and 1. */
const opacityZ = z.number().min(0).max(1);
/** A zod schema for a hue value between 0 and 360. */
const hueZ = z.number().min(0).max(360);
/** A zod schema for a saturation value between 0 and 100. */
const saturationZ = z.number().min(0).max(100);
/** A zod schema for a lightness value between 0 and 100. */
const lightnessZ = z.number().min(0).max(100);
/** A zod schema for an HSLA color. */
const hslaZ = z.tuple([hueZ, saturationZ, lightnessZ, opacityZ]);
/** A zod schema for an HSV value (brightness) between 0 and 100. */
const valueZ = z.number().min(0).max(100);
/** A zod schema for an HSVA color. */
const hsvaZ = z.tuple([hueZ, saturationZ, valueZ, opacityZ]);

/**
 * A color in RGBA format, with every component an integer between 0 and 255. See
 * https://en.wikipedia.org/wiki/RGBA_color_model
 */
export type RGBA = z.infer<typeof rgbaZ>;
/**
 * A color in HSLA format: hue 0-360, saturation and lightness 0-100, alpha 0-1. See
 * https://en.wikipedia.org/wiki/HSL_and_HSV
 */
export type HSLA = z.infer<typeof hslaZ>;
/**
 * A color in HSVA format: hue 0-360, saturation and value 0-100, alpha 0-1. Values are
 * not rounded. See https://en.wikipedia.org/wiki/HSL_and_HSV
 */
export type HSVA = z.infer<typeof hsvaZ>;
/**
 * A color in OKLCH format: lightness 0-1, chroma, hue 0-360, alpha 0-1. See
 * https://www.w3.org/TR/css-color-4/#ok-lab
 */
export type OKLCH = [number, number, number, number];

/** A zod schema for a crude color representation. */
export const crudeZ = z.union([stringZ, rgbZ, rgbaZ]);
/**
 * An unparsed representation of a color i.e. a value that can be converted into
 * a Color object.
 */
export type Crude = z.input<typeof crudeZ>;

/**
 * A zod schema to parse color values from various crude representations. The JSON
 * codec writes a parsed color as its hex string.
 */
export const colorZ = caseconv.encodeWith(
  crudeZ.transform((v) => fromCrude(v)),
  (c: Crude) => hex(c),
);

/**
 * A color in RGBA format, with every component an integer between 0 and 255. Used as
 * the standard representation of a color in this package. See
 * https://en.wikipedia.org/wiki/RGBA_color_model
 */
export type Color = RGBA;

/** @returns true if the color is a true Color type. */
export const isColor = (color: unknown): color is Color => z.validate(rgbaZ, color);

/**
 * Converts a crude color to a CSS color value.
 * @returns undefined if the color is undefined.
 * @returns the hex string of the color if it parses as a color.
 * @returns any other string as it is, such as a CSS variable.
 * @throws {zod.ParseError} if the color does not match any of the preceding
 * conditions.
 */
export interface CSSString {
  (color: Crude): string;
  (color?: Crude): string | undefined;
}

export const cssString = ((color?: Crude): string | undefined => {
  if (color == null) return undefined;
  const res = colorZ.safeParse(color);
  if (res.success) return hex(res.data);
  if (typeof color === "string") return color;
  throw new zod.ParseError({
    issues: res.error.issues,
    input: color,
    label: "color",
    cause: res.error,
  });
}) as CSSString;

/**
 * @constructor Creates a new color from a color name, hex, rgb(), or hsl() string, an
 * RGB or RGBA tuple of bytes, or another color. A value without an alpha gets an alpha
 * of 255.
 * @throws {zod.ParseError} if the value is not a valid crude color.
 */
export const construct = (color: Crude): Color =>
  fromCrude(zod.parse(crudeZ, color, { label: "color" }));

const fromCrude = (parsed: z.output<typeof crudeZ>): Color =>
  parsed.length === 3 ? [...parsed, 255] : parsed;

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
 * @returns the hex representation of the color. If the color is opaque, the returned
 * hex will be 6 characters long. Otherwise, it will be 8 characters long.
 */
export const hex = ((color?: Crude) => {
  if (color == null) return undefined;
  const [r, g, b, a] = construct(color);
  return `#${byteToHex(r)}${byteToHex(g)}${byteToHex(b)}${a === 255 ? "" : byteToHex(a)}`;
}) as ToHex;

/**
 * @returns the comma-separated RGB channels of the color, for a CSS custom property
 * that CSS reads as rgba(var(...), alpha) with its own alpha.
 * @example "255, 255, 255"
 */
export const rgbString = (color: Crude): string => {
  const [r, g, b] = construct(color);
  return `${r}, ${g}, ${b}`;
};

/**
 * @returns the comma-separated RGB channels and 0-1 alpha of the color, for a CSS
 * custom property that CSS reads as rgb(var(...)). Returns undefined when no color is
 * given, so the property falls back to its default.
 * @example "255, 0, 0, 0.5"
 */
export const rgbaString = (color?: Crude): string | undefined =>
  color == null ? undefined : `${rgbString(color)}, ${opacity(color)}`;

/**
 * @returns the color as a tuple of four numbers between 0 and 1, as WebGL takes it.
 */
export const rgba1 = (color: Crude): [number, number, number, number] => [
  ...rgb1(color),
  opacity(color),
];

/** @returns the color normalized as an RGB tuple between 0 and 1. */
const rgb1 = (color: Crude): [number, number, number] => {
  const [r, g, b] = construct(color);
  return [r / 255, g / 255, b / 255];
};

/** @returns the opacity of the color, between 0 and 1. */
export const opacity = (color: Crude): number => construct(color)[3] / 255;

/** @returns the HSLA representation of the color. */
export const hsla = (color: Crude): HSLA => rgbaToHSLA(construct(color));

/**
 * @returns A new color with the given opacity.
 * @param color - The color to set the opacity on.
 * @param opacity - The opacity to set, between 0 and 1.
 * @throws {zod.ParseError} if the opacity is outside 0 to 1.
 */
export const setOpacity = (color: Crude, opacity: number): Color => {
  const [r, g, b] = construct(color);
  const o = zod.parse(opacityZ, opacity, { label: "opacity" });
  return [r, g, b, Math.round(o * 255)];
};
/**
 * @returns the luminance of the color, between 0 and 1.
 * @see https://en.wikipedia.org/wiki/Relative_luminance for more information.
 */
const luminance = (color: Crude): number => {
  const [r, g, b] = construct(color).map(toLinear);
  return Number((0.2126 * r + 0.7152 * g + 0.0722 * b).toFixed(3));
};

/** Converts an sRGB byte to a linear-light 0-1 value. */
const toLinear = (c: number): number => {
  const v = c / 255;
  return v <= 0.04045 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4;
};

/** Converts a linear-light value to an sRGB byte, clamping it into gamut. */
const fromLinear = (v: number): number => {
  const c = v <= 0.0031308 ? 12.92 * v : 1.055 * v ** (1 / 2.4) - 0.055;
  return Math.round(Math.min(1, Math.max(0, c)) * 255);
};

/**
 * @returns the OKLCH representation of the color: lightness 0-1, chroma, hue in
 * degrees, and alpha 0-1. Values are not rounded.
 */
export const oklch = (color: Crude): OKLCH => {
  const [r8, g8, b8, a] = construct(color);
  const [r, g, b] = [r8, g8, b8].map(toLinear);
  const l = Math.cbrt(0.4122214708 * r + 0.5363325363 * g + 0.0514459929 * b);
  const m = Math.cbrt(0.2119034982 * r + 0.6806995451 * g + 0.1073969566 * b);
  const s = Math.cbrt(0.0883024619 * r + 0.2817188376 * g + 0.6299787005 * b);
  const L = 0.2104542553 * l + 0.793617785 * m - 0.0040720468 * s;
  const A = 1.9779984951 * l - 2.428592205 * m + 0.4505937099 * s;
  const B = 0.0259040371 * l + 0.7827717662 * m - 0.808675766 * s;
  return [L, Math.hypot(A, B), (Math.atan2(B, A) * 180) / Math.PI, a / 255];
};

/** @returns the color for an OKLCH tuple, clamped into the sRGB gamut. */
export const fromOKLCH = ([L, c, hDeg, a]: OKLCH): Color => {
  const h = (hDeg * Math.PI) / 180;
  const A = c * Math.cos(h);
  const B = c * Math.sin(h);
  const l = (L + 0.3963377774 * A + 0.2158037573 * B) ** 3;
  const m = (L - 0.1055613458 * A - 0.0638541728 * B) ** 3;
  const s = (L - 0.0894841775 * A - 1.291485548 * B) ** 3;
  return [
    fromLinear(4.0767416621 * l - 3.3077115913 * m + 0.2309699292 * s),
    fromLinear(-1.2684380046 * l + 2.6097574011 * m - 0.3413193965 * s),
    fromLinear(-0.0041960863 * l - 0.7034186147 * m + 1.707614701 * s),
    Math.round(a * 255),
  ];
};

/**
 * @returns fg painted over bg at the given opacity, as an opaque color.
 * @param opacity - The opacity of fg, between 0 and 1.
 */
export const over = (fg: Crude, bg: Crude, opacity: number): Color => {
  const f = construct(fg);
  const b = construct(bg);
  const mix = (i: number): number => Math.round(f[i] * opacity + b[i] * (1 - opacity));
  return [mix(0), mix(1), mix(2), 255];
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

/** Transparent black, the color with every component 0. */
export const TRANSPARENT: Color = [0, 0, 0, 0];

const byteToHex = (n: number): string => Math.floor(n).toString(16).padStart(2, "0");

const FUNCTION_PATTERN = /^(rgb|hsl)a?\((.*)\)$/;
const NUMBER_PATTERN = /^[+-]?(\d+\.?\d*|\.\d+)$/;

/**
 * Parses a color string: a CSS color name or the transparent keyword, a hex value of 3,
 * 4, 6, or 8 digits with or without a leading '#', or an rgb(), rgba(), hsl(), or
 * hsla() function in comma or space syntax with an optional alpha. Out-of-range values
 * do not parse.
 */
const parseString = (input: string): RGBA | undefined => {
  const s = input.trim().toLowerCase();
  if (Object.hasOwn(NAMES, s)) return [...NAMES[s]];
  const fn = s.match(FUNCTION_PATTERN);
  if (fn != null) return parseFunction(fn[1], fn[2]);
  return parseHex(s.startsWith("#") ? s.slice(1) : s);
};

const parseHex = (digits: string): RGBA | undefined => {
  if (!/^[0-9a-f]+$/.test(digits)) return undefined;
  if (digits.length === 3 || digits.length === 4)
    digits = [...digits].map((d) => d + d).join("");
  if (digits.length !== 6 && digits.length !== 8) return undefined;
  const byte = (i: number): number => parseInt(digits.slice(i, i + 2), 16);
  return [byte(0), byte(2), byte(4), digits.length === 8 ? byte(6) : 255];
};

const parseFunction = (name: string, body: string): RGBA | undefined => {
  const args = body.split(/[\s,/]+/).filter((a) => a.length > 0);
  if (args.length !== 3 && args.length !== 4) return undefined;
  const alpha = args.length === 4 ? parseNumber(args[3], 1) : 1;
  if (alpha == null || alpha < 0 || alpha > 1) return undefined;
  const a = Math.round(alpha * 255);
  if (name === "rgb") {
    const channels = args.slice(0, 3).map((arg) => parseNumber(arg, 255));
    if (channels.some((c) => c == null || c < 0 || c > 255)) return undefined;
    const [r, g, b] = channels.map((c) => Math.round(c ?? 0));
    return [r, g, b, a];
  }
  const h = parseNumber(args[0].replace(/deg$/, ""), 1);
  const sat = parseNumber(args[1].replace(/%$/, ""), 100);
  const light = parseNumber(args[2].replace(/%$/, ""), 100);
  if (h == null || sat == null || light == null) return undefined;
  if (sat < 0 || sat > 100 || light < 0 || light > 100) return undefined;
  return hslToRGBA((((h % 360) + 360) % 360) / 360, sat / 100, light / 100, a);
};

/** Parses a number, reading a percentage as a fraction of scale. */
const parseNumber = (arg: string, scale: number): number | undefined => {
  const percent = arg.endsWith("%");
  const n = percent ? arg.slice(0, -1) : arg;
  if (!NUMBER_PATTERN.test(n)) return undefined;
  return percent ? (parseFloat(n) * scale) / 100 : parseFloat(n);
};

/** @returns the color for an HSLA tuple. */
export const fromHSLA = (hsla: HSLA): RGBA => {
  const [h, s, l, a] = zod.parse(hslaZ, hsla, { label: "hsla" });
  return hslToRGBA(h / 360, s / 100, l / 100, Math.round(a * 255));
};

/** Converts 0-1 hue, saturation, and lightness and a byte alpha to an RGBA color. */
const hslToRGBA = (h: number, s: number, l: number, a: number): RGBA => {
  if (s === 0) {
    const v = Math.round(l * 255);
    return [v, v, v, a];
  }
  const q = l < 0.5 ? l * (1 + s) : l + s - l * s;
  const p = 2 * l - q;
  return [
    Math.round(hueToRGB(p, q, h + 1 / 3) * 255),
    Math.round(hueToRGB(p, q, h) * 255),
    Math.round(hueToRGB(p, q, h - 1 / 3) * 255),
    a,
  ];
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

  h *= 360;
  s *= 100;
  l *= 100;

  return [Math.round(h), Math.round(s), Math.round(l), a / 255];
};

/**
 * @returns the HSVA representation of the color. Values are not rounded, so a round
 * trip through `fromHSVA` returns the same color.
 */
export const hsva = (color: Crude): HSVA => {
  const [r, g, b, a] = rgba1(color);
  const max = Math.max(r, g, b);
  const d = max - Math.min(r, g, b);
  let h = 0;
  if (d !== 0)
    if (max === r) h = ((g - b) / d + 6) % 6;
    else if (max === g) h = (b - r) / d + 2;
    else h = (r - g) / d + 4;
  return [h * 60, max === 0 ? 0 : (d / max) * 100, max * 100, a];
};

/** @returns the color for an HSVA tuple. */
export const fromHSVA = (hsva: HSVA): Color => {
  const [h, s, v, a] = zod.parse(hsvaZ, hsva, { label: "hsva" });
  const s1 = s / 100;
  const v1 = v / 100;
  const channel = (n: number): number => {
    const k = (n + h / 60) % 6;
    return Math.round((v1 - v1 * s1 * Math.max(0, Math.min(k, 4 - k, 1))) * 255);
  };
  return [channel(5), channel(3), channel(1), Math.round(a * 255)];
};
