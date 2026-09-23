import { bounds } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { type DragButtonExtraProps } from "./DragButton";
import { type TextProps } from "./Text";
import { type Control } from "./types";
/** Props for {@link Numeric}. */
export interface NumericProps extends Omit<TextProps, "type" | "onBlur" | "value" | "onChange">, DragButtonExtraProps, Control<number> {
    /** Whether focusing selects the whole value. Defaults to true. */
    selectOnFocus?: boolean;
    /** Whether to show the drag handle that scrubs the value. Defaults to true. */
    showDragHandle?: boolean;
    /** Clamps the committed value. */
    bounds?: bounds.Crude;
    onBlur?: () => void;
    /** Unit suffix shown after the value, e.g. "Hz". */
    units?: string;
    emptyValue?: number;
}
/**
 * A number input. It accepts any math expression `mathjs` can evaluate, so a user can
 * type `2 * 60` or `1 kHz`, and it commits on blur or Enter rather than per keystroke.
 * A drag handle scrubs the value.
 *
 * @example <Input.Numeric value={rate} onChange={setRate} units="Hz" />
 */
export declare const Numeric: ({ ref, onChange, value, dragDirection, showDragHandle, dragScale, selectOnFocus, bounds: propsBounds, onlyChangeOnBlur, resetValue, variant, preview, className, children, disabled, onBlur, units, size, color, emptyValue, ...rest }: NumericProps) => ReactElement;
//# sourceMappingURL=Numeric.d.ts.map