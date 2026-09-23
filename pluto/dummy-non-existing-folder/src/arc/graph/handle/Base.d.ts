import { type location } from "@synnaxlabs/x";
import { type HandleProps, Position } from "@xyflow/react";
export interface BaseProps extends Omit<HandleProps, "position"> {
    location: location.Outer;
}
export declare const locationToRFPosition: (location: location.Outer) => Position;
export declare const Base: ({ location, className, ...props }: BaseProps) => import("react").JSX.Element | null;
//# sourceMappingURL=Base.d.ts.map