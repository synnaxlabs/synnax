// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { color } from "@/color/core";
import { specZ } from "@/text/types";

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

export const themeZ = z.object({
  name: z.string(),
  key: z.string(),
  colors: z.object({
    border: color.Color.z,
    primary: z.object({
      m2: color.Color.z,
      m1: color.Color.z,
      z: color.Color.z,
      p1: color.Color.z,
      p2: color.Color.z,
    }),
    gray: grayScaleZ,
    error: z.object({
      m2: color.Color.z,
      m1: color.Color.z,
      z: color.Color.z,
      p1: color.Color.z,
      p2: color.Color.z,
    }),
    visualization: z.object({
      palettes: z.record(z.array(color.Color.z)),
    }),
    white: color.Color.z,
    black: color.Color.z,
    text: color.Color.z,
    textContrast: color.Color.z,
    logo: z.string(),
  }),
  sizes: z.object({
    base: z.number(),
    border: z.object({
      radius: z.number(),
      width: z.number(),
    }),
    pid: z.object({
      elementStrokeWidth: z.number(),
    }),
  }),
  typography: z.object({
    family: z.string(),
    h1: specZ,
    h2: specZ,
    h3: specZ,
    h4: specZ,
    h5: specZ,
    p: specZ,
    small: specZ,
  }),
});

export type ThemeSpec = z.input<typeof themeZ>;
export type Theme = z.output<typeof themeZ>;

const fontFamily = "'Inter Variable', sans-serif";
const baseSize: number = 6;

const setLightness = (color: color.HSLA, lightness: number): color.HSLA => [
  color[0],
  color[1],
  lightness,
  color[3],
];

// Error

const ERROR_HSLA: color.HSLA = [357, 91, 55, 1];

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

const synnaxBase: ThemeSpec = {
  key: "synnax-base",
  name: "Synnax Base",
  colors: {
    primary: {
      m2: "#041B3D",
      m1: "#164FA0",
      z: "#3774D0",
      p1: "#5E94EE",
      p2: "#8AB8FF",
    },
    gray: lightGrayScale,
    border: lightGrayScale.l3,
    error: {
      m2: color.fromHSLA(setLightness(ERROR_HSLA, 30)),
      m1: color.fromHSLA(setLightness(ERROR_HSLA, 40)),
      z: color.fromHSLA(ERROR_HSLA),
      p1: color.fromHSLA(setLightness(ERROR_HSLA, 60)),
      p2: color.fromHSLA(setLightness(ERROR_HSLA, 70)),
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
    textContrast: lightGrayScale.l0,
  },
  sizes: {
    base: baseSize,
    border: {
      radius: 2,
      width: 1,
    },
    pid: {
      elementStrokeWidth: 2,
    },
  },
  typography: {
    family: fontFamily,
    h1: {
      size: 5,
      weight: "500",
      lineHeight: 6,
    },
    h2: {
      size: 4,
      weight: "medium",
      lineHeight: 5,
    },
    h3: {
      size: 3,
      weight: "medium",
      lineHeight: 4,
    },
    h4: {
      size: 2.5,
      weight: "medium",
      lineHeight: 3,
    },
    h5: {
      size: 2.25,
      weight: "medium",
      lineHeight: 2.5,
      textTransform: "uppercase",
    },
    p: {
      size: 2.25,
      weight: "regular",
      lineHeight: 2.5,
    },
    small: {
      size: 2,
      weight: "regular",
      lineHeight: 2.3,
    },
  },
};

export const synnaxLight: ThemeSpec = {
  ...synnaxBase,
  key: "synnax-light",
  name: "Synnax Light",
};

const DARK_SCALE = [
  "#020202",
  "#080808",
  "#141414",
  "#1a1a1a",
  "#242424",
  "#515151",
  "#7f7f7f",
  "#9D9D9D",
  "#BFBFBF",
  "#E2E2E2",
  "#FDFDFD",
];

const darkGrayScale: GrayScale = Object.fromEntries(
  DARK_SCALE.map((color, index) => [`l${index}`, color]),
) as GrayScale;

export const synnaxDark: ThemeSpec = {
  ...synnaxBase,
  key: "synnax-dark",
  name: "Synnax Dark",
  colors: {
    ...synnaxBase.colors,
    gray: darkGrayScale,
    logo: "var(--pluto-text-color)",
    border: darkGrayScale.l3,
    text: darkGrayScale.l9,
    textContrast: darkGrayScale.l0,
  },
};

export const themes = { synnaxDark, synnaxLight };
