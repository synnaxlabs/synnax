import "./Indicator.css";
import { type PropsWithChildren, type ReactElement } from "react";
import { type z } from "zod";
import { control } from "./aether";
export interface IndicatorProps extends Omit<z.input<typeof control.indicatorStateZ>, "status" | "color">, PropsWithChildren {
}
export interface StatusDetails {
    color?: string;
}
export declare const Indicator: ({ colorSource, statusSource, }: IndicatorProps) => ReactElement;
//# sourceMappingURL=Indicator.d.ts.map