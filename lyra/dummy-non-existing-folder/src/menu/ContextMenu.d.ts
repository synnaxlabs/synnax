import "./ContextMenu.css";
import { xy } from "@synnaxlabs/x";
import { type ReactNode, type RefCallback } from "react";
import { type RenderProp } from "../component/renderProp";
import { Flex } from "../flex";
interface ContextMenuState {
    visible: boolean;
    keys: string[];
    position: xy.XY;
    cursor: xy.XY;
}
/** Supported event types for triggering a context menu. */
export type ContextMenuEvent = xy.Client & {
    preventDefault: () => void;
    stopPropagation: () => void;
    target: Element;
};
export interface ContextMenuOpen {
    (e: xy.Crude | ContextMenuEvent): void;
}
/** Return value for the {@Menu.useContextMenu} hook. */
export interface UseContextMenuReturn extends ContextMenuState {
    visible: boolean;
    close: () => void;
    open: ContextMenuOpen;
    ref: RefCallback<HTMLDivElement>;
    className: string;
}
/**
 * Menu.useContextMenu extracts the logic for toggling a context menu, allowing
 * the caller to control the menu's visibility and position.
 *
 * @returns visible - Whether the menu is visible.
 * @returns close - A function to close the menu.
 * @returns open - A function to open the menu. The function accepts an XY coordinate and
 * an optional set of keys to set as the selected menu items. It's important to note
 * that these keys override the default behavior of the menu, which is explained in
 * the documentation for {@link Menu.ContextMenu}.
 */
export declare const useContextMenu: () => UseContextMenuReturn;
export interface ContextMenuMenuProps extends ContextMenuState {
    keys: string[];
}
export interface ContextMenuProps extends Omit<UseContextMenuReturn, "className">, Omit<Flex.BoxProps, "ref"> {
    menu?: RenderProp<ContextMenuMenuProps>;
}
/**
 * Menu.ContextMenu wraps a set of children with a context menu. When the user
 * right clicks within wrapped area, the provided menu will be shown.
 * Menu.ContextMenu should be used in conjunction with the Menu.useContextMenu
 * hook.
 *
 * The rendered menu is provided with a set of keys that identify the context
 * target elements. A target's key is its `data-menu-key` attribute when present,
 * otherwise its HTML id. Set `data-menu-key` when the element's id is reserved for
 * another purpose (e.g. a tab whose id encodes ARIA linking). The first target is
 * evaluated by traversing the parents of the element that was right clicked until
 * an element with the class "pluto-context__target" is found. If no such element is
 * found, the right clicked element itself is used as the target. If this target has
 * the class "pluto-context--selected", then subsequent targets are found by querying
 * all siblings of the first target that have the "pluto-context--selected" class.
 * Otherwise, the only key is the first target.
 *
 * While the menu is visible, the target the menu opened over (not the wider
 * selection) carries the `data-context-menu-open` attribute so styles can keep
 * it highlighted.
 *
 * @example <caption>Example DOM structure</caption>
 *   <div id="pluto-menu-context__container">
 *    <div className="pluto-context__target" id="1">
 *      <span>
 *        <h2>I was right clicked!</h2>
 *      </span>
 *    </div>
 *    <div className="pluto-context__target pluto-context--selected" id="2">
 *    <div className="pluto-context__target" id="3">
 *   </div>
 *
 * In the above example, the keys provided to the menu would be ["1"].
 *
 * If the <div> element with id="1" had a className of "pluto-context__target
 * pluto-context--selected" instead, the keys provided would be ["1", "2"].
 *
 * The target resolution logic is ideal for both single and multi-select
 * scenarios, such as lists that have several selected rows that should be acted
 * upon together.
 *
 * @param props - Props for the component. Expects all return values from the
 * useContextMenu hook. All non-hook and unlisted props will be spread to the
 * underlying div component acting as the root element.
 * @param props.menu - The menu to show when the user right clicks.
 */
export declare const ContextMenu: ({ menu, children, ...rest }: ContextMenuProps) => ReactNode;
export {};
//# sourceMappingURL=ContextMenu.d.ts.map