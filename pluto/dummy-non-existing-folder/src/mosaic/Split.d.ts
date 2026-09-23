import { Resize } from "@synnaxlabs/lyra/resize";
import { type ReactElement } from "react";
export interface SplitProps extends Omit<Resize.SplitProps, "onResizeEnd"> {
    /** The key identifying this split, passed to the Frame's onResize handler. */
    nodeKey: number;
}
/**
 * Split lays out two panes of a mosaic side by side with a draggable handle
 * between them. When a handle drag ends, the Frame's onResize handler is called
 * with the split's key and the committed ratio. Children are typically Leaf parts
 * or nested Splits.
 */
export declare const Split: ({ nodeKey, ...rest }: SplitProps) => ReactElement;
//# sourceMappingURL=Split.d.ts.map