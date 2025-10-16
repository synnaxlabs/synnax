// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/context-menu/ContextMenu.css";

import { xy } from "@synnaxlabs/x";
import { type ReactNode } from "react";
import { createPortal } from "react-dom";

import { type RenderProp } from "@/component/renderProp";
import { CSS_CLASS } from "@/context-menu/types";
import { type State, type UseReturn } from "@/context-menu/use";
import { CSS } from "@/css";
import { Flex } from "@/flex";

export interface MenuProps extends State {
  keys: string[];
}

export interface ContextMenuProps
  extends Omit<UseReturn, "className">,
    Omit<Flex.BoxProps, "ref"> {
  menu?: RenderProp<MenuProps>;
}

const Internal = ({
  ref,
  menu,
  visible,
  open,
  close,
  position,
  keys,
  className,
  cursor,
  style,
  onClick,
  ...rest
}: ContextMenuProps): ReactNode | null => {
  if (!visible) return null;
  return createPortal(
    <Flex.Box
      className={CSS(CSS_CLASS, CSS.bordered())}
      ref={ref}
      style={{ ...xy.css(position), ...style }}
      onClick={(e) => {
        close();
        onClick?.(e);
      }}
      gap="tiny"
      {...rest}
    >
      {menu?.({ keys, visible, position, cursor })}
    </Flex.Box>,
    document.body,
  );
};

/**
 * Menu.ContextMenu wraps a set of children with a context menu. When the user right
 * clicks within wrapped area, the provided menu will be shown. Menu.ContextMenu should
 * be used in conjunction with the Menu.useContextMenu hook.
 *
 * The rendered menu is provided with a set of keys that represent the HTML IDs of the
 * context target elements. The first target is evaluated by traversing the parents of
 * the element that was right clicked until an element with the class
 * "pluto-context-target" is found. If no such element is found, the right clicked
 * element itself is used as the target. If this target has the class
 * "pluto-context-selected", then subsequent targets are found by querying all siblings
 * of the first target that have the "pluto-context-selected" class. Otherwise, the only
 * key is the first target.
 *
 * @example <caption>Example DOM structure</caption>
 *   <div id="pluto-context-menu__container">
 *    <div className="pluto-context-target" id="1">
 *      <span>
 *        <h2>I was right clicked!</h2>
 *      </span>
 *    </div>
 *    <div className="pluto-context-target pluto-context-selected" id="2">
 *    <div className="pluto-context-target" id="3">
 *   </div>
 *
 * In the above example, the keys provided to the menu would be ["1"].
 *
 * If the <div> element with id="1" had a className of "pluto-context-target
 * pluto-context-selected" instead, the keys provided would be ["1", "2"].
 *
 * The target resolution logic is ideal for both single and multi-select scenarios, such
 * as lists that have several selected rows that should be acted upon together.
 *
 * @param props - Props for the component. Expects all return values from the
 * useContextMenu hook. All non-hook and unlisted props will be spread to the underlying
 * div component acting as the root element.
 * @param props.menu - The menu to show when the user right clicks.
 */
export const ContextMenu = ({
  menu,
  children,
  ...rest
}: ContextMenuProps): ReactNode => (
  <>
    <Internal menu={menu} {...rest} />
    {children}
  </>
);
