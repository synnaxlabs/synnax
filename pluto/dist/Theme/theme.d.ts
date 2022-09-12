declare type size = number | string;
declare type color = number | string;
declare type typography = {
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
declare const white: color;
declare const black: color;
export declare const aryaLight: {
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
                [key: string]: color[];
                default: color[];
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
};
export declare const aryaDark: {
    name: string;
    colors: {
        gray: {
            m2: color;
            m1: color;
            z: color;
            p1: color;
            p2: color;
        };
        background: color;
        text: color;
        primary: {
            m1: color;
            z: color;
            p1: color;
        };
        error: {
            m1: color;
            z: color;
            p1: color;
        };
        visualization: {
            palettes: {
                [key: string]: color[];
                default: color[];
            };
        };
        white: color;
        black: color;
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
};
export {};
