// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/menu/Dialog.css";

import { type ReactNode, use, useEffect, useRef } from "react";

import { CSS } from "@/css";
import { Dialog as Base } from "@/dialog";
import { useCombinedRefs } from "@/hooks";
import { Filtered, useFilter } from "@/menu/filter";
import { SubContext } from "@/menu/submenus";

/** Props for {@link Dialog}. */
export interface DialogProps extends Base.DialogProps {
  /**
   * Takes the focus when it opens, moves it over the items with the arrow keys, and
   * filters the items as the user types, as a context menu does. A submenu ignores it.
   */
  filterable?: boolean;
}

/**
 * A menu that opens in a dialog, on the surface of a context menu. It holds menu items,
 * or the search and options of a selection: the search is a line of text over a
 * hairline, and the options are menu items. As the content of a {@link Sub}, it opens
 * beside the menu with its first item level with the item of the submenu.
 *
 * @example
 * <Dialog.Frame>
 *   <Dialog.Trigger>Filter</Dialog.Trigger>
 *   <Menu.Dialog>
 *     <Menu.Menu>{items}</Menu.Menu>
 *   </Menu.Dialog>
 * </Dialog.Frame>
 */
export const Dialog = ({
  ref,
  className,
  children,
  filterable = false,
  onKeyDown,
  ...rest
}: DialogProps): ReactNode => {
  const sub = use(SubContext);
  const { visible } = Base.useContext();
  const keyed = filterable && sub == null;
  const filter = useFilter(visible);
  const surfaceRef = useRef<HTMLDivElement>(null);
  const combinedRef = useCombinedRefs(ref, sub?.content, surfaceRef);
  useEffect(() => {
    if (keyed && visible) surfaceRef.current?.focus({ preventScroll: true });
  }, [keyed, visible]);
  return (
    <Base.Dialog
      ref={combinedRef}
      background={2}
      borderColor={6}
      rounded="large"
      pack={false}
      className={CSS.cls(
        CSS.B("menu-dialog"),
        sub != null && CSS.BM("menu-dialog", "sub"),
        sub?.swap === true && CSS.BM("menu-dialog", "swap"),
        className,
      )}
      onKeyDown={(e) => {
        onKeyDown?.(e);
        if (keyed) filter.onKeyDown(e);
      }}
      {...rest}
    >
      <SubContext value={undefined}>
        <Filtered term={keyed ? filter.term : ""}>{children}</Filtered>
      </SubContext>
    </Base.Dialog>
  );
};
