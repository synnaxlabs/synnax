import { type location } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement } from "react";
export interface BoundaryProps extends PropsWithChildren<{}> {
    orientation: location.Outer;
    refreshDeps?: unknown;
}
export declare const Boundary: ({ children, orientation, refreshDeps, }: BoundaryProps) => ReactElement | null;
//# sourceMappingURL=Boundary.d.ts.map