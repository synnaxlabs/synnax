import { type location } from "@synnaxlabs/x";
import { Position as RFPosition } from "@xyflow/react";
export declare const ORIENTATION_RF_POSITIONS: Record<location.Outer, Record<location.Outer, RFPosition>>;
export declare const smart: (position: location.Outer, orientation: location.Outer) => RFPosition;
export declare const swap: (position: RFPosition, bypass?: boolean) => RFPosition;
export declare const adjust: (top: number, left: number, orientation: location.Outer, prevent?: boolean) => {
    left: number;
    top: number;
};
//# sourceMappingURL=position.d.ts.map