import { type schematic } from "@synnaxlabs/client";
import { type location } from "@synnaxlabs/x";
import { type ReactElement } from "react";
export interface StaticProps {
    specKey: string;
    orientation?: location.Outer;
    scale?: number;
    className?: string;
    stateOverrides?: schematic.symbol.State[];
}
export declare const Static: ({ specKey, orientation, scale, className, stateOverrides, }: StaticProps) => ReactElement;
//# sourceMappingURL=Static.d.ts.map