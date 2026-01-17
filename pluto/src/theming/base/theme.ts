// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { z } from "zod";

import { text } from "@/text/base";

const grayScaleZ = z.object({
  // Main background surface
  l0: color.colorZ,
  // Large surfaces to contrast against background
  l1: color.colorZ,
  // Small surfaces to contrast against background
  l2: color.colorZ,
  // Hover on small surfaces
  l3: color.colorZ,
  // Border strength 1
  l4: color.colorZ,
  // Border strength 2
  // Border strength 1 hover
  l5: color.colorZ,
  // Border strength 2 hover
  // Border strength 1 active
  l6: color.colorZ,
  // Border strength 2 active
  l7: color.colorZ,
  // Text strength 1
  l8: color.colorZ,
  // Text strength 2
  l9: color.colorZ,
  // Text strength 3
  l10: color.colorZ,
  // Text strength 4
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
    } as const as z.infer<typeof strictScaleZ>;
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
      logo: z.string(),
    }),
    sizes: z.object({
      base: z.number(),
      border: z.object({
        radius: z.number(),
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

const WARNING_HSLA: color.HSLA = [48, 90, 55, 1];

// Grayscale

const LIGHT_SCALE = [
  "#FEFEFE", // l0 - background
  "#FAFAFA", // l1 - large surfaces
  "#F2F2F2", // l2 - small surfaces
  "#ECECEC", // l3 - small surfaces hover
  "#E4E4E4", // l4 - border 1
  "#D1D1D1", // l5 - border 2
  "#BCBCBC", // l6 - border 2 hover
  "#ACACAC", // l7 - border 2 active
  "#8F8F8F", // l8 - text 1
  "#4F4F4F", // l10 - text 2
  "#292929", // l10 - text 3
  "#050505", // l11 - text 4
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
    border: lightGrayScale.l4,
    error: {
      m2: color.fromHSLA(setLightness(ERROR_HSLA, 30)),
      m1: color.fromHSLA(setLightness(ERROR_HSLA, 40)),
      z: color.fromHSLA(ERROR_HSLA),
      p1: color.fromHSLA(setLightness(ERROR_HSLA, 65)),
      p2: color.fromHSLA(setLightness(ERROR_HSLA, 77)),
    },
    warning: {
      m2: color.fromHSLA(setLightness(WARNING_HSLA, 30)),
      m1: color.fromHSLA(setLightness(WARNING_HSLA, 40)),
      z: color.fromHSLA(WARNING_HSLA),
      p1: color.fromHSLA(setLightness(WARNING_HSLA, 65)),
      p2: color.fromHSLA(setLightness(WARNING_HSLA, 75)),
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
    text: lightGrayScale.l11,
    textInverted: lightGrayScale.l0,
    textOnPrimary: lightGrayScale.l0,
  },
  sizes: {
    base: baseSize,
    border: { radius: 3, width: SUPPORTS_THIN_BORDER ? 0.5 : 1, thickWidth: 1 },
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
  "#020202", // l0
  "#080808", // l1
  "#151515", // l2
  "#202020", // l3
  "#272727", // l4
  "#3B3B3B", // l5
  "#4A4A4A", // l6
  "#5C5C5C", // l7
  "#767676", // l8
  "#959595", // l9
  "#dadada", // l10
  "#FAFAFA", // l11
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
    logo: "var(--pluto-text-color)",
    border: DARK_GRAY_SCALE.l4,
    text: DARK_GRAY_SCALE.l11,
    textInverted: DARK_GRAY_SCALE.l0,
    textOnPrimary: DARK_GRAY_SCALE.l11,
  },
});

export const SYNNAX_THEMES = { synnaxDark: SYNNAX_DARK, synnaxLight: SYNNAX_LIGHT };
