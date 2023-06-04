// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@fontsource/inter";

import { Color } from "../color";

import { TypographySpec } from "@/core/std/Typography";

export type Size = number | string;

export interface Theme {
  name: string;
  key: string;
  colors: {
    border: string;
    primary: {
      m1: string;
      z: string;
      p1: string;
    };
    gray: {
      m3: string;
      m2: string;
      m1: string;
      m0: string;
      p0: string;
      p1: string;
      p2: string;
      p3: string;
    };
    error: {
      m1: string;
      z: string;
      p1: string;
    };
    visualization: {
      palettes: {
        default: string[];
        [key: string]: string[];
      };
    };
    white: string;
    black: string;
    background: string;
    text: string;
    logo: string;
  };
  sizes: {
    base: number;
    border: {
      radius: Size;
      width: Size;
    };
  };
  typography: {
    family: string;
    h1: TypographySpec;
    h2: TypographySpec;
    h3: TypographySpec;
    h4: TypographySpec;
    h5: TypographySpec;
    p: TypographySpec;
    small: TypographySpec;
    tiny: TypographySpec;
  };
}

const white: string = "#FFFFFF";
const black: string = "#171716";
const fontFamily = "Inter, sans-serif";
const baseSize: Size = 6;

const synnaxBase: Theme = {
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
    text: new Color(black).setOpacity(0.85).hex,
  },
  sizes: {
    base: baseSize,
    border: {
      radius: 2,
      width: 1,
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
      weight: 900,
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
      lineHeight: 2 + 1 / 3,
    },
    tiny: {
      size: 1.75,
      weight: 300,
      lineHeight: 2,
    },
  },
};

export const synnaxLight: Theme = {
  ...synnaxBase,
  key: "synnax-light",
  name: "Synnax Light",
};

export const synnaxDark: Theme = {
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
    text: new Color(synnaxBase.colors.white).setOpacity(0.9).hex,
  },
};
