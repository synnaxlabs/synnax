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
      m1: color.Color.z,
      z: color.Color.z,
      p1: color.Color.z,
    }),
    gray: z.object({
      m3: color.Color.z,
      m2: color.Color.z,
      m1: color.Color.z,
      m0: color.Color.z,
      p0: color.Color.z,
      p1: color.Color.z,
      p2: color.Color.z,
      p3: color.Color.z,
    }),
    error: z.object({
      m1: color.Color.z,
      z: color.Color.z,
      p1: color.Color.z,
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

const white: string = "#FFFFFF";
const black: string = "#121212";
const fontFamily = "'Inter Variable', sans-serif";
const baseSize: number = 6;

const synnaxBase: ThemeSpec = {
  key: "synnax-base",
  name: "Synnax Base",
  colors: {
    primary: {
      m1: "#3363BE",
      z: "#3774D0",
      p1: "#3b84e5",
    },
    gray: {
      p3: "#1D1D1C",
      p2: "#2c2c2c",
      p1: "#474744",
      p0: "#676762",
      m0: "#94938D",
      m1: "#C8C7BF",
      m2: "#eceaea",
      m3: "#FEFEFD",
    },
    border: "#C8C7BF",
    error: {
      m1: "#CF1322",
      z: "#F5222D",
      p1: "#FF4547",
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
    white,
    black,
    background: white,
    text: new color.Color(black).setAlpha(0.85).hex,
    textContrast: new color.Color(white).setAlpha(0.85).hex,
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
      lineHeight: 3.25,
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
      lineHeight: 2.5,
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
      m3: synnaxBase.colors.gray.p3,
      m2: synnaxBase.colors.gray.p2,
      m1: synnaxBase.colors.gray.p1,
      m0: synnaxBase.colors.gray.p0,
      p0: synnaxBase.colors.gray.m0,
      p1: synnaxBase.colors.gray.m1,
      p2: synnaxBase.colors.gray.m2,
      p3: synnaxBase.colors.gray.m3,
    },
    logo: "var(--pluto-text-color)",
    border: synnaxBase.colors.gray.p1,
    background: synnaxBase.colors.black,
    text: new color.Color(synnaxBase.colors.white).setAlpha(0.9).hex,
    textContrast: new color.Color(synnaxBase.colors.black).setAlpha(0.9).hex,
  },
};

export const themes = { synnaxDark, synnaxLight };
