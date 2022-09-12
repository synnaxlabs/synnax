import { SVGProps } from "react";
import { textLevel } from "../../Atoms/Typography/Text";
export interface Metric {
    name: string;
    value: number;
    max: number;
    units: string;
}
export interface Title {
    text: string;
    textLevel: textLevel;
}
export interface HexagonBarProps extends SVGProps<any> {
    title?: Title;
    strokeWidth: number;
    metrics: Metric[];
}
export declare const HexagonBar: ({ strokeWidth, metrics, title, ...props }: HexagonBarProps) => JSX.Element;
