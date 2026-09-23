import "./Legend.css";
import { type color } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { Legend as Base } from "../../vis/legend";
export interface LegendProps extends Omit<Base.SimpleProps, "data" | "onColorChange" | "onLabelChange" | "onVisibleChange"> {
    colors?: Record<string, color.Color>;
    onColorsChange?: (colors: Record<string, color.Color>) => void;
}
export declare const Legend: (props: LegendProps) => ReactElement | null;
//# sourceMappingURL=Legend.d.ts.map