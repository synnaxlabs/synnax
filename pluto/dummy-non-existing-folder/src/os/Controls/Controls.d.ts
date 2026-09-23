import { type runtime } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { type InternalControlsProps } from "./types";
export interface ControlsProps extends InternalControlsProps {
    visibleIfOS?: runtime.OS;
}
export declare const Controls: ({ forceOS, visibleIfOS, ...rest }: ControlsProps) => ReactElement | null;
//# sourceMappingURL=Controls.d.ts.map