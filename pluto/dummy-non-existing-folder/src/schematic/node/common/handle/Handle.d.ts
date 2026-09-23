import { type location } from "@synnaxlabs/x";
import { type HandleProps as RFHandleProps, type Position as RFPosition } from "@xyflow/react";
import { type ReactElement } from "react";
export interface HandleProps extends Omit<RFHandleProps, "type" | "position"> {
    orientation: location.Outer;
    location: location.Outer;
    position?: RFPosition;
    preventAutoAdjust?: boolean;
    swap?: boolean;
    left: number;
    top: number;
    id: string;
}
export declare const Handle: ({ location, orientation, preventAutoAdjust, left, swap: swapPos, top, style, ...rest }: HandleProps) => ReactElement;
//# sourceMappingURL=Handle.d.ts.map