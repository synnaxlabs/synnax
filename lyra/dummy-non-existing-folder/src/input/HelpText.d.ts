import "./HelpText.css";
import { type ReactElement } from "react";
import { Status } from "../status";
import { Text } from "../text";
/** Props for the {@link HelpText} component. */
export interface HelpTextProps extends Omit<Text.TextProps<"small">, "level" | "ref" | "variant"> {
    variant?: Status.Variant;
}
/**
 * Hint or error text below an input, colored by its status variant. Prefer {@link Item}
 * with a `helpText` prop; reach for this only to place the text yourself.
 */
export declare const HelpText: ({ className, variant, ...rest }: HelpTextProps) => ReactElement;
//# sourceMappingURL=HelpText.d.ts.map