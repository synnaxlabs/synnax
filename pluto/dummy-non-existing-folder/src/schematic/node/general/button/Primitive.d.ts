import "./button.css";
import { type schematic } from "@synnaxlabs/client";
import { color } from "@synnaxlabs/x";
import { type MouseEventHandler, type ReactElement } from "react";
interface ButtonProps extends Partial<Pick<schematic.ButtonNodeConfig, "orientation" | "size" | "level" | "mode" | "onClickDelay">> {
    label?: schematic.ButtonNodeConfig["label"];
    color?: color.Crude;
    className?: string;
    onClick?: MouseEventHandler<HTMLButtonElement>;
    onMouseDown?: MouseEventHandler<HTMLButtonElement>;
    onMouseUp?: MouseEventHandler<HTMLButtonElement>;
}
export declare const Button: ({ onClick, onMouseDown, onMouseUp, orientation, label, color: colorVal, size, level, mode, onClickDelay: delay, }: ButtonProps) => ReactElement;
export {};
//# sourceMappingURL=Primitive.d.ts.map