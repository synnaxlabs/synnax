import "./select.css";
import { Button as BaseButton } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { type Input } from "@synnaxlabs/lyra/input";
import { type location } from "@synnaxlabs/x";
import { type ReactElement } from "react";
export interface Value {
    inner: location.Outer;
    outer: location.Location;
}
export interface SelectProps extends Input.Control<Value>, Omit<Flex.BoxProps, "value" | "onChange"> {
    hideOuter?: boolean;
    showOuterCenter?: boolean;
    hideInner?: boolean;
}
export declare const Select: ({ value, hideOuter, showOuterCenter, hideInner, onChange, }: SelectProps) => ReactElement;
export interface ButtonProps extends Omit<BaseButton.ButtonProps, "children"> {
    selected: boolean;
}
export declare const Button: ({ selected, className, ...rest }: ButtonProps) => ReactElement;
//# sourceMappingURL=select.d.ts.map