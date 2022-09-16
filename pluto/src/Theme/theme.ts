type size = number | string;
type color = number | string;
type typography = {
  size: size;
  weight: size;
  lineHeight: number;
  textTransform?: string;
};

export interface Theme {
  name: string;
  colors: {
    primary: {
      m1: color;
      z: color;
      p1: color;
    };
    gray: {
      m2: color;
      m1: color;
      z: color;
      p1: color;
      p2: color;
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
const black: color = "#212429";
const fontFamily = "Roboto, sans-serif";
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
      p2: "#51565e",
      p1: "#61636b",
      z: "#ACB5BD",
      m1: "#b2b2b2",
      m2: "#c9c9c9",
    },
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
      lineHeight: baseSize * 3.5,
    },
    small: {
      size: baseSize * 2,
      weight: "regular",
      lineHeight: baseSize * 3,
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
      m2: synnaxBase.colors.gray.p2,
      m1: synnaxBase.colors.gray.p1,
      z: synnaxBase.colors.gray.z,
      p1: synnaxBase.colors.gray.m1,
      p2: synnaxBase.colors.gray.m2,
    },
    background: synnaxBase.colors.black,
    text: synnaxBase.colors.white,
  },
};
