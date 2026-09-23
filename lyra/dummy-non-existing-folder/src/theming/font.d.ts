import { type text } from "@synnaxlabs/x";
export interface UseTypographyReturn extends text.Spec {
    toString: () => string;
    baseSize: number;
    lineHeightPx: number;
    sizePx: number;
}
export declare const useTypography: (level: text.Level) => UseTypographyReturn;
//# sourceMappingURL=font.d.ts.map