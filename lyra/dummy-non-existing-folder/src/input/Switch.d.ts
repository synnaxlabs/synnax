import "./Switch.css";
import { type ReactElement } from "react";
import { type BooleanProps } from "./Boolean";
/** Props for {@link Switch}. */
export interface SwitchProps extends Omit<BooleanProps, "inputType"> {
}
/** A boolean input drawn as a sliding switch. Use it for a setting that takes effect
 * at once. */
export declare const Switch: (props: SwitchProps) => ReactElement;
//# sourceMappingURL=Switch.d.ts.map