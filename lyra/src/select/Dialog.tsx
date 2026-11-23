// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/select/Dialog.css";

import { array, destructor } from "@synnaxlabs/x";
import {
  type MouseEvent,
  type PropsWithChildren,
  type ReactNode,
  useEffect,
  useLayoutEffect,
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
import { menuKeyDown, menuKeyUp, openContextMenu } from "@/menu/keyboard";
import { CONTEXT_TARGET } from "@/menu/types";
import { useContext } from "@/select/Context";
import { onFirstInput } from "@/select/firstInput";
import { useRegistryContext } from "@/select/registry";
import { ClosedContext, InlineContext, useViewingContext } from "@/select/scope";
import { useGetElement, useOrder } from "@/select/use";

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
  /**
   * Whether the option with the key opens `menu`, asked when the menu would open. A
   * refused option leaves the right click and the keys to the platform. Defaults to
   * every option. A pinned option, such as one in the footer, never opens `menu`.
   */
  hasMenu?: (key: string) => boolean;
}

const subscribe = (): destructor.Destructor => destructor.NOOP;
const getSnapshot = (): boolean => true;
const getServerSnapshot = (): boolean => false;

const Closed = ({ children }: PropsWithChildren): ReactNode => {
  const [detached] = useState(() => document.createElement("div"));
  return createPortal(<ClosedContext value>{children}</ClosedContext>, detached);
};

// The dialog opens on its first choice, as a native dropdown does. A fetched list can
// load the choice after the dialog is placed, so the reveal waits for it once, until
// the user's first input. The list never moves under the user.
const useRevealChoice = (placed: boolean): void => {
  const registry = useRegistryContext("Select.Dialog");
  const order = useOrder();
  const getElement = useGetElement();
  const { scrollToIndex, getData } = List.useUtilContext();
  const { getState } = useContext();
  // Read through a ref, so the reveal runs once per open as these change identity.
  const readRef = useSyncedRef({ order, getElement, scrollToIndex, getData, getState });
  useLayoutEffect(() => {
    if (!placed) return;
    const { order, getElement, scrollToIndex, getData, getState } = readRef.current;
    const selected = array.toArray(getState().value);
    const stops: destructor.Destructor[] = [];
    const stop = (): void => stops.splice(0).forEach((s) => s());
    const tryReveal = (): void => {
      // A search lists its matches in their own order.
      if (registry.getTerm() !== "") return stop();
      const key = order.getOrder().find((k) => selected.includes(k));
      if (key == null) return;
      stop();
      const element = getElement(key);
      if (element != null) element.scrollIntoView({ block: "nearest" });
      else scrollToIndex(getData().indexOf(key));
    };
    stops.push(onFirstInput(registry, stop), order.subscribe(tryReveal));
    tryReveal();
    return stop;
  }, [placed, registry, readRef]);
};

interface ViewsProps extends PropsWithChildren, Pick<DialogProps, "menu" | "hasMenu"> {
  /** Whether the dialog is open. Only an open dialog has an option menu. */
  open: boolean;
  /** Packs the views as the dialog packs, so the options join as they would in it. */
  pack: boolean;
}

interface MenuViewsProps extends Omit<ViewsProps, "open"> {
  menu: NonNullable<DialogProps["menu"]>;
}

const optionKey = (option: HTMLElement): string => option.dataset.menuKey ?? option.id;

// Like the hover's Enter, the menu keys act anywhere in the window while the dialog is
// open.
const MenuViews = ({ menu, hasMenu, pack, children }: MenuViewsProps): ReactNode => {
  const setViewing = useViewingContext("Select.Dialog");
  const [viewing, setLocalViewing] = useState(false);
  const contextMenu = Menu.useContextMenu();
  const { getState } = useContext();
  const getElement = useGetElement();
  const registry = useRegistryContext("Select.Dialog");
  const opens = (option: HTMLElement): boolean =>
    !registry.isPinned(option) && (hasMenu?.(optionKey(option)) ?? true);
  // The hovered option, when it opens a menu.
  const targetRef = useSyncedRef((): HTMLElement | undefined => {
    if (contextMenu.visible) return undefined;
    const { hover } = getState();
    const hovered = hover == null ? undefined : getElement(hover);
    return hovered != null && opens(hovered) ? hovered : undefined;
  });
  useEffect(() => {
    if (viewing) return;
    const handleKeyDown = (e: KeyboardEvent): void => {
      const target = targetRef.current();
      if (target != null) menuKeyDown(e, () => openContextMenu(target));
    };
    const handleKeyUp = (e: KeyboardEvent): void => {
      const target = targetRef.current();
      if (target != null) menuKeyUp(e, () => openContextMenu(target));
    };
    window.addEventListener("keydown", handleKeyDown);
    window.addEventListener("keyup", handleKeyUp);
    return () => {
      window.removeEventListener("keydown", handleKeyDown);
      window.removeEventListener("keyup", handleKeyUp);
    };
  }, [viewing]);
  const handleContextMenu = (e: MouseEvent<HTMLDivElement>): void => {
    if (viewing || !(e.target instanceof Element)) return;
    const option = e.target.closest<HTMLElement>(`.${CONTEXT_TARGET}`);
    if (option != null && e.currentTarget.contains(option) && opens(option))
      contextMenu.open(e);
  };
  // Inside the views, so that a menu item can open a view in place of the options.
  return (
    <BaseDialog.Views
      pack={pack}
      onViewingChange={(next) => {
        setLocalViewing(next);
        setViewing(next);
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

const Views = ({ menu, hasMenu, open, pack, children }: ViewsProps): ReactNode => {
  const setViewing = useViewingContext("Select.Dialog");
  if (menu != null && open)
    return (
      <MenuViews menu={menu} hasMenu={hasMenu} pack={pack}>
        {children}
      </MenuViews>
    );
  return (
    <BaseDialog.Views pack={pack} onViewingChange={setViewing}>
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
  hasMenu,
  ...rest
}: DialogProps): ReactNode => {
  const { visible, placed } = BaseDialog.useContext();
  useRevealChoice(placed);
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
    <Views
      menu={menu}
      hasMenu={hasMenu}
      open={visible && !nested}
      pack={rest.pack ?? (surface === "menu" ? menuSurface.pack : true)}
    >
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
