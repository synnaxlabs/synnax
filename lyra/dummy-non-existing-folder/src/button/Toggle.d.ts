import { type ReactElement } from "react";
import { type ButtonProps } from "./Button";
import { type Input } from "../input";
export interface ToggleProps extends Input.Control<boolean, boolean>, Omit<ButtonProps, "value" | "onChange" | "variant"> {
    rightClickToggle?: boolean;
}
export declare const Toggle: ({ value, onClick, onChange, rightClickToggle, className, ...rest }: ToggleProps) => ReactElement;
//# sourceMappingURL=Toggle.d.ts.map