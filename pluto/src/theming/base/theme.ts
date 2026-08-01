// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, text } from "@synnaxlabs/x";
import { z } from "zod";

const grayScaleZ = z.object({
  // App canvas
  l0: color.colorZ,
  // Raised surface
  l1: color.colorZ,
  // Elevated chrome
  l2: color.colorZ,
  // Component rest fill
  l3: color.colorZ,
  // Hover fill
  l4: color.colorZ,
  // Pressed fill
  l5: color.colorZ,
  // Subtle separator
  l6: color.colorZ,
  // Default control border
  l7: color.colorZ,
  // Strong border
  l8: color.colorZ,
  // Secondary text
  l9: color.colorZ,
  // Primary text
  l10: color.colorZ,
  // Emphatic text
  l11: color.colorZ,
});

type GrayScale = z.input<typeof grayScaleZ>;

export type Shade = 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11;

const setLightness = (color: color.HSLA, lightness: number): color.HSLA => [
  color[0],
  color[1],
  lightness,
  color[3],
];

const strictScaleZ = z.object({
  m2: color.colorZ,
  m1: color.colorZ,
  z: color.colorZ,
  p1: color.colorZ,
  p2: color.colorZ,
});

const scaleZ = strictScaleZ.or(
  color.colorZ.transform((c) => {
    const hsla = color.hsla(c);
    return {
      m2: color.fromHSLA(setLightness(hsla, 40)),
      m1: color.fromHSLA(setLightness(hsla, 45)),
      z: c,
      p1: color.fromHSLA(setLightness(hsla, 55)),
      p2: color.fromHSLA(setLightness(hsla, 65)),
    } as const;
  }),
);

export const themeZ = z
  .object({
    name: z.string(),
    key: z.string(),
    colors: z.object({
      border: color.colorZ,
      primary: scaleZ,
      gray: grayScaleZ,
      error: scaleZ,
      secondary: scaleZ,
      warning: scaleZ,
      palettes: z.record(z.string(), color.paletteZ),
      visualization: z
        .object({ palettes: z.record(z.string(), z.array(color.colorZ)) })
        .default({ palettes: {} }),
      white: color.colorZ,
      black: color.colorZ,
      text: color.colorZ,
      textInverted: color.colorZ,
      textOnPrimary: color.colorZ.default(color.ZERO),
      primaryText: color.colorZ,
      errorText: color.colorZ,
      warningText: color.colorZ,
      logo: z.string(),
    }),
    sizes: z.object({
      base: z.number(),
      border: z.object({
        /* Radius steps are rem multiples of the base size, rounded to whole
           pixels on emission. Tiny is the theme's default radius. */
        radius: z.object({
          tiny: z.number(),
          small: z.number(),
          medium: z.number(),
          large: z.number(),
          huge: z.number(),
        }),
        width: z.number(),
        thickWidth: z.number(),
      }),
      schematic: z.object({ elementStrokeWidth: z.number() }),
    }),
    typography: z.object({
      family: z.string(),
      codeFamily: z.string(),
      h1: text.specZ,
      h2: text.specZ,
      h3: text.specZ,
      h4: text.specZ,
      h5: text.specZ,
      p: text.specZ,
      small: text.specZ,
    }),
  })
  .transform((theme) => {
    if (theme.colors.textOnPrimary == null || color.isZero(theme.colors.textOnPrimary))
      theme.colors.textOnPrimary = color.pickByContrast(
        theme.colors.primary.z,
        theme.colors.text,
        theme.colors.textInverted,
      );
    return theme;
  });

export type ThemeSpec = z.input<typeof themeZ>;
export type Theme = z.infer<typeof themeZ>;

const fontFamily = '"Inter Variable", sans-serif';
const codeFontFamily = '"Geist Mono", monospace';
const baseSize: number = 6;

// Error

const ERROR_HSLA: color.HSLA = [357, 91, 55, 1];

// Warning

// Pure yellow only reads on dark surfaces; on light ones it washes out, and
// darkening it alone goes olive. The light ramp rotates toward amber instead.
const DARK_WARNING_HSLA: color.HSLA = [48, 90, 55, 1];
const LIGHT_WARNING_HSLA: color.HSLA = [38, 92, 46, 1];

// Grayscale

// Both ramps are generated in OKLCH (hue 258, whisper chroma) with even
// perceptual steps inside each band. Bands never overlap: a component fill is
// always at least one step from any surface.
const LIGHT_SCALE = [
  "#FDFDFF", // l0 - app canvas
  "#F6F7F9", // l1 - raised surface
  "#EFF1F4", // l2 - elevated chrome
  "#E8EAED", // l3 - component rest fill
  "#DFE2E6", // l4 - hover fill
  "#D6D9DE", // l5 - pressed fill
  "#CCD0D5", // l6 - subtle separator
  "#BBBEC3", // l7 - default control border
  "#9EA2A7", // l8 - strong border
  "#63666C", // l9 - secondary text (AA on all surfaces)
  "#2E3034", // l10 - primary text
  "#07080A", // l11 - emphatic text
];

const lightGrayScale: GrayScale = Object.fromEntries(
  LIGHT_SCALE.map((color, index) => [`l${index}`, color]),
) as GrayScale;

const supportsThinBorder = () => {
  if (typeof window === "undefined") return false;
  return window.devicePixelRatio > 1;
};

const SUPPORTS_THIN_BORDER = supportsThinBorder();

