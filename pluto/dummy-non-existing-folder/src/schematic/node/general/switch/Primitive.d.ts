import "./switch.css";
import { type MouseEventHandler, type ReactElement } from "react";
import { type Toggle } from "../../common/toggle";
export interface Props extends Omit<Toggle.ButtonProps, "onClick" | "onMouseDown"> {
    onClick?: MouseEventHandler<HTMLElement>;
    scale?: number;
}
export declare const Switch: ({ enabled, onClick, onClickDelay, orientation, color: colorVal, scale, disabled, }: Props) => ReactElement;
//# sourceMappingURL=Primitive.d.ts.map