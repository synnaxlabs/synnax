import "./DragButton.css";
import { type direction, xy } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { Button } from "../button";
import { type Control } from "./types";
/** Drag behavior an input passes down to its {@link DragButton}. */
export interface DragButtonExtraProps {
    direction?: direction.Crude;
    /** Restricts scrubbing to one axis. Both axes are live when unset. */
    dragDirection?: direction.Crude;
    /** Value change per pixel dragged, per axis. */
    dragScale?: xy.Crude | number;
    /** Pixels the pointer must travel before scrubbing starts. */
    dragThreshold?: xy.Crude | number;
    /** Value a double click restores. */
    resetValue?: number;
}
export interface DragButtonProps extends Omit<Button.ButtonProps, "direction" | "onChange" | "onDragStart" | "children" | "value" | "onDragEnd" | "onBlur">, Control<number>, DragButtonExtraProps {
    onDragEnd?: (value: number) => void;
    onBlur?: () => void;
}
/**
 * A handle that scrubs a number as the pointer drags across it, horizontally by the x
 * scale and vertically by the y. A double click restores `resetValue`.
 */
export declare const DragButton: ({ direction, className, dragScale, dragThreshold, dragDirection, onChange, value, resetValue, onDragEnd, disabled, ...rest }: DragButtonProps) => ReactElement;
//# sourceMappingURL=DragButton.d.ts.map