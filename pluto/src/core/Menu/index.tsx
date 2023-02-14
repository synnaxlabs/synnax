// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ContextMenu, useContextMenu } from "./ContextMenu";
import { Menu as CoreMenu } from "./Menu";
import { MenuItem, MenuItemIcon } from "./MenuItem";
export type {
  ContextMenuProps,
  UseContextMenuReturn,
  ContextMenuMenuProps,
} from "./ContextMenu";
export type { MenuProps } from "./Menu";
export type { MenuItemProps, MenuItemIconProps } from "./MenuItem";

type CoreMenuType = typeof CoreMenu;

export interface MenuType extends CoreMenuType {
  /**
   * Menu.ContextMenu wraps a set of children with a context menu. When the user right
   * clicks within wrapped area, the provided menu will be shown. Menu.ContextMenu should
   * be used in conjunction with the Menu.useContextMenu hook.
   *
   * The rendered menu is provided with a set of keys that represent the HTML IDs of the
   * context target elements. The first target is evaluated by traversing the parents
   * of the element that was right clicked until an element with the class "pluto-context-target"
   * is found. If no such element is found, the right clicked element itself is used as
   * the target. Subsequent targets are found by querying all siblings of the first target
   * that have the "pluto-context-selected" class.
   *
   * @example <caption>Example DOM structure</caption>
   *   <div id="pluto-menu-context__container">
   *    <div className="pluto-context-target" id="1">
   *      <span>
   *        <h2>I was right clicked!</h2>
   *      </span>
   *    </div>
   *    <div className="pluto-context-target pluto-context-selected" id="2">
   *    <div className="pluto-context-target" id="3">
   *   </div>
   *
   * In the above example, the keys provided to the menu would be ["1", "2"].
   *
   * The target resolution logic is ideal for both single and multi-select scenarios,
   * such as lists that have several selected rows that should be acted upon together.
   *
   * @param props - Props for the component. Expects all return values from the
   * useContextMenu hook. All non-hook and unlisted props will be spread to the
   * underlying div component acting as the root element.
   * @param props.menu - The menu to show when the user right clicks.
   */
  ContextMenu: typeof ContextMenu;
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
  useContextMenu: typeof useContextMenu;
  /**
   * Menu.Item renders a menu item.
   *
   * @param props - Props for the component. Identical props to those of Button except
   * for the ones listed below.
   * @param props.itemKey - The key of the item. This is used to identify the item and
   * is passed to the onChange callback of the Menu.
   */
  Item: typeof MenuItem;
  /**
   * Menu.ItemIcon renders a menu item with only an icon.
   *
   * @param props - Props for the component. Identical props to those of Button.Icon except
   * for the ones listed below.
   * @param props.itemKey - The key of the item. This is used to identify the item and
   *   is passed to the onChange callback of the Menu.
   */
  ItemIcon: typeof MenuItemIcon;
}

/**
 * Menu is a modular component that allows you to create a menu with a list of items.
 * It satisfies the InputControl string interface, so it's selected value can be
 * controlled.
 *
 * @param props - Props for the component. All unlisted props will be spread to the
 * underlying Space component acting as the root element.
 * @param props.onChange - Callback executed when the selected item changes.
 * @param props.value - The selected item.
 */
export const Menu = CoreMenu as MenuType;

Menu.ContextMenu = ContextMenu;
Menu.useContextMenu = useContextMenu;
Menu.Item = MenuItem;
Menu.ItemIcon = MenuItemIcon;
