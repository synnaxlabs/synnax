import "./setpoint.css";
import { type schematic } from "@synnaxlabs/client";
import { Input as BaseInput } from "@synnaxlabs/lyra/input";
import { type CSSProperties, type ReactElement } from "react";
interface RenderProps extends Partial<Omit<schematic.SetpointNodeConfig, "variant" | "label" | "scale">>, Omit<BaseInput.Control<number>, "value"> {
    className?: string;
    style?: CSSProperties;
}
export declare const Setpoint: ({ orientation, className, style, units, color, onChange, size, disabled, onClickDelay, }: RenderProps) => ReactElement;
export {};
//# sourceMappingURL=Primitive.d.ts.map