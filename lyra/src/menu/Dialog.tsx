// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/menu/Dialog.css";

import { type ReactNode, use } from "react";

import { CSS } from "@/css";
import { Dialog as Base } from "@/dialog";
import { useCombinedRefs } from "@/hooks";
import { SubContext } from "@/menu/submenus";

/** Props for {@link Dialog}. */
export interface DialogProps extends Base.DialogProps {}

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
  ...rest
}: DialogProps): ReactNode => {
  const sub = use(SubContext);
  const combinedRef = useCombinedRefs(ref, sub?.content);
  return (
    <Base.Dialog
      ref={combinedRef}
      background={false}
      borderColor={6}
      rounded="large"
      pack={false}
      className={CSS.cls(
        CSS.B("menu-dialog"),
        sub != null && CSS.BM("menu-dialog", "sub"),
        sub?.swap === true && CSS.BM("menu-dialog", "swap"),
        className,
      )}
      {...rest}
    >
      <SubContext value={undefined}>{children}</SubContext>
    </Base.Dialog>
  );
};
