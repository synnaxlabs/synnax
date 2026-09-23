import "./Item.css";
import { type record } from "@synnaxlabs/x";
import { type MouseEvent, type MouseEventHandler, type ReactElement } from "react";
import { Button } from "../button";
import { type RenderProp } from "../component/renderProp";
export interface ItemRenderProps<K extends record.Key = record.Key> {
    index: number;
    key: K;
    itemKey: K;
    className?: string;
    translate?: number;
}
export type ItemProps<K extends record.Key, E extends Button.ElementType = "div"> = Omit<Button.ButtonProps<E>, "key" | "onSelect" | "translate" | "onClick"> & ItemRenderProps<K> & {
    draggingOver?: boolean;
    rightAligned?: boolean;
    onClick?: MouseEventHandler<HTMLElement>;
    onSelect?: (key: K, e: MouseEvent<HTMLElement>) => void;
    selected?: boolean;
    hovered?: boolean;
};
export type ItemRenderProp<K extends record.Key> = RenderProp<ItemRenderProps<K>>;
/**
 * itemNameID returns the DOM id for editable name text rendered inside the list item
 * with the given key. Item assigns the raw key as the row element's own id, so nested
 * editable text (e.g. Text.edit rename targets) must carry this derived id to keep
 * DOM ids unique.
 */
export declare const itemNameID: (itemKey: record.Key) => string;
export declare const Item: <K extends record.Key, E extends Button.ElementType = "div">({ itemKey, className, index, el, draggingOver, rightAligned, selected, translate, onSelect, onClick, hovered, style, role, ...rest }: ItemProps<K, E>) => ReactElement;
//# sourceMappingURL=Item.d.ts.map