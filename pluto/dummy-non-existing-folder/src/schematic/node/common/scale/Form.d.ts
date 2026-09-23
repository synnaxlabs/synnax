import { type direction } from "@synnaxlabs/x";
import { type ReactElement } from "react";
export interface FormProps {
    /** Path to the scale config within the symbol's config. */
    path: string;
}
export interface TelemFormProps extends FormProps {
    /** When true, clearing the channel unbinds the scale instead of pinning it to 0. */
    allowNone?: boolean;
}
export declare const TelemForm: ({ path, allowNone, }: TelemFormProps) => ReactElement;
export interface DisplayFieldsProps extends FormProps {
    /** The axis the bar fills along, which the ticks must sit clear of. */
    axis?: direction.Direction;
}
/** Which parts of the scale are drawn, and the sides the ticks and readout sit on. */
export declare const DisplayFields: ({ path, axis, }: DisplayFieldsProps) => ReactElement;
/**
 * Colors of the scale and its labels, and the text size. The fill color is the symbol's
 * own, so the caller renders it against whichever path holds it.
 */
export declare const StyleFields: ({ path }: FormProps) => ReactElement;
//# sourceMappingURL=Form.d.ts.map