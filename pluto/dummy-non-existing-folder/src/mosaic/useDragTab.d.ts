import { type record } from "@synnaxlabs/x";
import { type DragEvent } from "react";
export interface UseDragTabReturn {
    /**
     * startDrag begins hauling the tab with the given key. Call from a draggable
     * tab element's onDragStart handler. Anything passed as data reaches the drop
     * handler untouched, including a handler in another window.
     */
    startDrag: (e: DragEvent<HTMLElement>, tabKey: string, data?: record.Unknown) => void;
    /** onDragEnd resolves the drag. Spread onto the same draggable tab element. */
    onDragEnd: (e: DragEvent<HTMLElement>) => void;
}
/**
 * useDragTab wires tab elements up as mosaic drag sources. The returned handlers
 * can be shared by every tab in a strip: pass the tab's key to startDrag from its
 * onDragStart, and attach onDragEnd alongside it.
 */
export declare const useDragTab: () => UseDragTabReturn;
//# sourceMappingURL=useDragTab.d.ts.map