import "./Items.css";
import { type record } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode } from "react";
import { Flex } from "../flex";
import { type ItemRenderProp } from "./Item";
/** Props for {@link Items}. */
export interface ItemsProps<K extends record.Key = record.Key> extends Omit<Flex.BoxProps, "children" | "ref"> {
    /** Renders one item. It is called once per visible key. */
    children: ItemRenderProp<K>;
    /** Rendered in place of the items when the list is empty. */
    emptyContent?: ReactNode;
    /** Sizes the list to hold this many items before it scrolls. */
    displayItems?: number;
    /**
     * Smooths the height change when the item count changes. Set it only when the list
     * is sized by its content; a list sized by its container lags behind every resize.
     */
    animateHeight?: boolean;
}
declare const BaseItems: <K extends record.Key = record.Key, E extends record.Keyed<K> | undefined = record.Keyed<K>>({ className, children, emptyContent, displayItems, animateHeight, style, direction, x, y, ...rest }: ItemsProps<K>) => ReactElement;
/**
 * The scroll container for a {@link Frame}. It renders the visible items, handles
 * virtualization, and shows `emptyContent` when there are none.
 */
export declare const Items: typeof BaseItems;
export {};
//# sourceMappingURL=Items.d.ts.map