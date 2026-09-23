import { type xy } from "@synnaxlabs/x";
import { type Mock } from "vitest";
/**
 * Fires a drag event of the given type at the given cursor position on target. jsdom
 * has no DragEvent, so testing-library falls back to a plain Event and the coordinates
 * in the init are lost; they are defined on the event instead.
 */
export declare const fireDragEvent: (target: Element, type: "dragOver" | "drop", cursor: xy.XY) => void;
/**
 * Fires a pointer down event at the given cursor position on target. jsdom has no
 * PointerEvent, so testing-library falls back to a plain Event and the coordinates in
 * the init are lost; they are defined on the event instead.
 */
export declare const firePointerDown: (target: Element, cursor: xy.XY) => void;
/**
 * Gives every element a fixed box in jsdom, which otherwise reports zero for all of
 * them. A virtualized list measures its scroll container this way, so without it no
 * item is ever inside the window.
 */
export declare const mockGeometry: (width: number, height: number) => void;
export declare const mockBoundingClientRect: (top: number, left: number, width: number, height: number) => Mock<typeof HTMLElement.prototype.getBoundingClientRect>;
//# sourceMappingURL=dom.d.ts.map