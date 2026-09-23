import { type ReactElement } from "react";
import { Legend as Base } from "../vis/legend";
import { type EntriesProps } from "../vis/legend/Entries";
export interface LegendProps extends Omit<Base.SimpleProps, "data" | "onColorChange" | "onLabelChange" | "onVisibleChange"> {
    variant?: "floating" | "fixed";
    onLineColorChange?: EntriesProps["onColorChange"];
    onLineLabelChange?: EntriesProps["onLabelChange"];
    onLineVisibleChange?: EntriesProps["onVisibleChange"];
}
export declare const Legend: ({ variant, ...rest }: LegendProps) => ReactElement;
//# sourceMappingURL=Legend.d.ts.map