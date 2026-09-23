import "./drag.css";
import { box, xy } from "@synnaxlabs/x";
import { type PointerEvent as ReactPointerEvent } from "react";
import { Triggers } from "../triggers";
/**
 * Class applying the `touch-action`/`user-select` rules a cursor-drag element needs. Add
 * it to whatever element receives the handler returned by {@link useDrag}.
 */
export declare const DRAG_CLASS: string;
export interface UseDragProps {
    /**
     * Called once when the pointer has moved past the activation threshold, marking the
     * true start of a drag. Receives the press location, the mouse button, and the element
     * the gesture started on.
     */
    onStart?: (loc: xy.XY, mouseKey: Triggers.Key, el: HTMLElement) => void;
    /** Called on every pointer move during a drag with the box from start to current. */
    onMove?: (box: box.Box, mouseKey: Triggers.Key, e: PointerEvent) => void;
    /** Called once when the drag ends (pointer up or cancel). Not called for a click. */
    onEnd?: (box: box.Box, mouseKey: Triggers.Key, e: PointerEvent) => void;
    /**
     * Cancels the press's default action. WebKit otherwise anchors a text selection on
     * the press and extends it across whatever the pointer passes over. Only for an
     * element with nothing focusable inside it: the cancelled default also places the
     * caret and moves focus.
     */
    preventDefault?: boolean;
}
export type UseDragStart = (e: ReactPointerEvent) => void;
/**
 * Returns a handler to spread onto an element's `onPointerDown` that drives a pointer
 * drag. The gesture activates only after the pointer moves past {@link DRAG_THRESHOLD},
 * so a stationary press stays a click. Once active, the pointer is captured to the
 * element so moves and the terminating up/cancel are delivered even outside its bounds.
 * Add {@link DRAG_CLASS} to the element so touch gestures and text selection don't
 * pre-empt the drag.
 */
export declare const useDrag: ({ onMove, onStart, onEnd, preventDefault, }: UseDragProps) => UseDragStart;
//# sourceMappingURL=drag.d.ts.map