import { type ComponentPropsWithoutRef, type ReactElement } from "react";
declare const DELEGATED_EVENTS: {
    readonly onClick: {
        readonly name: "click";
        readonly capture: false;
    };
    readonly onClickCapture: {
        readonly name: "click";
        readonly capture: true;
    };
    readonly onDoubleClick: {
        readonly name: "dblclick";
        readonly capture: false;
    };
    readonly onContextMenu: {
        readonly name: "contextmenu";
        readonly capture: false;
    };
    readonly onPointerDown: {
        readonly name: "pointerdown";
        readonly capture: false;
    };
    readonly onPointerUp: {
        readonly name: "pointerup";
        readonly capture: false;
    };
    readonly onMouseDown: {
        readonly name: "mousedown";
        readonly capture: false;
    };
    readonly onMouseUp: {
        readonly name: "mouseup";
        readonly capture: false;
    };
    readonly onMouseEnter: {
        readonly name: "mouseenter";
        readonly capture: false;
    };
    readonly onMouseLeave: {
        readonly name: "mouseleave";
        readonly capture: false;
    };
    readonly onKeyDown: {
        readonly name: "keydown";
        readonly capture: false;
    };
    readonly onKeyUp: {
        readonly name: "keyup";
        readonly capture: false;
    };
};
type DelegatedEventMap = typeof DELEGATED_EVENTS;
/**
 * HostHandlers mirrors the matching React handler props on a div, but each
 * receives the native event since it fires from a natively bound listener.
 */
export type HostHandlers = {
    [K in keyof DelegatedEventMap]?: (ev: HTMLElementEventMap[DelegatedEventMap[K]["name"]]) => void;
};
export interface OutProps extends Omit<ComponentPropsWithoutRef<"div">, "children" | keyof HostHandlers>, HostHandlers {
    /**
     * itemKey addresses the {@link In} content to host. While null or not yet registered,
     * the Out renders an empty host and attaches the content as soon as it appears.
     */
    itemKey?: string | null;
}
/**
 * Out hosts the content of the {@link In} registered under itemKey at its own
 * position in the DOM. When several Outs address the same key, the last one
 * mounted hosts the content. Handler props ({@link HostHandlers}) fire for
 * interaction with the hosted content; remaining props are forwarded to the
 * host div, whose children the content lays out as.
 */
export declare const Out: ({ itemKey, onClick, onClickCapture, onDoubleClick, onContextMenu, onPointerDown, onPointerUp, onMouseDown, onMouseUp, onMouseEnter, onMouseLeave, onKeyDown, onKeyUp, ...rest }: OutProps) => ReactElement;
export {};
//# sourceMappingURL=Out.d.ts.map