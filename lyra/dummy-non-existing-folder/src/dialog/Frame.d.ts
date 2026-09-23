import { location, type state as xstate } from "@synnaxlabs/x";
import { type CSSProperties, type ReactElement, type RefCallback } from "react";
import { type Component } from "../component";
import { Flex } from "../flex";
import { position } from "../position";
/**
 * How a dialog attaches to its trigger.
 *
 * - `connected`: butts against the trigger and matches its width.
 * - `floating`: sits near the trigger at its own width.
 * - `modal`: centers over the page behind a backdrop.
 */
export type Variant = "connected" | "floating" | "modal";
/** How far a modal dialog rises from its resting place as the viewport tightens. */
export type ModalPosition = "slammed" | "shifted" | "base";
/** Props for the {@link Frame} component. */
export interface FrameProps extends Omit<Flex.BoxProps, "ref" | "reverse" | "size" | "empty"> {
    initialVisible?: boolean;
    /** Set it to control visibility from outside. The frame owns it when left unset. */
    visible?: boolean;
    onVisibleChange?: xstate.Setter<boolean>;
    /** Pins the dialog to one corner pairing instead of choosing one that fits. */
    location?: position.LocationPreference;
    variant?: Variant;
    maxHeight?: Component.Size | number;
    zIndex?: number;
    modalPosition?: ModalPosition;
}
interface State {
    targetCorner: location.XY;
    dialogCorner: location.XY;
    modalPosition: ModalPosition;
    style: CSSProperties;
}
/** State the enclosing {@link Frame} publishes to its trigger and dialog. */
export interface ContextValue {
    close: () => void;
    open: () => void;
    toggle: () => void;
    visible: boolean;
    variant: Variant;
    /** The corner of the trigger the dialog opens from. */
    location: location.XY;
}
declare const useContext: () => ContextValue;
export { useContext };
interface InternalContextValue extends Pick<State, "targetCorner" | "dialogCorner" | "style" | "modalPosition"> {
    ref: RefCallback<HTMLDivElement>;
    /** Ties the portaled dialog back to this frame for click-outside checks. */
    id: string;
}
declare const useInternalContext: (hookOrComponentName: string) => InternalContextValue;
export { useInternalContext };
export declare function Frame({ children, location: propsLocation, onPointerEnter, className, variant, maxHeight, zIndex, initialVisible, visible: propsVisible, onVisibleChange: propsOnVisibleChange, modalPosition: propsModalPosition, ...rest }: FrameProps): ReactElement;
export declare namespace Frame {
    var displayName: string;
}
//# sourceMappingURL=Frame.d.ts.map