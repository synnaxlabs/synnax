import "./Trigger.css";
import { type ReactElement } from "react";
import { Button } from "../button";
/** Props for {@link Trigger}. */
export interface TriggerProps extends Button.ButtonProps {
    hideCaret?: boolean;
}
/**
 * The button that opens and closes the enclosing {@link Frame}, carrying a caret that
 * turns with the dialog.
 */
export declare const Trigger: ({ onClick, className, hideCaret, children, variant: triggerVariant, preview, ...rest }: TriggerProps) => ReactElement;
//# sourceMappingURL=Trigger.d.ts.map