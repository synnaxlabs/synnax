type size = number | string;
type color = number | string;
type typography = {
  size: size;
  weight: size;
  lineHeight: number;
  textTransform?: string;
};

export type FontLevel = "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";

export interface Theme {
  name: string;
  colors: {
    border: color;
    primary: {
      m1: color;
      z: color;
      p1: color;
    };
    gray: {
      m3: color;
      m2: color;
      m1: color;
      m0: color;
      p0: color;
      p1: color;
      p2: color;
      p3: color;
    };
    error: {
      m1: color;
      z: color;
      p1: color;
    };
    visualization: {
      palettes: {
        default: color[];
        [key: string]: color[];
      };
    };
    white: color;
    black: color;
    background: color;
    text: color;
  };
  sizes: {
    base: number;
    border: {
      radius: size;
      width: size;
    };
  };
  typography: {
    family: string;
    h1: typography;
    h2: typography;
    h3: typography;
    h4: typography;
    h5: typography;
    p: typography;
    small: typography;
  };
}

const white: color = "#FFFFFF";
const black: color = "#171716";
const fontFamily = "Inter, sans-serif";
const baseSize: size = 6;

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
      m2: "#F5F5F1",
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
      weight: "regular",
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
