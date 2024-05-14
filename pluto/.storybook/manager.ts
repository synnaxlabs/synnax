// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { addons } from "@storybook/addons";
import { themes, create } from "@storybook/theming";
import "./index.css";

const baseSize = 6;

const setLightness = (color, lightness: number) => [
  color[0],
  color[1],
  lightness,
  color[3],
];

// Error

const ERROR_HSLA = [357, 91, 55, 1];

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

const lightGrayScale = Object.fromEntries(
  LIGHT_SCALE.map((color, index) => [`l${index}`, color]),
);

const fontFamily = "'Inter Variable', sans-serif";

const synnaxBase = {
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
    secondary: {
      m2: "#2D8F4E",
      m1: "#38B261",
      z: "#50C878",
      p1: "#73D393",
      p2: "#96DEAE",
    },
    gray: lightGrayScale,
    border: lightGrayScale.l3,
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
      lineHeight: 3.25,
    },
    small: {
      size: 2,
      weight: "regular",
      lineHeight: 2.3,
    },
  },
};

export const synnaxLight = {
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
  "#EDEDED",
  "#FDFDFD",
];

const darkGrayScale = Object.fromEntries(
  DARK_SCALE.map((color, index) => [`l${index}`, color]),
);

export const synnaxDark = {
  ...synnaxBase,
  key: "synnax-dark",
  name: "Synnax Dark",
  colors: {
    ...synnaxBase.colors,
    gray: darkGrayScale,
    logo: "var(--pluto-text-color)",
    border: darkGrayScale.l3,
    text: darkGrayScale.l9,
    textInverted: darkGrayScale.l0,
  },
};

const t = synnaxDark;

const theme = create({
  ...t,
  colorSecondary: t.colors.primary.z,
  appBg: t.colors.gray.l0,
  appContentBg: t.colors.gray.l0,
  appBorderColor: t.colors.border,
  appBorderRadius: t.sizes.border.radius as number,
  fontBase: t.typography.family,
  brandImage:
    "https://raw.githubusercontent.com/synnaxlabs/synnax/main/x/media/static/logo/title-white.png",
  brandUrl: "https://docs.synnaxlabs.com",
});

addons.setConfig({
  theme: theme,
});
