import "./Item.css";
import { type ReactNode } from "react";
import { Flex } from "../flex";
import { type Status } from "../status";
/**
 * Resolves a control's aria-labelledby: an explicit value wins, then the enclosing
 * {@link Item}'s label id, unless the control carries its own aria-label.
 */
export declare const useLabelledBy: (props: {
    "aria-label"?: string;
    "aria-labelledby"?: string;
}) => string | undefined;
export interface ItemProps extends Flex.BoxProps {
    label?: string;
    required?: boolean;
    showLabel?: boolean;
    helpText?: string;
    padHelpText?: boolean;
    status?: Status.Variant;
    showHelpText?: boolean;
}
export declare const Item: ({ label, showLabel, helpText, direction, x, y, className, children, required, align, gap: size, padHelpText, status, showHelpText, ...rest }: ItemProps) => ReactNode;
//# sourceMappingURL=Item.d.ts.map