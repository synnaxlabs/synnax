import "./Input.css";
import { type ReactElement, type ReactNode } from "react";
import { Button } from "../button";
import { type InputProps, type Variant } from "./types";
import { type Status } from "../status";
import { type Tooltip } from "../tooltip";
export interface TextProps extends InputProps<string>, Omit<Button.ExtensionProps, "variant">, Tooltip.ExtensionProps {
    selectOnFocus?: boolean;
    centerPlaceholder?: boolean;
    resetOnBlurIfEmpty?: boolean;
    status?: Status.Variant;
    variant?: Variant;
    placeholder?: ReactNode;
    children?: ReactNode;
    endContent?: ReactNode;
    startContent?: ReactNode;
    onlyChangeOnBlur?: boolean;
    area?: boolean;
    flush?: boolean;
}
/**
 * A controlled string input component.
 *
 * @param props - The props for the input component. Unlisted props are passed to the
 * underlying input element.
 * @param props.value - The value of the input.
 * @param props.onChange - A function to call when the input value changes.
 * @param props.size - The size of the input: "small" | "medium" | "large".
 * @param props.selectOnFocus - Whether the input should select its contents when focused.
 * @param props.centerPlaceholder - Whether the placeholder should be centered.
 * @param props.resetOnBlurIfEmpty - Whether the input should reset to its previous value if
 * blurred while empty.
 * @param props.onlyChangeOnBlur - If true, the input will only call `onChange` when the
 * user blurs the input or the user presses 'Enter'.
 * @param props.flush - Marks the input as its surface's sole keystroke target,
 * removing the focus ring and widening the content inset.
 */
export declare const Text: ({ size, ref, value, onChange, className, onFocus, onKeyDown, selectOnFocus, centerPlaceholder, placeholder, variant, level, onBlur, disabled, resetOnBlurIfEmpty, status, weight, style, color: pColor, sharp, onlyChangeOnBlur, endContent, full, children, grow, shrink, borderColor, borderWidth, bordered, rounded, tabIndex, trigger, triggerIndicator, textColor, textVariant, preventClick, onClickDelay, startContent, tooltip, tooltipLocation, hideTooltip, reveal, area, flush, preview, propagateClick, ...rest }: TextProps) => ReactElement;
//# sourceMappingURL=Text.d.ts.map