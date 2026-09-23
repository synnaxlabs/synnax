import { type CrudeTimeSpan, TimeSpan } from "@synnaxlabs/x";
import { type KeyboardEventHandler, type MouseEventHandler } from "react";
export interface UseHoldProps<E extends Element> {
    onClick?: MouseEventHandler<E>;
    onMouseDown?: MouseEventHandler<E>;
    onClickDelay?: CrudeTimeSpan;
    /** Ignores presses and cancels a hold in progress. */
    disabled?: boolean;
}
export interface UseHoldReturn<E extends Element> {
    /** The activation delay. Zero means a click actuates at once. */
    delay: TimeSpan;
    /** Whether a primary-button or activation-key press is in progress. */
    pressed: boolean;
    onClick: MouseEventHandler<E>;
    onMouseDown: MouseEventHandler<E>;
    /** Paints the pressed state for Space and Enter. Never starts a hold. */
    onKeyDown: KeyboardEventHandler<E>;
    onKeyUp: KeyboardEventHandler<E>;
}
/**
 * Gates onClick behind a press-and-hold of onClickDelay. Only a primary press starts
 * the hold. A release, drag, window blur, unmount, or disable cancels it. The hold
 * releases itself when it fires, so the control does not read as pressed after the
 * actuation changes its state.
 */
export declare const useHold: <E extends Element>({ onClick, onMouseDown, onClickDelay, disabled, }: UseHoldProps<E>) => UseHoldReturn<E>;
//# sourceMappingURL=useHold.d.ts.map