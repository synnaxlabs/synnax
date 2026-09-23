import "./Label.css";
import { type DetailedHTMLProps, type HTMLAttributes, type ReactElement } from "react";
/** Props for the {@link Label} component. */
export interface LabelProps extends DetailedHTMLProps<HTMLAttributes<HTMLLabelElement>, HTMLLabelElement> {
    required?: boolean;
}
/**
 * A styled `label` element, marking a required field with an asterisk. Prefer
 * {@link Item} with a `label` prop; reach for this only to place the label yourself.
 */
export declare const Label: ({ className, required, children, ...rest }: LabelProps) => ReactElement;
//# sourceMappingURL=Label.d.ts.map