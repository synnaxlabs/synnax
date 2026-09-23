import { box, type CrudeTimeSpan, type direction } from "@synnaxlabs/x";
import { type RefCallback } from "react";
/** Options for {@link useResize}. */
export interface UseResizeOpts {
    /** A list of triggers that should cause the callback to be called. */
    triggers?: direction.Direction[];
    /** Debounce the resize event by this duration. Useful for preventing expensive
    renders until resizing has stopped. */
    debounce?: CrudeTimeSpan;
    /** If false, the hook wont observe the element. Defaults to true. */
    enabled?: boolean;
}
/**
 * Called with the element's new bounding box and the element itself. Fires on the
 * observer's callback, so it runs outside React's render pass.
 */
export type UseResizeHandler = <E extends HTMLElement>(box: box.Box, el: E) => void;
/**
 * Tracks the dimensions of an element and executes a callback when they change.
 * @param onResize - A callback that receives a box representing the dimensions and
 * position of the element.
 * @param opts - Options for the hook. See UseResizeOpts.
 * @returns a ref callback to attach to the desire element.
 */
export declare const useResize: <E extends HTMLElement>(onResize: UseResizeHandler, opts?: UseResizeOpts) => RefCallback<E>;
/** Options for {@link useWindowResize}. */
export interface UseWindowResizeOptions {
    /** If false, the hook stops listening. Defaults to true. */
    enabled?: boolean;
}
/**
 * Calls onResize with the viewport's box whenever the window resizes. The handler is
 * read through a ref, so a fresh closure each render costs nothing.
 */
export declare const useWindowResize: (onResize: UseResizeHandler, opts?: UseWindowResizeOptions) => void;
//# sourceMappingURL=useResize.d.ts.map