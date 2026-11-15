// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/menu/Dialog.css";

import { type ReactNode, type Ref, use } from "react";

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

/** Props that give a dialog the surface of a menu. */
export type SurfaceProps = Required<
  Pick<
    Base.DialogProps,
    "ref" | "className" | "background" | "borderColor" | "rounded" | "pack"
  >
>;

/**
 * @returns props that give a dialog the surface of a menu. In a {@link Sub}, the dialog
 * opens beside the item of the submenu, with its first item level with it.
 * @param ref - The dialog's own ref.
 * @param className - The dialog's own class.
 */
export const useSurface = (
  ref?: Ref<HTMLDivElement>,
  className?: string,
): SurfaceProps => {
  const sub = use(SubContext);
  const combinedRef = useCombinedRefs(ref, sub?.content);
  return {
    ref: combinedRef,
    background: 2,
    borderColor: 6,
    rounded: "large",
    pack: false,
    className: CSS.cls(
      CSS.B("menu-dialog"),
      sub != null && CSS.BM("menu-dialog", "sub"),
      sub?.swap === true && CSS.BM("menu-dialog", "swap"),
      className,
    ),
  };
};

// The surface of an open menu that filters its items. It mounts only while open.
const Filtering = ({
  ref,
  onKeyDown,
  children,
  ...rest
}: Base.DialogProps): ReactNode => {
  const filter = useFilter();
  const combinedRef = useCombinedRefs(ref, filter.ref);
  return (
    <Base.Dialog
      ref={combinedRef}
      onKeyDown={(e) => {
        onKeyDown?.(e);
        filter.onKeyDown(e);
      }}
      {...rest}
    >
      <Filtered term={filter.term}>{children}</Filtered>
    </Base.Dialog>
  );
};

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
  ...rest
}: DialogProps): ReactNode => {
  const sub = use(SubContext);
  const { visible } = Base.useContext();
  const props: Base.DialogProps = { ...useSurface(ref, className), ...rest };
  const content = <SubContext value={undefined}>{children}</SubContext>;
  if (!filterable || sub != null)
    return <Base.Dialog {...props}>{content}</Base.Dialog>;
  if (!visible) return null;
  return <Filtering {...props}>{content}</Filtering>;
};
