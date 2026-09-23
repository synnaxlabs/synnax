import { type RefObject } from "react";
import { type UseDragProps } from "./drag";
export interface UseVirtualDragProps extends UseDragProps {
    ref: RefObject<HTMLElement | null>;
}
/**
 * A variant of {@link useDrag} that attaches its own `pointerdown` listener to a ref'd
 * element, for cases where the gesture initiator cannot receive an `onPointerDown` prop
 * directly. Activates immediately on press (no movement threshold).
 */
export declare const useVirtualDrag: ({ ref, onMove, onStart, onEnd, }: UseVirtualDragProps) => void;
//# sourceMappingURL=virtual.d.ts.map