import { type ReactElement } from "react";
import { Aether } from "../../aether";
import { measure } from "./aether";
export interface MeasureProps extends Aether.ComponentProps {
    mode?: measure.Mode;
    onModeChange?: (mode: measure.Mode) => void;
}
export declare const Measure: ({ aetherKey, mode, onModeChange, }: MeasureProps) => ReactElement;
//# sourceMappingURL=Measure.d.ts.map