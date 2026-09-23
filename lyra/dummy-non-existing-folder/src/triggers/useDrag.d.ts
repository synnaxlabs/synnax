import { box, xy } from "@synnaxlabs/x";
import { type RefObject } from "react";
import { type Stage, type Trigger } from "./triggers";
/** The event {@link useDrag} hands its callback. */
export interface DragEvent {
    stage: Stage;
    /** The region swept from the press point to the cursor. */
    box: box.Box;
    cursor: xy.XY;
    triggers: Trigger[];
}
export type DragCallback = (props: DragEvent) => void;
/** Props for {@link useDrag}. */
export interface UseDragProps {
    /** The drag starts only on a press inside this element. */
    bound: RefObject<HTMLElement | null>;
    /** Buttons that start a drag. Defaults to left and right. */
    triggers?: Trigger[];
    onDrag: DragCallback;
    loose?: boolean;
}
/**
 * Tracks a press-move-release drag that starts inside the bound element. The callback
 * runs on press, on every move, and on release, and the move keeps tracking past the
 * element's edge.
 */
export declare const useDrag: ({ onDrag, triggers, bound, loose, }: UseDragProps) => void;
//# sourceMappingURL=useDrag.d.ts.map