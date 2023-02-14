// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@fontsource/inter";

import { addOpacityToHex } from "@synnaxlabs/x";

import { TypographySpec } from "@/core/Typography";

export type Size = number | string;
export type Color = string;

export interface Theme {
  name: string;
  key: string;
  colors: {
    border: Color;
    primary: {
      m1: Color;
      z: Color;
      p1: Color;
    };
    gray: {
      m3: Color;
      m2: Color;
      m1: Color;
      m0: Color;
      p0: Color;
      p1: Color;
      p2: Color;
      p3: Color;
    };
    error: {
      m1: Color;
      z: Color;
      p1: Color;
    };
    visualization: {
      palettes: {
        default: Color[];
        [key: string]: Color[];
      };
    };
    white: Color;
    black: Color;
    background: Color;
    text: Color;
    logo: Color;
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

const white: Color = "#FFFFFF";
const black: Color = "#171716";
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
      p2: "#30302E",
      p1: "#474744",
      p0: "#676762",
      m0: "#94938D",
      m1: "#C8C7BF",
      m2: "#e5e5e5",
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
          "#470063",
          "#020877",
          "#D90429",
        ],
      },
    },
    logo: "url(#linear-gradient)",
    white,
    black,
    background: white,
    text: black,
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
      size: 2.5,
      weight: "medium",
      lineHeight: 2.75,
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

export const synnaxLight = {
  ...synnaxBase,
  key: "synnax-light",
  name: "Synnax Light",
};

export const synnaxDark = {
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
    text: addOpacityToHex(synnaxBase.colors.white, 80),
  },
};