const SYNNAX_BASE: ThemeSpec = {
  key: "synnaxBase",
  name: "Synnax Base",
  colors: {
    primary: {
      m2: "#041B3D",
      m1: "#356fc5",
      z: "#3774D0",
      p1: "#5E94EE",
      p2: "#8AB8FF",
    },
    secondary: {
      m2: "#2D8F4E",
      m1: "#38B261",
      z: "#50C878",
      p1: "#73D393",
      p2: "#96DEAE",
    },
    gray: lightGrayScale,
    border: lightGrayScale.l6,
    error: {
      m2: color.fromHSLA(setLightness(ERROR_HSLA, 30)),
      m1: color.fromHSLA(setLightness(ERROR_HSLA, 40)),
      z: color.fromHSLA(ERROR_HSLA),
      p1: color.fromHSLA(setLightness(ERROR_HSLA, 65)),
      p2: color.fromHSLA(setLightness(ERROR_HSLA, 77)),
    },
    warning: {
      m2: color.fromHSLA(setLightness(LIGHT_WARNING_HSLA, 28)),
      m1: color.fromHSLA(setLightness(LIGHT_WARNING_HSLA, 36)),
      z: color.fromHSLA(LIGHT_WARNING_HSLA),
      p1: color.fromHSLA(setLightness(LIGHT_WARNING_HSLA, 54)),
      p2: color.fromHSLA(setLightness(LIGHT_WARNING_HSLA, 64)),
    },
    palettes: { recent: { key: "recent", name: "Recent", swatches: [] } },
    visualization: {
      palettes: {
        default: [
          "#DC136C",
          "#20A4F3",
          "#7AC74F",
          "#FFC43D",
          "#FE5F55",
          "#8075FF",
          "#D90429",
          "#f7aef8",
          "#f18f01",
          "#791e94",
          "#279af1",
          "#d9fff5",
          "#ff6b6b",
          "#52ffb8",
        ],
      },
    },
    logo: "url(#synnax-linear-gradient)",
    white: "#FFFFFF",
    black: "#000000",
    text: lightGrayScale.l10,
    textInverted: lightGrayScale.l0,
    textOnPrimary: lightGrayScale.l0,
    primaryText: "#3b82f6",
    errorText: "#ef4444",
    warningText: "#f59e0b",
  },
  sizes: {
    base: baseSize,
    border: {
      radius: { tiny: 2 / 3, small: 1, medium: 1.5, large: 2, huge: 3 },
      width: SUPPORTS_THIN_BORDER ? 0.5 : 1,
      thickWidth: 1,
    },
    schematic: { elementStrokeWidth: 2 },
  },
  typography: {
    family: fontFamily,
    codeFamily: codeFontFamily,
    h1: { size: 5.5, weight: "500", lineHeight: 5.5 * 1.5 },
    h2: { size: 4.5, weight: "medium", lineHeight: 4.5 * 1.5 },
    h3: { size: 3.5, weight: "medium", lineHeight: 3.5 * 1.5 },
    h4: { size: 2.6666, weight: "medium", lineHeight: 2.6666 * 1.5 },
    h5: { size: 2.333333, weight: 450, lineHeight: 2.333333 * 1.5 },
    p: { size: 2.1666666666, weight: "regular", lineHeight: 2 * 1.5 },
    small: { size: 1.916666, weight: "regular", lineHeight: 2.3333333 },
  },
};

export const SYNNAX_LIGHT: ThemeSpec = Object.freeze({
  ...SYNNAX_BASE,
  key: "synnaxLight",
  name: "Synnax Light",
});

const DARK_SCALE = [
  "#040506", // l0 - app canvas
  "#0A0B0D", // l1 - raised surface
  "#111315", // l2 - elevated chrome
  "#191C20", // l3 - component rest fill
  "#23252A", // l4 - hover fill
  "#2C2F34", // l5 - pressed fill
  "#36393F", // l6 - subtle separator
  "#44484D", // l7 - default control border
  "#5D6166", // l8 - strong border
  "#A2A5A8", // l9 - secondary text (AA on all surfaces)
  "#CFD1D4", // l10 - primary text
  "#F1F2F3", // l11 - emphatic text
];

const DARK_GRAY_SCALE: GrayScale = Object.fromEntries(
  DARK_SCALE.map((color, index) => [`l${index}`, color]),
) as GrayScale;

export const SYNNAX_DARK: ThemeSpec = Object.freeze({
  ...SYNNAX_BASE,
  key: "synnaxDark",
  name: "Synnax Dark",
  colors: {
    ...SYNNAX_BASE.colors,
    gray: DARK_GRAY_SCALE,
    warning: {
      m2: color.fromHSLA(setLightness(DARK_WARNING_HSLA, 30)),
      m1: color.fromHSLA(setLightness(DARK_WARNING_HSLA, 40)),
      z: color.fromHSLA(DARK_WARNING_HSLA),
      p1: color.fromHSLA(setLightness(DARK_WARNING_HSLA, 65)),
      p2: color.fromHSLA(setLightness(DARK_WARNING_HSLA, 75)),
    },
    logo: "var(--pluto-text-color)",
    border: DARK_GRAY_SCALE.l6,
    text: DARK_GRAY_SCALE.l10,
    textInverted: DARK_GRAY_SCALE.l0,
    textOnPrimary: DARK_GRAY_SCALE.l11,
    primaryText: "#93c5fd",
    errorText: "#fca5a5",
    warningText: "#fcd34d",
  },
});

export const SYNNAX_THEMES = { synnaxDark: SYNNAX_DARK, synnaxLight: SYNNAX_LIGHT };
