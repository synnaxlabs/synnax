import { color } from "@synnaxlabs/x";
import { type BaseEdgeProps } from "@xyflow/react";
import { type ReactElement } from "react";
export interface BaseProps extends Omit<BaseEdgeProps, "color"> {
    color?: color.Crude;
}
export declare const Base: ({ style: baseStyle, color: stroke, className, ...props }: BaseProps) => ReactElement;
//# sourceMappingURL=Base.d.ts.map