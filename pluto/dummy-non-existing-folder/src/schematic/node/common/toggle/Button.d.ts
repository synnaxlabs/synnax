import { color, type CrudeTimeSpan } from "@synnaxlabs/x";
import { type ComponentPropsWithRef, type ReactElement } from "react";
import { type OrientableProps } from "../primitive/orientable";
export interface ButtonBaseProps extends Omit<ComponentPropsWithRef<"button">, "color" | "value"> {
    triggered?: boolean;
    enabled?: boolean;
    color?: color.Crude;
    onClickDelay?: CrudeTimeSpan;
}
export interface ButtonProps extends ButtonBaseProps, OrientableProps {
}
export declare const Button: ({ className, enabled, triggered, orientation, color: colorVal, onClickDelay, onClick, onMouseDown, disabled, style, children, ...rest }: ButtonProps) => ReactElement;
//# sourceMappingURL=Button.d.ts.map