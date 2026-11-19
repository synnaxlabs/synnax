// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/select/Dialog.css";

import { destructor } from "@synnaxlabs/x";
import {
  type MouseEvent,
  type PropsWithChildren,
  type ReactNode,
  useEffect,
  useState,
  useSyncExternalStore,
} from "react";
import { createPortal } from "react-dom";

import { CSS } from "@/css";
import { Dialog as BaseDialog } from "@/dialog";
import { useSyncedRef } from "@/hooks";
import { Input } from "@/input";
import { List } from "@/list";
import { Menu } from "@/menu";
import { usePath } from "@/menu/filter";
import { keyboardMenuKey, openContextMenu } from "@/menu/keyboard";
import { CONTEXT_TARGET } from "@/menu/types";
import { useContext } from "@/select/Context";
import { ClosedContext, InlineContext, useViewingContext } from "@/select/scope";

/** Props for {@link Dialog}. */
export interface DialogProps extends BaseDialog.DialogProps {
  /**
   * The surface the dialog draws. A menu surface looks like a `Menu.Dialog`: its search
   * spans the top, and its options take the size of menu items. It does not filter as
   * the user types.
   */
  surface?: "dialog" | "menu";
  /**
   * The options that match the filter of the menu show in the menu, after the name of
   * the submenu, while the menu filters by text that the name does not match. Only a
   * select in a `Menu.Sub` uses it.
   */
  filtered?: boolean;
  /**
   * The context menu of an option. A right click on an option opens it, and so do
   * Shift+F10 and the Menu key while an option holds the hover. It gets the key of that
   * option only, even when other options are selected.
   */
  menu?: Menu.ContextMenuProps["menu"];
}

const subscribe = (): destructor.Destructor => destructor.NOOP;
const getSnapshot = (): boolean => true;
const getServerSnapshot = (): boolean => false;

const Closed = ({ children }: PropsWithChildren): ReactNode => {
  const [detached] = useState(() => document.createElement("div"));
  return createPortal(<ClosedContext value>{children}</ClosedContext>, detached);
};

interface ViewsProps extends PropsWithChildren, Pick<DialogProps, "menu"> {
  /** Whether the dialog is open, so that the menu keys act. */
  open: boolean;
}

interface MenuViewsProps extends ViewsProps {
  menu: NonNullable<DialogProps["menu"]>;
}

const optionKey = (option: HTMLElement): string => option.dataset.menuKey ?? option.id;

// Like the hover's Enter, the menu keys act anywhere in the window while the dialog is
// open.
const MenuViews = ({ menu, open, children }: MenuViewsProps): ReactNode => {
  const setViewing = useViewingContext();
  const [viewing, setLocalViewing] = useState(false);
  const contextMenu = Menu.useContextMenu();
  const { getState } = useContext();
  const getElement = List.useGetElement();
  const hoveredRef = useSyncedRef((): HTMLElement | undefined => {
    if (contextMenu.visible) return undefined;
    const { hover } = getState();
    return hover == null ? undefined : getElement(hover);
  });
  useEffect(() => {
    if (!open || viewing) return;
    const handleKeyDown = (e: KeyboardEvent): void => {
      const key = keyboardMenuKey(e);
      const hovered = key == null ? undefined : hoveredRef.current();
      if (hovered == null) return;
      e.preventDefault();
      if (key === "F10" && !e.repeat) openContextMenu(hovered);
    };
    const handleKeyUp = (e: KeyboardEvent): void => {
      if (keyboardMenuKey(e) !== "ContextMenu") return;
      const hovered = hoveredRef.current();
      if (hovered == null) return;
      e.preventDefault();
      openContextMenu(hovered);
    };
    window.addEventListener("keydown", handleKeyDown);
    window.addEventListener("keyup", handleKeyUp);
    return () => {
      window.removeEventListener("keydown", handleKeyDown);
      window.removeEventListener("keyup", handleKeyUp);
    };
  }, [open, viewing]);
  const handleContextMenu = (e: MouseEvent<HTMLDivElement>): void => {
    if (viewing || !(e.target instanceof Element)) return;
    const option = e.target.closest(`.${CONTEXT_TARGET}`);
    if (option != null && e.currentTarget.contains(option)) contextMenu.open(e);
  };
  // Inside the views, so that a menu item can open a view in place of the options.
  return (
    <BaseDialog.Views
      pack
      onViewingChange={(next) => {
        setLocalViewing(next);
        setViewing?.(next);
      }}
      onContextMenu={handleContextMenu}
    >
      <Menu.ContextMenu
        {...contextMenu}
        menu={(props) =>
          menu({
            ...props,
            keys: props.target == null ? [] : [optionKey(props.target)],
          })
        }
      >
        {children}
      </Menu.ContextMenu>
    </BaseDialog.Views>
  );
};

// Packed as the dialog is, so the options join as they would in the dialog itself.
const Views = ({ menu, open, children }: ViewsProps): ReactNode => {
  const setViewing = useViewingContext();
  if (menu != null)
    return (
      <MenuViews menu={menu} open={open}>
        {children}
      </MenuViews>
    );
  return (
    <BaseDialog.Views pack onViewingChange={setViewing}>
      {children}
    </BaseDialog.Views>
  );
};

/**
 * The floating surface of a selection. While closed it renders its children into an
 * element outside the page, so a selected fixed {@link Item} can still show its label
 * in the trigger. Its children can show a view in their place with `Dialog.useViews`,
 * except in a menu that shows the options inline.
 */
export const Dialog = ({
  ref,
  className,
  children,
  surface = "dialog",
  filtered = false,
  menu,
  ...rest
}: DialogProps): ReactNode => {
  const { visible } = BaseDialog.useContext();
  // Under a submenu that the menu filter passes, a selection shows its options in the
  // menu or nothing.
  const nested = usePath().length > 0;
  const cls = CSS.cls(CSS.BE("select", "dialog"), className);
  const menuSurface = Menu.useSurface(ref, cls);
  // False on the server and during hydration, since the server renders no portal.
  const hydrated = useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
  const options = <Input.ItemBoundary>{children}</Input.ItemBoundary>;
  if (nested && filtered) return <InlineContext value>{options}</InlineContext>;
  const content = (
    <Views menu={menu} open={visible && !nested}>
      {options}
    </Views>
  );
  if (!visible || nested) return hydrated && <Closed>{content}</Closed>;
  if (surface === "menu")
    return (
      <BaseDialog.Dialog {...menuSurface} {...rest}>
        {content}
      </BaseDialog.Dialog>
    );
  return (
    <BaseDialog.Dialog ref={ref} {...rest} className={cls} bordered={false}>
      {content}
    </BaseDialog.Dialog>
  );
};
