// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { color } from "@/color/core";
import { text } from "@/text/core";

const grayScaleZ = z.object({
  l0: color.Color.z,
  l1: color.Color.z,
  l2: color.Color.z,
  l3: color.Color.z,
  l4: color.Color.z,
  l5: color.Color.z,
  l6: color.Color.z,
  l7: color.Color.z,
  l8: color.Color.z,
  l9: color.Color.z,
  l10: color.Color.z,
});

type GrayScale = z.input<typeof grayScaleZ>;

const setLightness = (color: color.HSLA, lightness: number): color.HSLA => [
  color[0],
  color[1],
  lightness,
  color[3],
];

const strictScaleZ = z.object({
  m2: color.Color.z,
  m1: color.Color.z,
  z: color.Color.z,
  p1: color.Color.z,
  p2: color.Color.z,
});

const scaleZ = strictScaleZ.or(
  color.Color.z.transform((c) => {
    const hsla = c.hsla;
    return {
      m2: new color.Color(color.fromHSLA(setLightness(hsla, 40))),
      m1: new color.Color(color.fromHSLA(setLightness(hsla, 45))),
      z: c,
      p1: new color.Color(color.fromHSLA(setLightness(hsla, 55))),
      p2: new color.Color(color.fromHSLA(setLightness(hsla, 65))),
    } as const as z.output<typeof strictScaleZ>;
  }),
);

export const themeZ = z
  .object({
    name: z.string(),
    key: z.string(),
    colors: z.object({
      border: color.Color.z,
      primary: scaleZ,
      gray: grayScaleZ,
      error: scaleZ,
      secondary: scaleZ,
      warning: scaleZ,
      visualization: z
        .object({
          palettes: z.record(z.array(color.Color.z)),
        })
        .optional()
        .default({ palettes: {} }),
      white: color.Color.z,
      black: color.Color.z,
      text: color.Color.z,
      textInverted: color.Color.z,
      textOnPrimary: color.Color.z.optional().default(color.ZERO),
      logo: z.string(),
    }),
    sizes: z.object({
      base: z.number(),
      border: z.object({
        radius: z.number(),
        width: z.number(),
      }),
      schematic: z.object({
        elementStrokeWidth: z.number(),
      }),
    }),
    typography: z.object({
      family: z.string(),
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
    if (theme.colors.textOnPrimary == null || theme.colors.textOnPrimary.isZero)
      theme.colors.textOnPrimary = theme.colors.primary.z.pickByContrast(
        theme.colors.text,
        theme.colors.textInverted,
      );
    return theme;
  });

export type ThemeSpec = z.input<typeof themeZ>;
export type Theme = z.output<typeof themeZ>;

const fontFamily = "'Inter Variable', sans-serif";
const baseSize: number = 6;

// Error

const ERROR_HSLA: color.HSLA = [357, 91, 55, 1];

// Warning

const WARNING_HSLA: color.HSLA = [42, 100, 50, 1];

// Grayscale

const LIGHT_SCALE = [
  "#FCFCFC",
  "#F9F9F9",
  "#F4F4F4",
  "#ededed",
  "#d9d9d9",
  "#ADADAD",
  "#878787",
  "#616161",
  "#404040",
  "#1C1C1C",
  "#050505",
];

const lightGrayScale: GrayScale = Object.fromEntries(
  LIGHT_SCALE.map((color, index) => [`l${index}`, color]),
) as GrayScale;

const SYNNAX_BASE: ThemeSpec = {
  key: "synnaxBase",
  name: "Synnax Base",
  colors: {
    primary: {
      m2: "#041B3D",
      m1: "#164FA0",
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
    border: lightGrayScale.l3,
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
    text: lightGrayScale.l9,
    textInverted: lightGrayScale.l0,
  },
  sizes: {
    base: baseSize,
    border: {
      radius: 3,
      width: 1,
    },
    schematic: {
      elementStrokeWidth: 2,
    },
  },
  typography: {
    family: fontFamily,
    h1: {
      size: 7,
      weight: "500",
      lineHeight: 7 * 1.5,
    },
    h2: {
      size: 4.5,
      weight: "medium",
      lineHeight: 4.5 * 1.5,
    },
    h3: {
      size: 3.5,
      weight: "medium",
      lineHeight: 3.5 * 1.5,
    },
    h4: {
      size: 2.75,
      weight: "medium",
      lineHeight: 2.75 * 1.5,
    },
    h5: {
      size: 2.333333,
      weight: 450,
      lineHeight: 2.333333 * 1.5,
    },
    p: {
      size: 2.1666666666,
      weight: "regular",
      lineHeight: 2 * 1.5,
    },
    small: {
      size: 1.916666,
      weight: "regular",
      lineHeight: 1.9166666 * 1.5,
    },
  },
};

export const SYNNAX_LIGHT: ThemeSpec = Object.freeze({
  ...SYNNAX_BASE,
  key: "synnaxLight",
  name: "Synnax Light",
});

const DARK_SCALE = [
  "#020202",
  "#0D0D0D",
  "#191919",
  "#252525",
  "#313131",
  "#515151",
  "#7f7f7f",
  "#9D9D9D",
  "#BFBFBF",
  "#EDEDED",
  "#FDFDFD",
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
    border: DARK_GRAY_SCALE.l3,
    text: DARK_GRAY_SCALE.l9,
    textInverted: DARK_GRAY_SCALE.l0,
  },
});

export const SYNNAX_THEMES = { synnaxDark: SYNNAX_DARK, synnaxLight: SYNNAX_LIGHT };
