import "./Mosaic.css";
import { Flex } from "@synnaxlabs/lyra/flex";
import { type ReactElement } from "react";
export interface LeafProps extends Omit<Flex.BoxProps, "onDrop" | "onDragOver"> {
    /** The key identifying this leaf, passed to the Frame's drop handlers. */
    nodeKey: number;
}
/**
 * Leaf is the haul drop target for the body of one pane of a mosaic. It resolves the
 * drop region from the cursor: an edge splits (showing a mask over the affected half)
 * and the middle drops in the center. Resolved drops are reported to the Frame's
 * onDrop, onCreate, and onFileDrop handlers with this leaf's key. Drops on the pane's
 * tab strip belong to the Tabs.Selector composed inside it: wire it up with {@link
 * useSelectorDropProps} so it claims strip drops (with an insertion index) before they
 * reach the leaf. OS file drags are the exception: the strip rejects them, so they fall
 * through to the leaf, which resolves the strip region to a center drop. The leaf
 * discovers its tabs from the DOM: tabs are elements with a `data-tab-key` attribute,
 * rendered by the Tabs parts composed inside it.
 */
export declare const Leaf: ({ nodeKey, className, children, onDragLeave, ref, ...rest }: LeafProps) => ReactElement;
//# sourceMappingURL=Leaf.d.ts.map