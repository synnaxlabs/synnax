import { type location } from "@synnaxlabs/x";
import { type ReactElement } from "react";
export interface RectangleProps {
    orientation: location.Outer;
    left: number;
    top: number;
    right: number;
    bottom: number;
    refreshDeps?: unknown;
}
export declare const Rectangle: ({ orientation, left, top, right, bottom, refreshDeps, }: RectangleProps) => ReactElement;
//# sourceMappingURL=Rectangle.d.ts.map