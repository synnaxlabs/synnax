import { type schematic } from "@synnaxlabs/client";
import { type MouseEventHandler, type ReactElement } from "react";
import { Toggle } from "../common/toggle";
export interface ActuatorProps extends Omit<Toggle.ButtonProps, "onClick"> {
    specKey: string;
    onClick?: MouseEventHandler<HTMLElement>;
    scale?: number;
    stateOverrides?: schematic.symbol.State[];
}
export declare const Actuator: ({ specKey, enabled, triggered, orientation, scale, className, stateOverrides, ...rest }: ActuatorProps) => ReactElement;
//# sourceMappingURL=Actuator.d.ts.map