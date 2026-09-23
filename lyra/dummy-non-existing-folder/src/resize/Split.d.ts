import { type ReactElement, type ReactNode } from "react";
import { Flex } from "../flex";
/** Props for the {@link Split} component. */
export interface SplitProps extends Omit<Flex.BoxProps, "size" | "onResize" | "children"> {
    /** The two panes to lay out, the first on the left/top and the second on the
     * right/bottom. Exactly two children are required. */
    children: [ReactNode, ReactNode];
    /**
     * The fraction of the container occupied by the first pane, as a decimal between 0 and
     * 1. Acts as the source of truth whenever the handle is not being actively dragged;
     * while a drag is in progress the panes render their transient drag ratio instead.
     * Defaults to 0.5.
     */
    size?: number;
    /** The smallest size, in pixels, either pane can be resized to. Defaults to 100. */
    minSize?: number;
    /**
     * Called continuously while the handle is being dragged with the first pane's live
     * fraction (0..1). Use for transient, per-frame side effects; do not persist from here.
     */
    onResize?: (size: number) => void;
    /**
     * Called once when a drag gesture ends with the committed fraction (0..1). This is the
     * callback to persist the new ratio to the source of truth.
     */
    onResizeEnd?: (size: number) => void;
    /**
     * Removes the drag handle, so the ratio can only change through `size`. Use where the
     * viewer may not persist a new ratio; the divider between the panes stays.
     */
    disabled?: boolean;
}
/**
 * A pair of panes that can be resized relative to one another by dragging the handle
 * between them. Both panes are sized as percentages of the container, so the split
 * ratio is preserved as the container resizes.
 * @param props - The component props. Unlisted props are forwarded to the underlying
 * {@link Flex.Box}. Exactly two children should be provided; the first is placed on the
 * left/top and the second on the right/bottom.
 */
export declare const Split: ({ onResize, onResizeEnd, children, className, size, minSize, align, direction: propsDirection, x, y, pack, disabled, ...rest }: SplitProps) => ReactElement;
//# sourceMappingURL=Split.d.ts.map