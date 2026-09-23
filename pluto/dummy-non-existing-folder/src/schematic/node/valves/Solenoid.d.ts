import "./Solenoid.css";
import { type ReactElement } from "react";
import { Primitive } from "../common/primitive";
import { Toggle } from "../common/toggle";
export interface Props extends Toggle.ButtonProps, Primitive.SVGBasedProps {
    normallyOpen?: boolean;
}
export declare const Solenoid: ({ className, color, orientation, normallyOpen, scale, ...rest }: Props) => ReactElement;
//# sourceMappingURL=Solenoid.d.ts.map