import { type ReactElement } from "react";
import { type Status } from "./status";
import { Text as BaseText } from "../text";
/** Props for {@link Summary}. Pass a whole `status`, or its parts one by one. */
export interface SummaryProps extends Omit<BaseText.TextProps, "wrap" | "variant" | "status">, Partial<Omit<Status, "key">> {
    hideIcon?: boolean;
    status?: Status;
}
/**
 * Renders a status as an {@link Indicator}, its message, and its description. Use it
 * wherever an error or a result has to read inline.
 */
export declare const Summary: ({ level, variant, description, hideIcon, status, className, children, message, color, ...rest }: SummaryProps) => ReactElement;
//# sourceMappingURL=Summary.d.ts.map