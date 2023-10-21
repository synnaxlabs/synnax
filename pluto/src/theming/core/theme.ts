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
    gray: z.object({
      m4: color.Color.z,
      m3: color.Color.z,
      m2: color.Color.z,
      m1: color.Color.z,
      m0: color.Color.z,
      p0: color.Color.z,
      p1: color.Color.z,
      p2: color.Color.z,
      p3: color.Color.z,
      p4: color.Color.z,
    }),
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
    background: color.Color.z,
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

const WHITE = "#FFFFFF";
const BLACK = "#121212";
const GRAY_P4 = "#1C1C1C";
const GRAY_P3 = "#2B2B2B";
const GRAY_P2 = "#424242";
const GRAY_P1 = "#5F5F5F";
const GRAY_P0 = "#7D7D7D";
const GRAY_M0 = "#9B9B9B";
const GRAY_M1 = "#B5B5B5";
const GRAY_M2 = "#D1D1D1";
const GRAY_M3 = "#EDEDED";
const GRAY_M4 = "#F7F7F7";

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
    gray: {
      p4: GRAY_P4,
      p3: GRAY_P3,
      p2: GRAY_P2,
      p1: GRAY_P1,
      p0: GRAY_P0,
      m0: GRAY_M0,
      m1: GRAY_M1,
      m2: GRAY_M2,
      m3: GRAY_M3,
      m4: GRAY_M4,
    },
    border: GRAY_M3,
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
    white: WHITE,
    black: BLACK,
    background: WHITE,
    text: GRAY_P4,
    textContrast: GRAY_M4,
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

export const synnaxDark: ThemeSpec = {
  ...synnaxBase,
  key: "synnax-dark",
  name: "Synnax Dark",
  colors: {
    ...synnaxBase.colors,
    gray: {
      m4: synnaxBase.colors.gray.p4,
      m3: synnaxBase.colors.gray.p3,
      m2: synnaxBase.colors.gray.p2,
      m1: synnaxBase.colors.gray.p1,
      m0: synnaxBase.colors.gray.p0,
      p0: synnaxBase.colors.gray.m0,
      p1: synnaxBase.colors.gray.m1,
      p2: synnaxBase.colors.gray.m2,
      p3: synnaxBase.colors.gray.m3,
      p4: synnaxBase.colors.gray.m4,
    },
    logo: "var(--pluto-text-color)",
    border: synnaxBase.colors.gray.p3,
    background: synnaxBase.colors.black,
    text: synnaxBase.colors.gray.m4,
    textContrast: synnaxBase.colors.black,
  },
};

export const themes = { synnaxDark, synnaxLight };
