import { TypographyDefinition } from "../Atoms/Typography";

type Size = number | string;
type Color = string;

export interface Theme {
  name: string;
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
    h1: TypographyDefinition;
    h2: TypographyDefinition;
    h3: TypographyDefinition;
    h4: TypographyDefinition;
    h5: TypographyDefinition;
    p: TypographyDefinition;
    small: TypographyDefinition;
  };
}

const white: Color = "#FFFFFF";
const black: Color = "#171716";
const fontFamily = "Inter, sans-serif";
const baseSize: Size = 6;

const synnaxBase: Theme = {
  name: "synnax-base",
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
      size: baseSize * 7,
      weight: "500",
      lineHeight: baseSize * 8,
    },
    h2: {
      size: baseSize * 6,
      weight: "medium",
      lineHeight: baseSize * 7,
    },
    h3: {
      size: baseSize * 4,
      weight: "medium",
      lineHeight: baseSize * 5,
    },
    h4: {
      size: baseSize * 3.5,
      weight: "medium",
      lineHeight: baseSize * 4,
    },
    h5: {
      size: baseSize * 2,
      weight: "medium",
      lineHeight: baseSize * 3,
      textTransform: "uppercase",
    },
    p: {
      size: baseSize * 2.5,
      weight: "regular",
      lineHeight: baseSize * 3,
    },
    small: {
      size: baseSize * 2,
      weight: 300,
      lineHeight: baseSize * (2 + 1 / 3),
    },
  },
};

export const synnaxLight = {
  ...synnaxBase,
  name: "synnax-light",
};

export const synnaxDark = {
  ...synnaxBase,
  name: "synnax-dark",
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
    border: synnaxBase.colors.gray.p1,
    background: synnaxBase.colors.black,
    text: synnaxBase.colors.white,
  },
};
