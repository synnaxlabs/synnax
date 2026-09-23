import "./line.css";
import { color, type xy } from "@synnaxlabs/x";
import { type ReactElement } from "react";
export interface LineProps {
    className?: string;
    color?: color.Crude;
    start?: xy.XY;
    end?: xy.XY;
    strokeWidth?: number;
}
export declare const Line: ({ className, color: colorVal, start, end, strokeWidth, }: LineProps) => ReactElement;
//# sourceMappingURL=Primitive.d.ts.map