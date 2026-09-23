import "./Dialog.css";
import { type ReactElement, type ReactNode, type Ref } from "react";
import { position } from "../position";
interface ChildProps {
    ref?: Ref<HTMLElement>;
    "aria-describedby"?: string;
    "aria-label"?: string;
    children?: ReactNode;
    onPointerEnter?: (e: React.PointerEvent<HTMLElement>) => void;
    onPointerLeave?: (e: React.PointerEvent<HTMLElement>) => void;
    onPointerDown?: (e: React.PointerEvent<HTMLElement>) => void;
    onFocus?: (e: React.FocusEvent<HTMLElement>) => void;
    onBlur?: (e: React.FocusEvent<HTMLElement>) => void;
}
export interface DialogProps {
    location?: position.Location;
    hide?: boolean;
    children: [ReactNode, ReactElement<ChildProps>];
}
/** The props a component adds to take an optional tooltip. */
export interface ExtensionProps {
    /** The tooltip content. Nothing shows while this is unset. */
    tooltip?: DialogProps["children"][0];
    /** The preferred location relative to the element. Chosen by position when unset. */
    tooltipLocation?: DialogProps["location"];
    /** Forces the tooltip to stay hidden. */
    hideTooltip?: DialogProps["hide"];
}
/**
 * A tooltip that appears when the user hovers or keyboard-focuses an element.
 *
 * @param props.children - The tooltip's content, followed by the element to attach
 * the tooltip to.
 * @param props.location - The preferred location for the tooltip relative to the
 * element. If unspecified or the tooltip would overflow the window, the best
 * location is chosen automatically.
 * @param props.hide - Force the tooltip to remain hidden.
 * @default false.
 */
export declare const Dialog: ({ children, location: locationProp, hide, }: DialogProps) => ReactElement;
export declare const formatTip: (tip: ReactNode) => ReactNode;
export {};
//# sourceMappingURL=Dialog.d.ts.map