import { type ReactElement, type ReactNode } from "react";
import { Button } from "../button";
import { type InputProps } from "./types";
export interface BooleanProps extends Omit<InputProps<boolean>, "onClick">, Omit<Button.ExtensionProps, "variant"> {
    inputType: "switch" | "checkbox";
    onClick?: (e: React.MouseEvent<HTMLElement>) => void;
    /** When set, replaces the box indicator with this glyph in the checked state. */
    checkedIcon?: ReactNode;
    /** Glyph shown in the unchecked state when {@link checkedIcon} is set. */
    uncheckedIcon?: ReactNode;
}
/**
 * Base Boolean input component for switch and checkbox variants.
 */
export declare const Boolean: ({ ref, className, value, reveal, disabled, onChange, inputType, size, variant, preview, style, color, borderColor, borderWidth, bordered, rounded, background, textColor, onClick, checkedIcon, uncheckedIcon, tooltip, tooltipLocation, hideTooltip, ...rest }: BooleanProps) => ReactElement;
//# sourceMappingURL=Boolean.d.ts.map